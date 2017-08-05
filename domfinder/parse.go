// Author: Liam Stanley <me@liamstanley.io>
// Docs: https://marill.liam.sh/
// Repo: https://github.com/lrstanley/marill

package domfinder

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/lrstanley/marill/utils"
)

var reIP = regexp.MustCompile(`^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$`)

// cpanelVhost represents a /var/cpanel/userdata/<user>/<domain>.cache file.
type cpanelVhost struct {
	Servername   string
	Serveralias  string
	Homedir      string
	User         string
	IP           string
	Documentroot string
	Port         string
}

func (vhost *cpanelVhost) String() string {
	return fmt.Sprintf("<[cPanel dom] home:%q user:%q ip:%q docroot:%q port:%q name:%q alias:%q>", vhost.Homedir, vhost.User, vhost.IP, vhost.Documentroot, vhost.Port, vhost.Servername, vhost.Serveralias)
}

// ReadCpanelVars crawls through /var/cpanel/userdata/ and returns all valid
// domains/ports that the cPanel server is hosting
func (f *Finder) ReadCpanelVars() error {
	cphosts, err := filepath.Glob("/var/cpanel/userdata/[a-z0-9_]*/*.*.cache")

	if err != nil {
		return err
	}

	var suspended []string
	var domains []*Domain
	for i := range cphosts {
		raw, err := ioutil.ReadFile(cphosts[i])
		if err != nil {
			f.Log.Printf("unable to read file '%s' during domain search: %s", cphosts[i], err)
			continue
		}

		vhost := &cpanelVhost{}

		err = json.Unmarshal(raw, &vhost)
		if err != nil {
			f.Log.Printf("unable to parse json in file '%s' during domain search: %s", cphosts[i], err)
			continue
		}

		if vhost.User == "nobody" || reIP.MatchString(vhost.Servername) {
			// assume it's an invalid user or it's an ip. we can ignore.
			f.Log.Printf("found invalid vhost during domain search, skipping: %s", vhost)
			continue
		}

		// check to see if the user is suspended
		var isSuspended bool
		for s := 0; s < len(suspended); s++ {
			if vhost.User == suspended[s] {
				isSuspended = true
				break
			}
		}
		if isSuspended {
			f.Log.Printf("cPanel user '%s' is suspended. skipping vhost. (from %s)", vhost.User, vhost)
			continue // outright skip
		}

		// actually get the cPanel user data
		cpuser, err := ioutil.ReadFile(fmt.Sprintf("/var/cpanel/users/%s", vhost.User))
		if err != nil {
			f.Log.Printf("unable to read user file '%s' during domain search, skipping: %s", fmt.Sprintf("/var/cpanel/users/%s", vhost.User), vhost)
			continue
		}

		// TODO: This can be cached.
		if strings.Contains(string(cpuser), "SUSPENDED=1") {
			// assume they are suspended
			suspended = append(suspended, vhost.User)
			f.Log.Printf("cPanel user '%s' is suspended. skipping vhost. (from %s)", vhost.User, vhost)
			continue
		}

		domainURL, err := utils.IsDomainURL(vhost.Servername, vhost.Port)
		if err != nil {
			// assume the actual domain is invalid
			f.Log.Printf("invalid uri from cPanel vhost: %s", vhost)
			continue
		}

		domains = append(domains, &Domain{
			IP:   vhost.IP,
			Port: vhost.Port,
			URL:  domainURL,
		})

		for _, subvhost := range strings.Split(vhost.Serveralias, " ") {
			subURL, err := utils.IsDomainURL(subvhost, vhost.Port)
			if err != nil {
				// assume bad domain
				f.Log.Printf("invalid uri from cPanel vhost: %s (from: %q)", subvhost, strings.Split(vhost.Serveralias, " "))
				continue
			}

			domains = append(domains, &Domain{
				IP:   vhost.IP,
				Port: vhost.Port,
				URL:  subURL,
			})
		}
	}

	stripDups(&domains)
	stripPredefined(&domains)
	f.Domains = domains

	return nil
}

var reNameVhost = regexp.MustCompile(`\s+ port (\d{2,5}) namevhost ([^ ]+)`)

// ReadApacheVhosts interprets and parses the "httpd -S" directive entries.
// docs: http://httpd.apache.org/docs/current/vhosts/#directives
func (f *Finder) ReadApacheVhosts(raw string) error {
	// some regex patterns to pull out data from the vhost results
	reVhostblock := regexp.MustCompile(`(?sm:^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\:\d{2,5} \s+is a NameVirtualHost)`)
	reStripvars := regexp.MustCompile(`(?ms:[\w-]+: .*$)`)
	reVhostipport := regexp.MustCompile(`^(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})\:(\d{2,5})\s+`)

	// save the original, in case we need it
	original := raw

	// we'll want to get the hostname to test against (e.g. we want to ignore hostname urls)
	hostname := utils.GetHostname()

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
		return &NewErr{Code: ErrApacheNoEntries}
	}

	// now we should have a list of loaded virtual host blocks.
	for i, rvhost := range results {
		// we should probably get the line count just to be helpful
		line := strings.Count(original[0:indexes[i][0]], "\n")

		rawipport := reVhostipport.FindAllStringSubmatch(rvhost, -1)
		if len(rawipport) == 0 {
			return &NewErr{Code: ErrApacheParseVhosts, value: fmt.Sprintf("line %d", line)}
		}

		ip := rawipport[0][1]
		port := rawipport[0][2]
		if len(ip) == 0 || len(port) == 0 {
			return &NewErr{Code: ErrApacheParseVhosts, value: fmt.Sprintf("line %d, unable to determine ip/port", line)}
		}

		tmp := reNameVhost.FindAllStringSubmatch(rvhost, -1)

		if len(tmp) == 0 {
			// no vhost entries within the IP address -- or all aliases
			continue
		}

		for _, item := range tmp {
			domainPort := item[1]
			domainName := item[2]

			if len(domainPort) == 0 || len(domainName) == 0 || reIP.MatchString(domainName) || hostname == domainName {
				// assume that we didn't parse the string properly
				f.Log.Printf("unable to parse apache domain %s (port %s) during domain search", domainName, domainPort)
				continue
			}

			// lets try and parse it into a URL
			domainURL, err := utils.IsDomainURL(domainName, domainPort)

			if err != nil {
				// assume they have an entry in apache that just simply isn't a valid
				// domain
				f.Log.Printf("unable to parse apache domain %s (port %s): %s", domainName, domainPort, err)
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
	stripPredefined(&domains)
	f.Domains = domains

	return nil
}
