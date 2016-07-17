package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"golang.org/x/net/html"
)

// Helper function to pull the src attribute from a Token
func getSrc(t html.Token, attr string, req *http.Request) (string, error) {
	var src string

	for _, a := range t.Attr {
		if a.Key == attr {
			src = a.Val
			break
		}
	}

	if len(src) == 0 || strings.HasSuffix(src, "/") {
		return "", errors.New("No src attribute found")
	}

	if strings.HasPrefix(src, "./") {
		src = req.URL.Scheme + "://" + req.URL.Host + req.URL.Path + strings.SplitN(src, "./", 2)[1]
	}

	if strings.HasPrefix(src, "//") {
		src = req.URL.Scheme + ":" + src
	}

	if strings.HasPrefix(src, "/") {
		src = req.URL.Scheme + "://" + req.URL.Host + src
	}

	if !strings.HasPrefix(src, "http") {
		return "", errors.New("Not a valid URL")
	}

	return src, nil
}

// Extract all http** links from a given webpage
func crawl(url string, ch chan string, chFinished chan bool) {
	resp, err := http.Get(url)

	defer func() {
		// Notify that we're done after this function
		chFinished <- true
	}()

	if err != nil {
		fmt.Println("ERROR: Failed to crawl \"" + url + "\"")
		return
	}

	b := resp.Body
	defer b.Close() // close Body when the function returns

	z := html.NewTokenizer(b)

	for {
		tt := z.Next()

		switch {
		case tt == html.ErrorToken:
			// End of the document, we're done
			return
		case tt == html.StartTagToken:
			t := z.Token()

			allowed := map[string]string{
				"link":   "href",
				"script": "src",
				"img":    "src",
			}
			var isInAllowed bool
			var checkType string
			for key := range allowed {
				if t.Data == key {
					isInAllowed = true
					checkType = allowed[key]
					break
				}
			}

			if !isInAllowed {
				continue
			}

			// Extract the src value, if there is one
			src, err := getSrc(t, checkType, resp.Request)
			if err != nil {
				fmt.Println(src, err)
				continue
			}

			// Make sure the src begines in http**
			ch <- src
		}
	}
}

func main() {
	foundUrls := make(map[string]bool)
	seedUrls := os.Args[1:]

	// Channels
	chUrls := make(chan string)
	chFinished := make(chan bool)

	// Kick off the crawl process (concurrently)
	for _, url := range seedUrls {
		go crawl(url, chUrls, chFinished)
	}

	// Subscribe to both channels
	for c := 0; c < len(seedUrls); {
		select {
		case url := <-chUrls:
			foundUrls[url] = true
		case <-chFinished:
			c++
		}
	}

	fmt.Println("\nFound", len(foundUrls), "unique urls:")

	for url := range foundUrls {
		fmt.Println(" - " + url)
	}

	close(chUrls)
}
