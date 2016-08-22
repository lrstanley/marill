package domfinder

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/Liamraystanley/marill/procfinder"
)

// webservers represents the list of nice-name processes that we should be checking
// configurations for.
var webservers = map[string]bool{
	"cpsrvd":  true,
	"httpd":   true,
	"apache":  true,
	"lshttpd": true,
	"nginx":   true,
}

func GetWebservers() (pl []*procfinder.Process, err error) {
	tmp, err := procfinder.GetProcs()

	if err != nil {
		return nil, err
	}

	for i := range tmp {
		// correction for cPanel proc names, as cPanel dynamically updates these based
		// on the state of cPanel (idling, SSL, etc)
		if strings.Contains(tmp[i].Name, "cpsrvd") {
			tmp[i].Name = "cpsrvd"
		}

		if webservers[tmp[i].Name] {
			pl = append(pl, tmp[i])
		}
	}

	if len(pl) == 0 {
		return nil, &NewErr{Code: ErrNoWebservers}
	}

	// check to see what's listening on ports 80/443, and check to see if we support 'em
	var stdports *procfinder.Process
	for i := range pl {
		if pl[i].Port == 80 || pl[i].Port == 443 {
			stdports = pl[i]
			break
		}
	}

	if !webservers[stdports.Name] {
		// assume whatever is listening on port 80/443 is something we don't support
		return nil, errors.New(fmt.Sprintf("Found process PID %s (%s) on port %d, which we don't support!", stdports.PID, stdports.Name, stdports.Port))
	}

	return pl, nil
}

func getWebserverMap(pl []*procfinder.Process) (mpl map[string]*procfinder.Process) {
	mpl = make(map[string]*procfinder.Process)
	for i := range pl {
		mpl[pl[i].Name] = pl[i]
	}

	return mpl
}

func GetMainWebserver(pl []*procfinder.Process) *procfinder.Process {
	for i := range pl {
		if pl[i].Port == 80 || pl[i].Port == 443 {
			return pl[i]
		}
	}

	return nil
}

// Domain represents a domain we should be checking, including the necessary data
// to fetch it, with the included host/port proxiable op, and public ip
type Domain struct {
	IP       string
	Port     string
	URL      *url.URL
	PublicIP string
}

// GetDomains represents all of the domains that the current webserver has virtual
// hosts for.
func GetDomains(pl []*procfinder.Process) (*procfinder.Process, []*Domain, Err) {
	// we want to get just one of the webservers, (or procs), to run our
	// domain pulling from. commonly httpd spawns multiple child processes
	// which we don't need to check each one.
	proc := GetMainWebserver(pl)
	mpl := getWebserverMap(pl)

	if proc == nil {
		return nil, nil, &NewErr{Code: ErrNoWebservers}
	}

	// check to see if there were any cPanel processes within the list
	if proc, ok := mpl["cpsrvd"]; ok {
		// assume cPanel based. we can crawl /var/cpanel/ for necessary data.
		domains, err := ReadCpanelVars()

		if err != nil {
			return nil, nil, UpgradeErr(err)
		}

		return proc, domains, nil
	}

	if proc.Name == "httpd" || proc.Name == "apache" || proc.Name == "lshttpd" {
		// assume apache based. should be able to use "-S" switch:
		// docs: http://httpd.apache.org/docs/current/vhosts/#directives
		output, err := exec.Command(proc.Exe, "-S").Output()
		out := string(output)

		if err != nil {
			return nil, nil, &NewErr{Code: ErrApacheFetchVhosts, value: err.Error()}
		}

		if !strings.Contains(out, "VirtualHost configuration") {
			return nil, nil, &NewErr{Code: ErrApacheInvalidVhosts, value: "binary: " + proc.Exe}
		}

		domains, err := ReadApacheVhosts(out)
		if err != nil {
			return nil, nil, UpgradeErr(err)
		}

		return proc, domains, nil
	}

	return nil, nil, &NewErr{Code: ErrNotImplemented, value: proc.Name}
}

const (
	kernHostname = "/proc/sys/kernel/hostname"
)

func getHostname() string {
	data, err := ioutil.ReadFile(kernHostname)

	if err != nil {
		return "unknown"
	}

	return strings.Replace(string(data), "\n", "", 1)
}

var reIP = regexp.MustCompile(`^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$`)

type cpanelVhost struct {
	Servername   string
	Serveralias  string
	Homedir      string
	User         string
	IP           string
	Documentroot string
	Port         string
}

func ReadCpanelVars() ([]*Domain, error) {
	cphosts, err := filepath.Glob("/var/cpanel/userdata/[a-z0-9_]*/*.*.cache")

	if err != nil {
		return nil, err
	}

	var domains []*Domain
	for i := range cphosts {
		raw, err := ioutil.ReadFile(cphosts[i])
		if err != nil {
			continue
		}

		vhost := &cpanelVhost{}

		err = json.Unmarshal(raw, &vhost)
		if err != nil {
			continue
		}

		if vhost.User == "nobody" || reIP.MatchString(vhost.Servername) {
			// assume it's an invalid user or it's an ip. we can ignore.
			continue
		}

		// actually get the cPanel user data
		cpuser, err := ioutil.ReadFile(fmt.Sprintf("/var/cpanel/users/%s", vhost.User))
		if err != nil {
			continue
		}

		if strings.Contains(string(cpuser), "SUSPENDED=1") {
			// assume they are suspended
			continue
		}

		domainURL, err := isDomainURL(vhost.Servername, vhost.Port)
		if err != nil {
			// assume the actual domain is invalid
			continue
		}

		domains = append(domains, &Domain{
			IP:   vhost.IP,
			Port: vhost.Port,
			URL:  domainURL,
		})

		for _, subvhost := range strings.Split(vhost.Serveralias, " ") {
			subURL, err := isDomainURL(subvhost, vhost.Port)
			if err != nil {
				// assume bad domain
				continue
			}

			domains = append(domains, &Domain{
				IP:   vhost.IP,
				Port: vhost.Port,
				URL:  subURL,
			})
		}
	}

	return domains, nil
}

// ReadApacheVhosts interprets and parses the "httpd -S" directive entries.
// docs: http://httpd.apache.org/docs/current/vhosts/#directives
func ReadApacheVhosts(raw string) ([]*Domain, error) {
	// some regex patterns to pull out data from the vhost results
	reVhostblock := regexp.MustCompile(`(?sm:^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\:\d{2,5} \s+is a NameVirtualHost)`)
	reStripvars := regexp.MustCompile(`(?ms:[\w-]+: .*$)`)
	reVhostipport := regexp.MustCompile(`^(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})\:(\d{2,5})\s+`)

	// save the original, in case we need it
	original := raw

	// we'll want to get the hostname to test against (e.g. we want to ignore hostname urls)
	hostname := getHostname()

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

			if len(domainPort) == 0 || len(domainName) == 0 || reIP.MatchString(domainName) || hostname == domainName {
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

	stripDups(&domains)

	return domains, nil
}

// stripDups strips all domains that have the same resulting URL
func stripDups(domains *[]*Domain) {
	var tmp []*Domain

	for _, dom := range *domains {
		isIn := false
		for _, other := range tmp {
			if dom.URL.String() == other.URL.String() {
				isIn = true
				break
			}
		}
		if !isIn {
			tmp = append(tmp, dom)
		}
	}

	*domains = tmp

	return
}

// isDomainURL should validate the data we are obtaining from the webservers to
// ensure it is a proper hostname and/or port (within reason. custom configs are
// custom)
func isDomainURL(host string, port string) (*url.URL, Err) {
	if port != "443" && port != "80" {
		host = fmt.Sprintf("%s:%s", host, port)
	}

	intport, err := strconv.Atoi(port)
	if err != nil {
		return nil, &NewErr{Code: ErrInvalidURL, value: fmt.Sprintf("%s (port: %s)", host, port)}
	}
	strport := strconv.Itoa(intport)
	if strport != port {
		return nil, &NewErr{Code: ErrInvalidURL, value: fmt.Sprintf("%s (port: %s)", host, port)}
	}

	// lets try and determine the scheme we need. Best solution would like be:
	//   - 443 -- https
	//   - anything else -- http
	var scheme string
	if port == "443" {
		scheme = "https://"
	} else {
		scheme = "http://"
	}
	host = scheme + host

	if strings.Contains(host, " ") {
		return nil, &NewErr{Code: ErrInvalidURL, value: fmt.Sprintf("%s (port: %s)", host, port)}
	}

	uri, err := url.Parse(host)

	if err != nil {
		return nil, &NewErr{Code: ErrInvalidURL, value: fmt.Sprintf("%s (port: %s)", host, port)}
	}

	return uri, nil
}
