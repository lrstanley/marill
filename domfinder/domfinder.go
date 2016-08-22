package domfinder

import (
	"errors"
	"fmt"
	"net/url"
	"os/exec"
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
