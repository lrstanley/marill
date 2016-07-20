package main

import "net/http"

// http://stackoverflow.com/questions/13263492/set-useragent-in-http-request
// https://godoc.org/net/http

func NewClient(url string, ip string) (*http.Request, error) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		return nil, err
	}

	// spoof useragent, as there are going to be sites/servers that are
	// setup to deny by a specific useragent string (or lack there of)
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/51.0.2704.79 Safari/537.36")

	return req, nil
}
