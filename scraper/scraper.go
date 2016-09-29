package scraper

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Liamraystanley/marill/utils"
)

// ResourceOrigin represents data originally used to create the request
type ResourceOrigin struct {
	URL  string // URL is the initial URL received by input
	IP   string // IP is the initial IP address received by input
	Host string // Host is the original requested hostname for the resource
}

// Response represents the data for the HTTP-based request, closely matching
// http.Response
type Response struct {
	Remote        bool                 // Remote is true if the origin is remote (unknown ip)
	Code          int                  // Code is the numeric HTTP based status code
	URL           *url.URL             // URL is the resulting static URL derived by the original result page
	Body          string               // Body is the response body. Used for primary requests, ignored for Resource structs.
	Headers       http.Header          // Headers is a map[string][]string of headers
	ContentLength int64                // ContentLength is the number of bytes in the body of the response
	TLS           *tls.ConnectionState // TLS is the SSL/TLS session if the resource was loaded over SSL/TLS
}

// Resource represents a single entity of many within a given crawl. These should
// only be of type css, js, jpg, png, etc (static resources).
type Resource struct {
	Request  ResourceOrigin     // request represents what we were provided before the request
	Response Response           // Response represents the end result/data/status/etc.
	Error    error              // Error represents an error of a completely failed request
	Time     *utils.TimerResult // Time is the time it took to complete the request
}

// fetchResource fetches a singular resource from a page, returning a *Resource struct.
// As we don't care much about the body of the resource, that can safely be ignored. We
// must still close the body object, however.
func (c *Crawler) fetchResource(rsrc *Resource) {
	defer c.ResPool.Free()
	var err error

	resp, err := c.Get(rsrc.Request.URL)
	if err != nil {
		rsrc.Error = err
		return
	}

	if resp.Body != nil {
		resp.Body.Close() // ensure the body stream is closed
	}

	rsrc.Request.Host, err = utils.GetHost(rsrc.Request.URL)
	if err != nil {
		rsrc.Error = err
		return
	}

	rsrc.Response = Response{
		URL:           resp.URL,
		Code:          resp.StatusCode,
		ContentLength: resp.ContentLength,
		Headers:       resp.Header,
		TLS:           resp.TLS,
	}

	if rsrc.Response.URL.Host != rsrc.Request.Host {
		rsrc.Response.Remote = true
	}

	rsrc.Time = resp.Time

	c.Log.Printf("fetched %s in %dms with status %d", rsrc.Response.URL.String(), rsrc.Time.Milli, rsrc.Response.Code)

	return
}

// Results -- struct returned by Crawl() to represent the entire crawl process
type Results struct {
	Resource                        // Inherit the Resource struct
	Resources    []*Resource        // Resources containing the needed resources for the given URL
	ResourceTime *utils.TimerResult // ResourceTime is the time it took to fetch all resources
	TotalTime    *utils.TimerResult // TotalTime is the time it took to crawl the site
}

func (r *Results) String() string {
	if r.Resources != nil && r.ResourceTime != nil && r.TotalTime != nil {
		return fmt.Sprintf("<url(%s) == %d, resources(%d), resourceTime(%dms), totalTime(%dms), err(%s)>", r.Response.URL.String(), r.Response.Code, len(r.Resources), r.ResourceTime.Milli, r.TotalTime.Milli, r.Error)
	}

	return fmt.Sprintf("<url(%s), ip(%s), err(%s)>", r.Request.URL, r.Request.IP, r.Error)
}

// FetchURL manages the fetching of the main resource, as well as all child resources,
// providing a Results struct containing the entire crawl data needed
func (c *Crawler) FetchURL(URL string) (res *Results) {
	var err error

	res = &Results{}
	crawlTimer := utils.NewTimer()
	defer func() {
		crawlTimer.End()
		res.TotalTime = crawlTimer.Result
	}()

	res.Request.URL = URL
	res.Request.Host, err = utils.GetHost(URL)
	if err != nil {
		res.Error = err
		return
	}
	res.Request.IP = c.ipmap[res.Request.Host]

	// actually fetch the request
	resp, err := c.Get(URL)
	if err != nil {
		res.Error = err
		return
	}

	if resp.Body != nil {
		defer resp.Body.Close() // ensure the body stream is closed
	}

	res.Response = Response{
		URL:           resp.URL,
		Code:          resp.StatusCode,
		ContentLength: resp.ContentLength,
		Headers:       resp.Header,
		TLS:           resp.TLS,
	}

	if res.Response.URL.Host != res.Request.Host {
		res.Response.Remote = true
	}

	res.Time = resp.Time

	buf, _ := ioutil.ReadAll(resp.Body)
	b := ioutil.NopCloser(bytes.NewReader(buf))
	defer b.Close()

	bbytes, err := ioutil.ReadAll(bytes.NewBuffer(buf))
	if err == nil && len(bbytes) != 0 {
		res.Response.Body = string(bbytes[:])
	}

	urls := getSrc(b, resp.Request)

	c.Log.Printf("fetched %s in %dms with status %d", res.Response.URL.String(), res.Time.Milli, res.Response.Code)

	resourceTime := utils.NewTimer()

	defer func() {
		resourceTime.End()
		res.ResourceTime = resourceTime.Result
	}()

	if c.Cnf.Recursive {
		c.ResPool = utils.NewPool(4)

		for i := range urls {
			if c.Cnf.NoRemote {
				host, err := utils.GetHost(urls[i])
				if err != nil {
					c.Log.Printf("unable to get host of %s, skipping", urls[i])
					continue
				}

				if c.IsRemote(host) {
					c.Log.Printf("host %s (url: %s) resolves to a unknown remote ip, skipping", host, urls[i])
					continue
				}
			}

			c.ResPool.Slot()

			rsrc := &Resource{Request: ResourceOrigin{URL: urls[i]}}
			res.Resources = append(res.Resources, rsrc)
			go c.fetchResource(res.Resources[i])
		}

		c.ResPool.Wait()
	}

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
	Pool    utils.Pool        // thread pool for fetching main resources
	ResPool utils.Pool        // thread pool for fetching assets
	Cnf     CrawlerConfig
}

// CrawlerConfig is the configuration which changes Crawler
type CrawlerConfig struct {
	Domains   []*Domain     // list of domains to scan
	Recursive bool          // if we want to pull the resources for the page too
	NoRemote  bool          // ignore all resources that match a remote IP
	Delay     time.Duration // delay before each resource is crawled
	Threads   int           // total number of threads to run crawls in
}

// Crawl represents the higher level functionality of scraper. Crawl should
// concurrently request the needed resources for a list of domains, allowing
// the bypass of DNS lookups where necessary.
func (c *Crawler) Crawl() {
	var results []*Results
	c.Pool = utils.NewPool(c.Cnf.Threads)
	timer := utils.NewTimer()

	// strip all common duplicate domain/ip pairs
	stripDups(&c.Cnf.Domains)

	c.ipmap = make(map[string]string)
	var dom string
	for i := 0; i < len(c.Cnf.Domains); i++ {
		c.ipmap[c.Cnf.Domains[i].URL.Host] = c.Cnf.Domains[i].IP
		dom = strings.TrimPrefix(c.Cnf.Domains[i].URL.Host, "www.")
		c.ipmap[dom] = c.Cnf.Domains[i].IP        // no www. directive
		c.ipmap["www."+dom] = c.Cnf.Domains[i].IP // www. directive
	}

	// loop through all supplied urls and send them to a worker to be fetched
	for _, domain := range c.Cnf.Domains {
		c.Pool.Slot()

		go func(domain *Domain) {
			defer c.Pool.Free()

			// delay if they have a time set
			if c.Cnf.Delay.String() == "0s" {
				c.Log.Printf("delaying %s before starting crawl on %s", c.Cnf.Delay.String(), domain.URL.String())
				time.Sleep(c.Cnf.Delay)
			}

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
	c.Pool.Wait()
	timer.End()

	c.Log.Printf("finished scanning %d urls in %d seconds", len(results), timer.Result.Seconds)

	// give some extra details
	var resSuccess, resError int
	for i := 0; i < len(results); i++ {
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
	for i := 0; i < len(c.Results); i++ {
		if c.Results[i].Request.URL == URL && c.Results[i].Request.IP == IP {
			return c.Results[i]
		}
	}

	return nil
}

// IsRemote checks to see if host is remote, and if it should be scanned
func (c *Crawler) IsRemote(host string) bool {
	if _, ok := c.ipmap[host]; ok {
		// it is in our IP map, so it was already specified
		return true
	}

	ip, err := utils.LookupIP(host)
	if err != nil {
		// there is some form of issue, assume it's local so the error
		// is returned during the scraping process
		return false
	}

	// check to see if the IP is in our map as a local IP address
	for _, v := range c.ipmap {
		if v == ip {
			return false
		}
	}

	return true
}
