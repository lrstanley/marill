package scraper

import (
	"errors"
	"net/http"
	"strings"
	"time"
)

// CustomClient is the state for our custom http wrapper, which houses
// the needed data to be able to rewrite the outgoing request during
// redirects.
type CustomClient struct {
	URL  string
	IP   string
	Host string
}

// CustomResponse is the wrapped response from http.Client.Do() which also
// includes a timer of how long the request took, and a few other minor
// extras.
type CustomResponse struct {
	*http.Response
	Time *TimerResult
	URL  string
}

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
	if strings.ToLower(req.URL.Host) == strings.ToLower(c.Host) || strings.ToLower(req.URL.Host) == strings.ToLower("www."+c.Host) {
		req.Host = req.URL.Host

		// and overwrite the host used to make the connection
		if len(c.IP) > 0 {
			req.URL.Host = c.IP
		}
	}

	return req
}

// getHandler wraps the standard net/http library, allowing us to spoof hostnames and IP addresses
func (c *CustomClient) getHandler() (*CustomResponse, error) {
	client := &http.Client{
		CheckRedirect: c.redirectHandler,
		Timeout:       time.Duration(10) * time.Second,
	}

	req, err := http.NewRequest("GET", c.URL, nil)

	if err != nil {
		return nil, err
	}

	c.Host = req.URL.Host

	c.requestWrap(req)

	// start tracking how long the request is going to take
	timer := NewTimer()

	// actually make the request here
	resp, err := client.Do(req)

	// stop tracking the request
	timer.End()

	var url string

	if err == nil {
		url = resp.Request.URL.String()
	} else {
		url = req.URL.String()
	}

	wrappedResp := &CustomResponse{resp, timer.Result, url}

	return wrappedResp, err
}

// Get wraps GetHandler -- easy interface for making get requests
func Get(url string, ip string) (*CustomResponse, error) {
	c := &CustomClient{URL: url, IP: ip}

	return c.getHandler()
}
