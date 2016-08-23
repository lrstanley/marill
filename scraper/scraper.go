package scraper

import (
	"bytes"
	"crypto/tls"
	"io/ioutil"
	"log"
	"net/url"
	"strings"
	"sync"
)

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
func (c *Crawler) fetchResource(rsrc *Resource) {
	var err error

	defer resourcePool.Done()

	// calculate the time it takes to fetch the request
	resp, err := Get(c.ipmap, rsrc.connURL)

	if err != nil {
		rsrc.Error = err
		return
	}

	if resp.Body != nil {
		resp.Body.Close()
	}

	rsrc.connHostname, err = getHost(rsrc.connURL)
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

	c.Log.Printf("fetched %s in %dms with status %d", rsrc.URL, rsrc.Time.Milli, rsrc.Code)

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
}

var resourcePool sync.WaitGroup

// FetchURL manages the fetching of the main resource, as well as all child resources,
// providing a Results struct containing the entire crawl data needed
func (c *Crawler) FetchURL(URL string) (res *Results) {
	res = &Results{}
	crawlTimer := NewTimer()

	host, err := getHost(URL)
	if err != nil {
		res.Error = err
		return
	}

	// actually fetch the request
	resp, err := Get(c.ipmap, URL)

	defer func() {
		crawlTimer.End()
		res.TotalTime = crawlTimer.Result
	}()

	if err != nil {
		res.Error = err
		return
	}

	defer resp.Body.Close()

	res.connHostname = host
	if err != nil {
		res.Error = err
		return
	}

	res.connURL = URL
	res.connIP = c.ipmap[host]
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

	c.Log.Printf("fetched %s in %dms with status %d", res.URL, res.Time.Milli, res.Code)

	resourceTime := NewTimer()

	defer func() {
		resourceTime.End()
		res.ResourceTime = resourceTime.Result
	}()

	for i := range urls {
		resourcePool.Add(1)

		rsrc := &Resource{connURL: urls[i], connIP: ""}
		res.Resources = append(res.Resources, rsrc)
		go c.fetchResource(res.Resources[i])
	}

	resourcePool.Wait()

	return
}

// Domain represents a url we need to fetch, including the items needed to
// fetch said url. E.g: host, port, ip, scheme, path, etc.
type Domain struct {
	URL *url.URL
	IP  string
}

type Crawler struct {
	Log     *log.Logger
	Domains []*Domain
	ipmap   map[string]string
}

// Crawl represents the higher level functionality of scraper. Crawl should
// concurrently request the needed resources for a list of domains, allowing
// the bypass of DNS lookups where necessary.
func (c *Crawler) Crawl() (results []*Results) {
	var wg sync.WaitGroup
	timer := NewTimer()

	c.ipmap = make(map[string]string)
	for i := range c.Domains {
		c.ipmap[c.Domains[i].URL.Host] = c.Domains[i].IP
		c.ipmap[strings.TrimPrefix(c.Domains[i].URL.Host, "www.")] = c.Domains[i].IP // no www. directive
		c.ipmap["www."+c.Domains[i].URL.Host] = c.Domains[i].IP                      // www. directive
	}

	// loop through all supplied urls and send them to a worker to be fetched
	for _, domain := range c.Domains {
		wg.Add(1)

		go func(domain *Domain) {
			defer wg.Done()

			result := c.FetchURL(domain.URL.String())
			results = append(results, result)

			if result.Error != nil {
				c.Log.Printf("error scanning %s (error: %s)", domain.URL.String(), result.Error)
			} else {
				c.Log.Printf("finished scanning %s (%dms)", domain.URL.String(), result.TotalTime.Milli)
			}
		}(domain)
	}

	// wait for all workers to complete their tasks
	wg.Wait()
	timer.End()

	c.Log.Printf("finished scanning %d urls in %d seconds", len(results), timer.Result.Seconds)

	// give some extra details
	var resSuccess, resError int
	for i := range results {
		if results[i].Error != nil {
			resError++
			continue
		}

		resSuccess++
	}

	c.Log.Printf("%d successful, %d errored", resSuccess, resError)

	return results
}
