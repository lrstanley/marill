package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"golang.org/x/net/html"
)

// Results -- struct returned by Crawl() to represent the entire crawl process
type Results struct {
	Resource
	Body      string
	Resources []*Resource
}

// Resource represents a single entity of many within a given crawl. These should
// only be of type css, js, jpg, png, etc (static resources).
type Resource struct {
	URL           string
	Error         error
	Code          int
	Proto         string
	ContentLength int64
	TLS           *tls.ConnectionState
}

// getSrc crawls the body of the Results page, yielding all img/script/link resources
// so they can later be fetched.
func getSrc(b io.ReadCloser, req *http.Request) (urls []string) {
	urls = []string{}

	z := html.NewTokenizer(b)

	for {
		tt := z.Next()

		switch {
		case tt == html.ErrorToken:
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
			var src string

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

			for _, a := range t.Attr {
				if a.Key == checkType {
					src = a.Val
					break
				}
			}

			if len(src) == 0 || strings.HasSuffix(src, "/") {
				continue
			}

			if len(req.URL.Path) == 0 {
				req.URL.Path = "/"
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
				continue
			}

			urls = append(urls, src)
		}
	}
}

// Crawl manages the fetching of the main resource, as well as all child resources,
// providing a Results struct containing the entire crawl data needed
func Crawl(url string) (res *Results) {
	res = &Results{}
	resp, err := http.Get(url)

	if err != nil {
		res.Error = err
		return
	}

	b := resp.Body
	defer b.Close()

	res.URL = url
	res.Code = resp.StatusCode
	res.Proto = resp.Proto
	res.ContentLength = resp.ContentLength
	res.TLS = resp.TLS

	urls := getSrc(b, resp.Request)

	bbytes, err := ioutil.ReadAll(b)
	if len(bbytes) != 0 {
		res.Body = string(bbytes[:])
	}

	fmt.Println(urls)

	// here we should fetch all of the other resources in a different goroutine

	return
}
