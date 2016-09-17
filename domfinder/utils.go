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
	kernDomain   = "/proc/sys/kernel/domainname"
)

// getHostname returns the servers hostname which we should compare against webserver
// vhost entries.
func getHostname() string {
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

// IsDomainURL should validate the data we are obtaining from the webservers to
// ensure it is a proper hostname and/or port (within reason. custom configs are
// custom)
func IsDomainURL(host, port string) (*url.URL, Err) {
	var uri *url.URL
	var err error

	if !strings.HasPrefix(host, "http") {
		if port != "443" && port != "80" && port != "" {
			host = fmt.Sprintf("%s:%s", host, port)
		}
	}

	if port != "" {
		intport, err := strconv.Atoi(port)
		if err != nil {
			return nil, &NewErr{Code: ErrInvalidURL, value: fmt.Sprintf("%s (port: %s)", host, port)}
		}
		strport := strconv.Itoa(intport)
		if strport != port {
			return nil, &NewErr{Code: ErrInvalidURL, value: fmt.Sprintf("%s (port: %s)", host, port)}
		}
	}

	if strings.HasPrefix(host, "http") {
		uri, err = url.Parse(host)
		if err != nil {
			return nil, &NewErr{Code: ErrInvalidURL, value: fmt.Sprintf("%s (port: %s)", host, port)}
		}

		if strings.HasPrefix(host, "http") && port != "" {
			uri.Host = uri.Host + ":" + port
		}
	} else {
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

		uri, err = url.Parse(host)
		if err != nil {
			return nil, &NewErr{Code: ErrInvalidURL, value: fmt.Sprintf("%s (port: %s)", host, port)}
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
