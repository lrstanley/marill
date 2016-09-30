// Author: Liam Stanley <me@liamstanley.io>
// Docs: https://marill.liam.sh/
// Repo: https://github.com/Liamraystanley/marill

package domfinder

import (
	"fmt"
	"log"
	"net/url"
	"os/exec"
	"strings"

	"github.com/Liamraystanley/marill/procfinder"
	"github.com/Liamraystanley/marill/utils"
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

// Domain represents a domain we should be checking, including the necessary data
// to fetch it, with the included host/port proxiable op, and public ip
type Domain struct {
	IP   string
	Port string
	URL  *url.URL
}

func (d *Domain) String() string {
	return fmt.Sprintf("<[%s]::%s:%s>", d.URL.String(), d.IP, d.Port)
}

// Finder represents the entire domain crawl process to find domains that the server
// is actually hosting.
type Finder struct {
	// Procs is a list of procs that should match up to a running webserver. these
	// should all have bound tcp (4) sockets.
	Procs []*procfinder.Process
	// MainProc represents the main process which we are pulling information from.
	MainProc *procfinder.Process
	// Domains represents the list of domains that are valid and should be crawled.
	Domains []*Domain
	// Log is a logger which we should dump debugging info to.
	Log *log.Logger
}

// DomainFilter filters Finder.Domains based on query input
type DomainFilter struct {
	IgnoreHTTP  bool   // ignore ^http urls
	IgnoreHTTPS bool   // ignore ^https urls
	IgnoreMatch string // ignore urls matching glob
	MatchOnly   string // ignore urls not matching glob
}

// Filter allows end users to filter out domains from the automated domain search
// functionality
func (f *Finder) Filter(cnf DomainFilter) {
	new := []*Domain{}

	for i := range f.Domains {
		if cnf.IgnoreHTTP && f.Domains[i].URL.Scheme == "http" {
			continue
		}
		if cnf.IgnoreHTTPS && f.Domains[i].URL.Scheme == "https" {
			continue
		}
		if len(cnf.IgnoreMatch) > 0 && (utils.Glob(f.Domains[i].URL.String(), cnf.IgnoreMatch) || utils.Glob(f.Domains[i].URL.Host, cnf.IgnoreMatch)) {
			continue
		}
		if len(cnf.MatchOnly) > 0 && !utils.Glob(f.Domains[i].URL.String(), cnf.MatchOnly) && !utils.Glob(f.Domains[i].URL.Host, cnf.MatchOnly) {
			continue
		}

		new = append(new, f.Domains[i])
	}

	f.Domains = new
}

// GetWebservers pulls only the web server processes from the process list on the
// server.
func (f *Finder) GetWebservers() (err error) {
	tmp, err := procfinder.GetProcs()

	if err != nil {
		return err
	}

	for i := range tmp {
		// correction for cPanel proc names, as cPanel dynamically updates these based
		// on the state of cPanel (idling, SSL, etc)
		if strings.Contains(tmp[i].Name, "cpsrvd") {
			tmp[i].Name = "cpsrvd"
		}

		if webservers[tmp[i].Name] {
			f.Procs = append(f.Procs, tmp[i])
		}
	}

	if len(f.Procs) == 0 {
		return &NewErr{Code: ErrNoWebservers}
	}

	// check to see what's listening on ports 80/443, and check to see if we support 'em
	var stdports *procfinder.Process
	for i := range f.Procs {
		if f.Procs[i].Port == 80 || f.Procs[i].Port == 443 {
			stdports = f.Procs[i]
			break
		}
	}

	if !webservers[stdports.Name] {
		// assume whatever is listening on port 80/443 is something we don't support
		return fmt.Errorf("found process PID %s (%s) on port %d, which we don't support", stdports.PID, stdports.Name, stdports.Port)
	}

	return nil
}

// getWebserverMap returns a map of processes based on a key map of the process
// names. useful to easily know if a process name is within the list
func (f *Finder) getWebserverMap() (mpl map[string]*procfinder.Process) {
	mpl = make(map[string]*procfinder.Process)
	for i := range f.Procs {
		mpl[f.Procs[i].Name] = f.Procs[i]
	}

	return mpl
}

// GetMainWebserver returns only one webserver which we should be pulling data
// from.
func (f *Finder) GetMainWebserver() {
	for i := range f.Procs {
		if f.Procs[i].Port == 80 || f.Procs[i].Port == 443 {
			f.MainProc = f.Procs[i]
			return
		}
	}

	f.MainProc = nil

	return
}

// GetDomains represents all of the domains that the current webserver has virtual
// hosts for.
func (f *Finder) GetDomains() Err {
	// we want to get just one of the webservers, (or procs), to run our
	// domain pulling from. commonly httpd spawns multiple child processes
	// which we don't need to check each one.
	mpl := f.getWebserverMap()

	if f.GetMainWebserver(); f.MainProc == nil {
		return &NewErr{Code: ErrNoWebservers}
	}

	// check to see if there were any cPanel processes within the list
	if proc, ok := mpl["cpsrvd"]; ok {
		f.MainProc = proc

		// assume cPanel based. we can crawl /var/cpanel/ for necessary data.
		if err := f.ReadCpanelVars(); err != nil {
			return UpgradeErr(err)
		}

		return nil
	}

	if f.MainProc.Name == "httpd" || f.MainProc.Name == "apache" || f.MainProc.Name == "lshttpd" {
		// assume apache based. should be able to use "-S" switch:
		// docs: http://httpd.apache.org/docs/current/vhosts/#directives
		output, err := exec.Command(f.MainProc.Exe, "-S").Output()
		out := string(output)

		if err != nil {
			return &NewErr{Code: ErrApacheFetchVhosts, value: err.Error()}
		}

		if !strings.Contains(out, "VirtualHost configuration") {
			return &NewErr{Code: ErrApacheInvalidVhosts, value: "binary: " + f.MainProc.Exe}
		}

		if err := f.ReadApacheVhosts(out); err != nil {
			return UpgradeErr(err)
		}

		return nil
	}

	return &NewErr{Code: ErrNotImplemented, value: f.MainProc.Name}
}
