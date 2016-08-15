package domfinder

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var webservers = map[string]bool{
	"httpd":   true,
	"apache":  true,
	"nginx":   true,
	"lshttpd": true,
}

type Process struct {
	PID  string
	Name string
	Exe  string
}

func GetProcs() (pl []*Process) {
	ps, _ := filepath.Glob("/proc/[0-9]*")

	for i := range ps {
		proc := &Process{}

		proc.PID = strings.Split(ps[i], "/")[2]

		// command name
		if data, err := ioutil.ReadFile(ps[i] + "/comm"); err != nil {
			continue
		} else {
			proc.Name = strings.Replace(string(data), "\n", "", 1)
		}

		if !webservers[proc.Name] {
			continue
		}

		// executable path
		if data, err := os.Readlink(ps[i] + "/exe"); err != nil {
			continue
		} else {
			proc.Exe = strings.Replace(string(data), "\n", "", 1)
		}

		pl = append(pl, proc)
	}

	return pl
}

type Domain struct {
	IP       string
	Port     string
	URL      *url.URL
	PublicIP string
}

func GetDomains(pl []*Process) (domains []*Domain, err error) {
	if len(pl) == 0 {
		return nil, &NewErr{Code: ErrNoWebservers}
	}

	// we want to get just one of the webservers, (or procs), to run our
	// domain pulling from. Commonly httpd spawns multiple child processes
	// which we don't need to check each one.

	proc := pl[0]

	if proc.Name == "httpd" || proc.Name == "apache" || proc.Name == "lshttpd" {
		// assume apache based. Should be able to use "-S" switch:
		// docs: http://httpd.apache.org/docs/current/vhosts/#directives
		output, err := exec.Command(proc.Exe, "-S").Output()
		out := string(output)

		if err != nil {
			return nil, &NewErr{Code: ErrApacheFetchVhosts, value: err.Error()}
		}

		if !strings.Contains(out, "VirtualHost configuration") {
			return nil, &NewErr{Code: ErrApacheInvalidVhosts, value: "binary: " + proc.Exe}
		}

		domains, err = ReadApacheVhosts(out)

		return domains, err
	}

	return nil, errors.New("Unimplemented webserver")
}

func ReadApacheVhosts(raw string) ([]*Domain, error) {
	// some regex patterns to pull out data from the vhost results
	reVhostblock := regexp.MustCompile(`(?sm:^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\:\d{2,5} \s+is a NameVirtualHost)`)
	reStripvars := regexp.MustCompile(`(?ms:[\w-]+: .*$)`)
	reVhostipport := regexp.MustCompile(`^(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})\:(\d{2,5})\s+`)

	// save the original, in case we need it
	original := raw

	var domains []*Domain

	// strip misc. variables from the end of the output, to prevent them from being added
	// into the vhost blocks. These could be used in the future, though. e.g:
	//   ServerRoot: "/etc/apache2"
	//   Main DocumentRoot: "/etc/apache2/htdocs"
	//   Main ErrorLog: "/etc/apache2/logs/error_log"
	//   Mutex mpm-accept: using_defaults
	//   Mutex rewrite-map: dir="/etc/apache2/run" mechanism=fcntl
	//   Mutex ssl-stapling-refresh: using_defaults
	//   Mutex ssl-stapling: using_defaults
	//   Mutex proxy: using_defaults
	//   Mutex ssl-cache: dir="/etc/apache2/run" mechanism=fcntl
	//   Mutex default: dir="/var/run/apache2/" mechanism=default
	//   PidFile: "/etc/apache2/run/httpd.pid"
	//   Define: DUMP_VHOSTS
	//   Define: DUMP_RUN_CFG
	//   User: name="nobody" id=99
	//   Group: name="nobody" id=99
	raw = reStripvars.ReplaceAllString(raw, "")

	// should give us [][]int, child [] consisting of start, and end index of each item.
	// with this, we should be able to loop through and get each vhost section
	indexes := reVhostblock.FindAllStringSubmatchIndex(raw, -1)

	results := make([]string, len(indexes))

	for i, index := range indexes {
		if i+1 == len(indexes) {
			// assume it's the last one, we can go to the end
			results[i] = raw[index[0] : len(raw)-1]
		} else {
			results[i] = raw[index[0] : indexes[i+1][0]-1]
		}
	}

	if len(results) == 0 {
		return nil, &NewErr{Code: ErrApacheNoEntries}
	}

	// now we should have a list of loaded virtual host blocks.
	for i, rvhost := range results {
		// we should probably get the line count just to be helpful
		line := strings.Count(original[0:indexes[i][0]], "\n")

		rawipport := reVhostipport.FindAllStringSubmatch(rvhost, -1)
		if len(rawipport) == 0 {
			return nil, &NewErr{Code: ErrApacheParseVhosts, value: fmt.Sprintf("line %s", line)}
		}

		ip := rawipport[0][1]
		port := rawipport[0][2]
		if len(ip) == 0 || len(port) == 0 {
			return nil, &NewErr{Code: ErrApacheParseVhosts, value: fmt.Sprintf("line %s, unable to determine ip/port", line)}
		}

		reNameVhost := regexp.MustCompile(`\s+ port (\d{2,5}) namevhost ([^ ]+)`)
		tmp := reNameVhost.FindAllStringSubmatch(rvhost, -1)

		if len(tmp) == 0 {
			// no vhost entries within the IP address -- or all aliases
			continue
		}

		for _, item := range tmp {
			domainPort := item[1]
			domainName := item[2]

			if len(domainPort) == 0 || len(domainName) == 0 {
				// assume that we didn't parse the string properly -- might add logs for debugging
				// in the future
				continue
			}

			// lets try and parse it into a URL
			domainURL, err := isDomainURL(domainName, domainPort)

			if err != nil {
				// assume they have an entry in apache that just simply isn't a valid
				// domain
				continue
			}

			dom := &Domain{
				IP:   ip,
				Port: domainPort,
				URL:  domainURL,
			}

			domains = append(domains, dom)
		}
	}

	return domains, nil
}

func isDomainURL(host string, port string) (*url.URL, error) {
	uri, err := url.Parse(host)

	if err != nil {
		return nil, err
	}

	// lets try and determine the scheme we need. Best solution would like be:
	//   - 443 -- https
	//   - anything else -- http
	if port == "443" {
		uri.Scheme = "https"
	} else {
		uri.Scheme = "http"
	}

	return uri, nil
}
