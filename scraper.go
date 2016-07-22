package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"

	"golang.org/x/net/html"
)

// Results -- struct returned by Crawl() to represent the entire crawl process
type Results struct {
	// Inherit the Resource struct
	Resource
	// Body represents a string implementation of the byte array returned by
	// http.Response
	Body string
	// Slice of Resource structs containing the needed resources for the given URL
	Resources []*Resource
}

// Resource represents a single entity of many within a given crawl. These should
// only be of type css, js, jpg, png, etc (static resources).
type Resource struct {
	// connURL is the initial URL received by input
	connURL string
	// connIP is the initial IP address received by input
	connIP string
	// URL represents the resulting static URL derived by the original result page
	URL string
	// Error represents any errors that may have occurred when fetching the resource
	Error error
	// Code represents the numeric HTTP based status code
	Code int
	// Proto represents the end protocol used to fetch the page. For example, HTTP/2.0
	Proto string
	// ContentLength represents the number of bytes in the body of the response
	ContentLength int64
	// TLS represents the SSL/TLS handshake/session if the resource was loaded over
	// SSL.
	TLS *tls.ConnectionState
}

var resourcePool sync.WaitGroup

// getSrc crawls the body of the Results page, yielding all img/script/link resources
// so they can later be fetched.
func getSrc(b io.ReadCloser, req *http.Request) (urls []string) {
	urls = []string{}

	z := html.NewTokenizer(b)

	for {
		// loop through all tokens in the html body response
		tt := z.Next()

		switch {
		case tt == html.ErrorToken:
			// this assumes that there are no further tokens -- end of document
			return
		case tt == html.StartTagToken:
			t := z.Token()

			// the tokens that we are pulling resources from, and the attribute we are
			// pulling from
			allowed := map[string]string{
				"link":   "href",
				"script": "src",
				"img":    "src",
			}
			var isInAllowed bool
			var checkType string
			var src string

			// loop through all allowed elements, and see if the current element is
			// allowed
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

			// this assumes that the resource is something along the lines of:
			//   http://something.com/ -- which we don't care about
			if len(src) == 0 || strings.HasSuffix(src, "/") {
				continue
			}

			// add trailing slash to the end of the path
			if len(req.URL.Path) == 0 {
				req.URL.Path = "/"
			}

			// site was developed using relative paths. E.g:
			//  - url: http://domain.com/sub/path and resource: ./something/main.js
			//    would equal http://domain.com/sub/path/something/main.js
			if strings.HasPrefix(src, "./") {
				src = req.URL.Scheme + "://" + req.URL.Host + req.URL.Path + strings.SplitN(src, "./", 2)[1]
			}

			// site is loading resources from a remote location that supports both
			// http and https. browsers should natively tack on the current sites
			// protocol to the url. E.g:
			//  - url: http://domain.com/ and resource: //other.com/some-resource.js
			//    generates: http://other.com/some-resource.js
			//  - url: https://domain.com/ and resource: //other.com/some-resource.js
			//    generates: https://other.com/some-resource.js
			if strings.HasPrefix(src, "//") {
				src = req.URL.Scheme + ":" + src
			}

			// non-host-absolute resource. E.g. resource is loaded based on the docroot
			// of the domain. E.g:
			//  - url: http://domain.com/ and resource: /some-resource.js
			//    generates: http://domain.com/some-resource.js
			//  - url: https://domain.com/sub/resource and resource: /some-resource.js
			//    generates: https://domain.com/some-resource.js
			if strings.HasPrefix(src, "/") {
				src = req.URL.Scheme + "://" + req.URL.Host + src
			}

			// ignore anything else that isn't http based. E.g. ftp://, and other svg-like
			// data urls, as we really can't fetch those.
			if !strings.HasPrefix(src, "http") {
				continue
			}

			urls = append(urls, src)
		}
	}
}

// FetchResource fetches a singular resource from a page, returning a *Resource struct.
// As we don't care much about the body of the resource, that can safely be ignored. We
// must still close the body object, however.
func (rsrc *Resource) FetchResource() {
	defer resourcePool.Done()
	resp, err := Get(rsrc.connURL, rsrc.connIP)
	resp.Body.Close()

	if err != nil {
		rsrc.Error = err
		return
	}

	rsrc.URL = resp.Request.URL.String()
	rsrc.Code = resp.StatusCode
	rsrc.Proto = resp.Proto
	rsrc.ContentLength = resp.ContentLength
	rsrc.TLS = resp.TLS

	fmt.Printf("[%d] [%s] %s\n", rsrc.Code, rsrc.Proto, rsrc.URL)

	return
}

// Crawl manages the fetching of the main resource, as well as all child resources,
// providing a Results struct containing the entire crawl data needed
func Crawl(url string, ip string) (res *Results) {
	res = &Results{}
	resp, err := Get(url, ip)

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

	fmt.Printf("[%d] [%s] %s\n", res.Code, res.Proto, res.URL)

	// here we should fetch all of the other resources in a different goroutine

	for i := range urls {
		resourcePool.Add(1)

		rsrc := &Resource{connURL: urls[i], connIP: ""}
		res.Resources = append(res.Resources, rsrc)
		go res.Resources[i].FetchResource()
	}

	resourcePool.Wait()

	return
}
