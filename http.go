package main

import "net/http"

// CustomClient is the state for our custom http wrapper, which houses
// the needed data to be able to rewrite the outgoing request during
// redirects.
type CustomClient struct {
	URL  string
	IP   string
	Host string
}

func (c *CustomClient) redirectHandler(req *http.Request, via []*http.Request) error {
	req = c.requestWrap(req)

	// rewrite Referer (Referrer) if it exists, to have the proper hostname
	uri := via[len(via)-1].URL
	uri.Host = via[len(via)-1].Host
	req.Header.Set("Referer", uri.String())

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

	// assign the origin host to the host header value
	req.Host = c.Host

	// and overwrite the host used to make the connection
	if len(c.IP) > 0 {
		req.URL.Host = c.IP
	}

	return req
}

// getHandler wraps the standard net/http library, allowing us to spoof hostnames and IP addresses
func (c *CustomClient) getHandler() (*http.Response, error) {
	client := &http.Client{
		CheckRedirect: c.redirectHandler,
	}

	req, err := http.NewRequest("GET", c.URL, nil)

	if err != nil {
		return nil, err
	}

	c.Host = req.URL.Host

	req = c.requestWrap(req)

	// actually make the request here
	resp, err := client.Do(req)

	return resp, err
}

// Get wraps GetHandler -- easy interface for making get requests
func Get(url string, ip string) (*http.Response, error) {
	c := &CustomClient{URL: url, IP: ip}

	return c.getHandler()
}
