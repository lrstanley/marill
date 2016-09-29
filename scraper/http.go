package scraper

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Liamraystanley/marill/utils"
)

// CustomClient is the state for our custom http wrapper, which houses
// the needed data to be able to rewrite the outgoing request during
// redirects.
type CustomClient struct {
	URL       string
	Host      string
	ResultURL url.URL  // represents the url for the resulting request, without modifications
	OriginURL *url.URL // represents the url from the original request, without modifications
	ipmap     map[string]string
}

// CustomResponse is the wrapped response from http.Client.Do() which also
// includes a timer of how long the request took, and a few other minor
// extras.
type CustomResponse struct {
	*http.Response
	Time *utils.TimerResult
	URL  *url.URL
}

var reIP = regexp.MustCompile(`^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$`)

func (c *CustomClient) redirectHandler(req *http.Request, via []*http.Request) error {
	c.requestWrap(req)

	// rewrite Referer (Referrer) if it exists, to have the proper hostname
	uri := via[len(via)-1].URL
	uri.Host = via[len(via)-1].Host
	req.Header.Set("Referer", uri.String())

	if len(via) > 3 {
		// assume too many redirects
		return errors.New("too many redirects (3)")
	}

	if reIP.MatchString(req.Host) && req.Host != c.Host {
		return errors.New("redirected to IP that doesn't match proxy/origin request")
	}

	// check to see if we're redirecting to a target which is possibly off this server
	// or not in this session of crawls
	if _, ok := c.ipmap[req.Host]; !ok {
		if c.OriginURL.Path == "" {
			// it's not in as a host -> ip map, but let's check to see if it resolves to a target ip
			var isin bool
			for _, val := range c.ipmap {
				if req.Host == val || req.URL.Host == val {
					isin = true
					break
				}
			}

			if !isin {
				return errors.New("redirection does not match origin host")
			}
		}
	}

	return nil
}

func (c *CustomClient) requestWrap(req *http.Request) *http.Request {
	// spoof useragent, as there are going to be sites/servers that are
	// setup to deny by a specific useragent string (or lack there of)
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/51.0.2704.79 Safari/537.36")

	// if an IP address is provided, rewrite the Host headers
	// of note: if we plan to support custom ports, these should be rewritten
	// within the header. E.g. "hostname.com:8080" -- though, common ports like
	// 80 and 443 are left out.

	// assign the origin host to the host header value, ONLY if it matches the domains
	// hostname
	if ip, ok := c.ipmap[req.URL.Host]; ok {
		req.Host = req.URL.Host

		// and overwrite the host used to make the connection
		if len(ip) > 0 {
			req.URL.Host = ip
		}
	}

	// update our cached resulting uri
	c.ResultURL = *req.URL
	if len(req.Host) > 0 {
		c.ResultURL.Host = req.Host
	}

	return req
}

// HostnameError appears when an invalid SSL certificate is supplied
type HostnameError struct {
	Certificate *x509.Certificate
	Host        string
}

func (h HostnameError) Error() string {
	c := h.Certificate

	var valid string
	if ip := net.ParseIP(h.Host); ip != nil {
		// Trying to validate an IP
		if len(c.IPAddresses) == 0 {
			return "x509: cannot validate certificate for " + h.Host + " because it doesn't contain any IP SANs"
		}
		for _, san := range c.IPAddresses {
			if len(valid) > 0 {
				valid += ", "
			}
			valid += san.String()
		}
	} else {
		if len(c.DNSNames) > 0 {
			valid = strings.Join(c.DNSNames, ", ")
		} else {
			valid = c.Subject.CommonName
		}
	}
	return "x509: certificate is valid for " + valid + ", not " + h.Host
}

// toLowerCaseASCII returns a lower-case version of in. See RFC 6125 6.4.1. We use
// an explicitly ASCII function to avoid any sharp corners resulting from
// performing Unicode operations on DNS labels.
func toLowerCaseASCII(in string) string {
	// If the string is already lower-case then there's nothing to do.
	isAlreadyLowerCase := true
	for _, c := range in {
		if c == utf8.RuneError {
			// If we get a UTF-8 error then there might be
			// upper-case ASCII bytes in the invalid sequence.
			isAlreadyLowerCase = false
			break
		}
		if 'A' <= c && c <= 'Z' {
			isAlreadyLowerCase = false
			break
		}
	}

	if isAlreadyLowerCase {
		return in
	}

	out := []byte(in)
	for i, c := range out {
		if 'A' <= c && c <= 'Z' {
			out[i] += 'a' - 'A'
		}
	}
	return string(out)
}

func matchHostnames(pattern, host string) bool {
	host = strings.TrimSuffix(host, ".")
	pattern = strings.TrimSuffix(pattern, ".")

	if len(pattern) == 0 || len(host) == 0 {
		return false
	}

	patternParts := strings.Split(pattern, ".")
	hostParts := strings.Split(host, ".")

	if len(patternParts) != len(hostParts) {
		return false
	}

	for i, patternPart := range patternParts {
		if i == 0 && patternPart == "*" {
			continue
		}
		if patternPart != hostParts[i] {
			return false
		}
	}

	return true
}

// verifyx509 returns nil if c is a valid certificate for the named host.
// Otherwise it returns an error describing the mismatch.
func verifyx509(c *x509.Certificate, h string) error {
	// IP addresses may be written in [ ].
	candidateIP := h
	if len(h) >= 3 && h[0] == '[' && h[len(h)-1] == ']' {
		candidateIP = h[1 : len(h)-1]
	}
	if ip := net.ParseIP(candidateIP); ip != nil {
		// We only match IP addresses against IP SANs.
		// https://tools.ietf.org/html/rfc6125#appendix-B.2
		for _, candidate := range c.IPAddresses {
			if ip.Equal(candidate) {
				return nil
			}
		}
		return HostnameError{c, candidateIP}
	}

	lowered := toLowerCaseASCII(h)

	if len(c.DNSNames) > 0 {
		for _, match := range c.DNSNames {
			if matchHostnames(toLowerCaseASCII(match), lowered) {
				return nil
			}
		}
		// If Subject Alt Name is given, we ignore the common name.
	} else if matchHostnames(toLowerCaseASCII(c.Subject.CommonName), lowered) {
		return nil
	}

	return HostnameError{c, h}
}

// VerifyHostname verifies if the tls.ConnectionState certificate matches the hostname
func VerifyHostname(c *tls.ConnectionState, host string) error {
	if c == nil {
		return nil
	}

	return verifyx509(c.PeerCertificates[0], host)
}

// getHandler wraps the standard net/http library, allowing us to spoof hostnames and IP addresses
func (c *CustomClient) getHandler() (*CustomResponse, error) {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			// unfortunately, ServerName will not persist over a redirect. so... we have to ignore
			// ssl invalidations and do them somewhat manually.
			InsecureSkipVerify: true,
			ServerName:         c.Host,
		},
	}
	client := &http.Client{
		CheckRedirect: c.redirectHandler,
		Timeout:       time.Duration(10) * time.Second,
		Transport:     transport,
	}

	req, err := http.NewRequest("GET", c.URL, nil)

	if err != nil {
		return nil, err
	}

	c.OriginURL = req.URL // set origin url for use in redirect wrapper
	c.requestWrap(req)

	// start tracking how long the request is going to take
	timer := utils.NewTimer()

	// actually make the request here
	resp, err := client.Do(req)

	// stop tracking the request
	timer.End()

	if err == nil {
		if err = VerifyHostname(resp.TLS, c.ResultURL.Host); err != nil {
			return nil, err
		}
	}

	if len(c.ResultURL.Host) > 0 {
		return &CustomResponse{resp, timer.Result, &c.ResultURL}, err
	}

	return &CustomResponse{resp, timer.Result, req.URL}, err
}

// Get wraps GetHandler -- easy interface for making get requests
func (c *Crawler) Get(url string) (*CustomResponse, error) {
	host, err := utils.GetHost(url)
	if err != nil {
		return nil, err
	}

	if ip, ok := c.ipmap[host]; ok {
		if len(ip) > 0 && !reIP.MatchString(ip) {
			return nil, errors.New("IP address provided is invalid")
		}
	}

	cc := &CustomClient{URL: url, Host: host, ipmap: c.ipmap}

	return cc.getHandler()
}
