// Author: Liam Stanley <me@liamstanley.io>
// Docs: https://marill.liam.sh/
// Repo: https://github.com/Liamraystanley/marill

package utils

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// GetHost obtains the host value from a url or domain
func GetHost(uri string) (string, error) {
	host, err := url.Parse(uri)

	if err != nil {
		return "", fmt.Errorf("invalid uri: %s", uri)
	}

	return host.Host, nil
}

const (
	kernHostname = "/proc/sys/kernel/hostname"
	kernDomain   = "/proc/sys/kernel/domainname"
	stdPort      = "80"
	sslPort      = "443"
)

// GetHostname returns the servers hostname which we should compare against webserver
// vhost entries. Also includes domain.
func GetHostname() string {
	host, herr := ioutil.ReadFile(kernHostname)
	domain, derr := ioutil.ReadFile(kernDomain)
	if herr != nil || derr != nil {
		return "unknown"
	}

	if strings.Contains(string(domain), "none") {
		return strings.Replace(string(host), "\n", "", 1)
	}

	return strings.Replace(string(host), "\n", "", 1) + "." + strings.Replace(string(domain), "\n", "", 1)
}

// IsDomainURL should validate the data we are obtaining from the webservers to
// ensure it is a proper hostname and/or port (within reason. custom configs are
// custom)
func IsDomainURL(host, port string) (*url.URL, error) {
	var uri *url.URL
	var err error

	if !strings.HasPrefix(host, "http") {
		if port != sslPort && port != stdPort && port != "" {
			host = fmt.Sprintf("%s:%s", host, port)
		}
	}

	if port != "" {
		intport, err := strconv.Atoi(port)
		if err != nil {
			return nil, fmt.Errorf("the host/port pair %s (port: %s) is invalid", host, port)
		}
		strport := strconv.Itoa(intport)
		if strport != port {
			return nil, fmt.Errorf("the host/port pair %s (port: %s) is invalid", host, port)
		}
	}

	if strings.Contains(host, " ") {
		return nil, fmt.Errorf("the host/port pair %s (port: %s) is invalid", host, port)
	}

	if strings.HasPrefix(host, "http") {
		uri, err = url.Parse(host)
		if err != nil {
			return nil, fmt.Errorf("the host/port pair %s (port: %s) is invalid", host, port)
		}

		if port == sslPort {
			uri.Scheme = "https"
			port = ""
		} else if port == stdPort {
			uri.Scheme = "http"
			port = ""
		}

		if port != "" {
			uri.Host = uri.Host + ":" + port
		}
	} else {
		// lets try and determine the scheme we need. Best solution would like be:
		//   - 443 -- https
		//   - anything else -- http
		var scheme string
		if port == sslPort {
			scheme = "https://"
		} else {
			scheme = "http://"
		}
		host = scheme + host

		uri, err = url.Parse(host)
		if err != nil {
			return nil, fmt.Errorf("the host/port pair %s (port: %s) is invalid", host, port)
		}
	}

	return uri, nil
}

// MustURL is much like isDomainURL, however will panic on error (useful for tests).
func MustURL(host, port string) *url.URL {
	uri, err := IsDomainURL(host, port)
	if err != nil {
		panic(err)
	}

	return uri
}

// LookupIP returns the first IP address from the resolving A record
func LookupIP(host string) (string, error) {
	ips, err := net.LookupIP(host)
	if err != nil {
		return "", err
	}

	if len(ips) == 0 {
		return "", fmt.Errorf("no a record was found for host: %s", host)
	}

	// select the first IP address in the list
	return ips[0].String(), nil
}
