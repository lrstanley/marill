package scraper

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"strings"
	"sync"
)

// ResourceOrigin represents data originally used to create this resource
type ResourceOrigin struct {
	// URL represents the initial URL received by input
	URL string
	// IP represents the initial IP address received by input
	IP string
	// Host represents the original requested hostname for the resource
	Host string
}

// Response represents the data for the HTTP-based request, closely matching
// http.Response
type Response struct {
	// Host represents the resulting host derived by the original returned
	// resource
	Host string

	// Remote represents if the resulting resource is remote to the original domain
	Remote bool

	// Code represents the numeric HTTP based status code
	Code int

	// Body represents a string implementation of the byte array returned by
	// http.Response. Only used for primary requests, ignored for Resource structs.
	Body string

	// Scheme represents the end scheme used to fetch the page. For example, https
	Scheme string

	// ContentLength represents the number of bytes in the body of the response
	ContentLength int64

	// TLS represents the SSL/TLS handshake/session if the resource was loaded over
	// SSL.
	TLS *tls.ConnectionState
}

// Resource represents a single entity of many within a given crawl. These should
// only be of type css, js, jpg, png, etc (static resources).
type Resource struct {
	// request represents what we were provided before the request
	request ResourceOrigin

	// Response represents the end result/data/status/etc.
	Response Response

	// URL represents the resulting static URL derived by the original result page
	URL string

	// Error represents any errors that may have occurred when fetching the resource
	Error error

	// Time represents the time it took to complete the request
	Time *TimerResult
}

// fetchResource fetches a singular resource from a page, returning a *Resource struct.
// As we don't care much about the body of the resource, that can safely be ignored. We
// must still close the body object, however.
func (c *Crawler) fetchResource(rsrc *Resource) {
	var err error

	defer resourcePool.Done()

	// calculate the time it takes to fetch the request
	resp, err := c.Get(rsrc.request.URL)

	if err != nil {
		rsrc.Error = err
		return
	}

	if resp.Body != nil {
		resp.Body.Close()
	}

	rsrc.request.Host, err = getHost(rsrc.request.URL)
	if err != nil {
		rsrc.Error = err
		return
	}

	rsrc.Response.Host = resp.Request.Host
	rsrc.URL = resp.URL
	rsrc.Response.Code = resp.StatusCode
	rsrc.Response.Scheme = resp.Request.URL.Scheme
	rsrc.Response.ContentLength = resp.ContentLength
	rsrc.Response.TLS = resp.TLS
	rsrc.Time = resp.Time

	if rsrc.Response.Host != rsrc.request.Host {
		rsrc.Response.Remote = true
	}

	c.Log.Printf("fetched %s in %dms with status %d", rsrc.URL, rsrc.Time.Milli, rsrc.Response.Code)

	return
}

// Results -- struct returned by Crawl() to represent the entire crawl process
type Results struct {
	// Inherit the Resource struct
	Resource

	// Slice of Resource structs containing the needed resources for the given URL
	Resources []*Resource

	// ResourceTime shows how long it took to fetch all resources
	ResourceTime *TimerResult

	// TotalTime represents the time it took to crawl the site
	TotalTime *TimerResult
}

func (r *Results) String() string {
	if r.Resources != nil && r.ResourceTime != nil && r.TotalTime != nil {
		return fmt.Sprintf("<url(%s) == %d, resources(%d), resourceTime(%dms), totalTime(%dms), err(%s)>", r.URL, r.Response.Code, len(r.Resources), r.ResourceTime.Milli, r.TotalTime.Milli, r.Error)
	}

	return fmt.Sprintf("<url(%s), ip(%s), err(%s)>", r.request.URL, r.request.IP, r.Error)
}

var resourcePool sync.WaitGroup

// FetchURL manages the fetching of the main resource, as well as all child resources,
// providing a Results struct containing the entire crawl data needed
func (c *Crawler) FetchURL(URL string) (res *Results) {
	res = &Results{}
	crawlTimer := NewTimer()

	var err error

	res.request.URL = URL
	res.request.Host, err = getHost(URL)
	if err != nil {
		res.Error = err
		return
	}
	res.request.IP = c.ipmap[res.request.Host]

	// actually fetch the request
	resp, err := c.Get(URL)

	defer func() {
		crawlTimer.End()
		res.TotalTime = crawlTimer.Result
	}()

	if err != nil {
		res.Error = err
		return
	}

	defer resp.Body.Close()

	res.Response.Host = resp.Request.Host
	res.URL = resp.URL
	res.Response.Code = resp.StatusCode
	res.Response.Scheme = resp.Request.URL.Scheme
	res.Response.ContentLength = resp.ContentLength
	res.Response.TLS = resp.TLS
	res.Time = resp.Time

	if res.Response.Host != res.request.Host {
		res.Response.Remote = true
	}

	buf, _ := ioutil.ReadAll(resp.Body)
	b := ioutil.NopCloser(bytes.NewReader(buf))
	defer b.Close()

	bbytes, err := ioutil.ReadAll(bytes.NewBuffer(buf))
	if err == nil && len(bbytes) != 0 {
		res.Response.Body = string(bbytes[:])
	}

	urls := getSrc(b, resp.Request)

	c.Log.Printf("fetched %s in %dms with status %d", res.URL, res.Time.Milli, res.Response.Code)

	resourceTime := NewTimer()

	defer func() {
		resourceTime.End()
		res.ResourceTime = resourceTime.Result
	}()

	for i := range urls {
		resourcePool.Add(1)

		rsrc := &Resource{request: ResourceOrigin{URL: urls[i]}}
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

// Crawler is the higher level struct which wraps the entire threaded crawl process
type Crawler struct {
	Log     *log.Logger       // output log
	ipmap   map[string]string // domain -> ip map, to easily tell if something is local
	Results []*Results        // scan results, should only be access when scan is complete
	Cnf     CrawlerConfig
}

// CrawlerConfig is the configuration which changes Crawler
type CrawlerConfig struct {
	Domains   []*Domain // list of domains to scan
	Recursive bool      // if we want to pull the resources for the page too
}

// Crawl represents the higher level functionality of scraper. Crawl should
// concurrently request the needed resources for a list of domains, allowing
// the bypass of DNS lookups where necessary.
func (c *Crawler) Crawl() {
	var results []*Results
	var wg sync.WaitGroup
	timer := NewTimer()

	// strip all common duplicate domain/ip pairs
	stripDups(&c.Cnf.Domains)

	c.ipmap = make(map[string]string)
	var dom string
	for i := range c.Cnf.Domains {
		c.ipmap[c.Cnf.Domains[i].URL.Host] = c.Cnf.Domains[i].IP
		dom = strings.TrimPrefix(c.Cnf.Domains[i].URL.Host, "www.")
		c.ipmap[dom] = c.Cnf.Domains[i].IP        // no www. directive
		c.ipmap["www."+dom] = c.Cnf.Domains[i].IP // www. directive
	}

	// loop through all supplied urls and send them to a worker to be fetched
	for _, domain := range c.Cnf.Domains {
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

	c.Results = results

	return
}

// GetResults gets the potential results of a given requested url/ip
func (c *Crawler) GetResults(URL, IP string) *Results {
	for i := range c.Results {
		if c.Results[i].request.URL == URL && c.Results[i].request.IP == IP {
			return c.Results[i]
		}
	}

	return nil
}
