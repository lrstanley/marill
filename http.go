package main

import "net/http"

// Get wraps the standard net/http library, allowing us to spoof hostnames and IP addresses
func Get(url string, ip string) (*http.Response, error) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		return nil, err
	}

	if len(ip) > 0 {
		req.Host = ip
	}

	// spoof useragent, as there are going to be sites/servers that are
	// setup to deny by a specific useragent string (or lack there of)
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/51.0.2704.79 Safari/537.36")

	resp, err := client.Do(req)

	return resp, err
}
