package scraper

import (
	"bytes"
	"crypto/tls"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"golang.org/x/net/html"
)

// getAttr pulls a specific attribute from a token/element
func getAttr(attr string, attrs []html.Attribute) (val string) {
	for _, item := range attrs {
		if item.Key == attr {
			val = item.Val
			break
		}
	}

	return
}

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
		case tt == html.StartTagToken || tt == html.SelfClosingTagToken:
			t := z.Token()

			var src string

			switch t.Data {
			case "link":
				src = getAttr("href", t.Attr)

				rel := getAttr("rel", t.Attr)

				if len(rel) > 0 && strings.ToLower(rel) != "stylesheet" && strings.ToLower(rel) != "shortcut icon" {
					continue
				}

			case "script":
				src = getAttr("src", t.Attr)

			case "img":
				src = getAttr("src", t.Attr)

			default:
				continue
			}

			if len(src) == 0 {
				continue
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
			if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
				continue
			}

			urls = append(urls, src)
		}
	}
}

func connHostname(URL string) (host string, err error) {
	tmp, err := url.Parse(URL)

	if err != nil {
		return
	}

	host = tmp.Host
	return
}

// Resource represents a single entity of many within a given crawl. These should
// only be of type css, js, jpg, png, etc (static resources).
type Resource struct {
	// connURL is the initial URL received by input
	connURL string

	// connIP is the initial IP address received by input
	connIP string

	// connHostname represents the original requested hostname for the resource
	connHostname string

	// URL represents the resulting static URL derived by the original result page
	URL string

	// Hostname represents the resulting hostname derived by the original returned
	// resource
	Hostname string

	// Remote represents if the resulting resource is remote to the original domain
	Remote bool

	// Error represents any errors that may have occurred when fetching the resource
	Error error

	// Code represents the numeric HTTP based status code
	Code int

	// Proto represents the end protocol used to fetch the page. For example, HTTP/2.0
	Proto string

	// Scheme represents the end scheme used to fetch the page. For example, https
	Scheme string

	// ContentLength represents the number of bytes in the body of the response
	ContentLength int64

	// TLS represents the SSL/TLS handshake/session if the resource was loaded over
	// SSL.
	TLS *tls.ConnectionState

	// Time represents the time it took to complete the request
	Time *TimerResult

	// logging functionality
	logger *log.Logger
}

// fetchResource fetches a singular resource from a page, returning a *Resource struct.
// As we don't care much about the body of the resource, that can safely be ignored. We
// must still close the body object, however.
func (rsrc *Resource) fetchResource() {
	var err error

	defer resourcePool.Done()

	// calculate the time it takes to fetch the request
	resp, err := Get(rsrc.connURL, rsrc.connIP)

	if err != nil {
		rsrc.Error = err
		return
	}

	if resp.Body != nil {
		resp.Body.Close()
	}

	rsrc.connHostname, err = connHostname(rsrc.connURL)
	if err != nil {
		rsrc.Error = err
		return
	}

	rsrc.Hostname = resp.Request.Host
	rsrc.URL = resp.URL
	rsrc.Code = resp.StatusCode
	rsrc.Proto = resp.Proto
	rsrc.Scheme = resp.Request.URL.Scheme
	rsrc.ContentLength = resp.ContentLength
	rsrc.TLS = resp.TLS
	rsrc.Time = resp.Time

	if rsrc.Hostname != rsrc.connHostname {
		rsrc.Remote = true
	}

	rsrc.logger.Printf("fetched %s in %dms with status %d", rsrc.URL, rsrc.Time.Milli, rsrc.Code)

	return
}

// Results -- struct returned by Crawl() to represent the entire crawl process
type Results struct {
	// Inherit the Resource struct
	Resource

	// Body represents a string implementation of the byte array returned by
	// http.Response
	Body string

	// Slice of Resource structs containing the needed resources for the given URL
	Resources []*Resource

	// ResourceTime shows how long it took to fetch all resources
	ResourceTime *TimerResult

	// TotalTime represents the time it took to crawl the site
	TotalTime *TimerResult

	// logging functionality
	logger *log.Logger
}

var resourcePool sync.WaitGroup

// fetchURL manages the fetching of the main resource, as well as all child resources,
// providing a Results struct containing the entire crawl data needed
func FetchURL(URL string, IP string, logger *log.Logger) (res *Results) {
	res = &Results{logger: logger}
	crawlTimer := NewTimer()

	// actually fetch the request
	resp, err := Get(URL, IP)

	defer func() {
		crawlTimer.End()
		res.TotalTime = crawlTimer.Result
	}()

	if err != nil {
		res.Error = err
		return
	}

	defer resp.Body.Close()

	res.connHostname, err = connHostname(URL)
	if err != nil {
		res.Error = err
		return
	}

	res.connURL = URL
	res.connIP = IP
	res.Hostname = resp.Request.Host
	res.URL = resp.URL
	res.Code = resp.StatusCode
	res.Proto = resp.Proto
	res.Scheme = resp.Request.URL.Scheme
	res.ContentLength = resp.ContentLength
	res.TLS = resp.TLS
	res.Time = resp.Time

	if res.Hostname != res.connHostname {
		res.Remote = true
	}

	buf, _ := ioutil.ReadAll(resp.Body)
	b := ioutil.NopCloser(bytes.NewReader(buf))
	defer b.Close()

	bbytes, err := ioutil.ReadAll(bytes.NewBuffer(buf))
	if len(bbytes) != 0 {
		res.Body = string(bbytes[:])
	}

	urls := getSrc(b, resp.Request)

	res.logger.Printf("fetched %s in %dms with status %d", res.URL, res.Time.Milli, res.Code)

	resourceTime := NewTimer()

	defer func() {
		resourceTime.End()
		res.ResourceTime = resourceTime.Result
	}()

	for i := range urls {
		resourcePool.Add(1)

		rsrc := &Resource{connURL: urls[i], connIP: "", logger: res.logger}
		res.Resources = append(res.Resources, rsrc)
		go res.Resources[i].fetchResource()
	}

	resourcePool.Wait()

	return
}

type Domain struct {
	URL *url.URL
	IP  string
}

func Crawl(domains []*Domain, logger *log.Logger) (results []*Results) {
	var wg sync.WaitGroup
	timer := NewTimer()

	// loop through all supplied urls and send them to a worker to be fetched
	for _, domain := range domains {
		wg.Add(1)

		go func(domain *Domain) {
			defer wg.Done()

			result := FetchURL(domain.URL.String(), "", logger)
			results = append(results, result)

			if result.Error != nil {
				logger.Printf("error scanning %s (error: %s)", domain.URL.String(), result.Error)
			} else {
				logger.Printf("finished scanning %s (%dms)", domain.URL.String(), result.TotalTime.Milli)
			}
		}(domain)
	}

	// wait for all workers to complete their tasks
	wg.Wait()
	timer.End()

	logger.Printf("finished scanning %d urls in %d seconds", len(results), timer.Result.Seconds)

	// give some extra details
	var resSuccess, resError int
	for i := range results {
		if results[i].Error != nil {
			resError++
			continue
		}

		resSuccess++
	}

	logger.Printf("%d successful, %d errored", resSuccess, resError)

	return results
}
