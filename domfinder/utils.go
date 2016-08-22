package domfinder

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"strconv"
	"strings"
)

const (
	kernHostname = "/proc/sys/kernel/hostname"
)

// getHostname returns the servers hostname which we should compare against webserver
// vhost entries.
func getHostname() string {
	data, err := ioutil.ReadFile(kernHostname)

	if err != nil {
		return "unknown"
	}

	return strings.Replace(string(data), "\n", "", 1)
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
