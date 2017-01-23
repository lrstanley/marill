// Author: Liam Stanley <me@liamstanley.io>
// Docs: https://marill.liam.sh/
// Repo: https://github.com/lrstanley/marill

package scraper

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/lrstanley/go-sempool"
	"github.com/lrstanley/marill/utils"
)

var reIP = regexp.MustCompile(`^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$`)

// Response represents the data for the HTTP-based request, closely matching
// http.Response
type Response struct {
	Remote        bool         // Remote is true if the origin is remote (unknown ip)
	Code          int          // Code is the numeric HTTP based status code
	URL           *url.URL     `json:"-"` // URL is the resulting static URL derived by the original result page
	Body          string       // Body is the response body. Used for primary requests, ignored for Resource structs.
	Headers       http.Header  // Headers is a map[string][]string of headers
	ContentLength int64        // ContentLength is the number of bytes in the body of the response
	TLS           *TLSResponse // TLS is the SSL/TLS session if the resource was loaded over SSL/TLS
}

// Resource represents a single entity of many within a given crawl. These should
// only be of type css, js, jpg, png, etc (static resources).
type Resource struct {
	URL      string             // the url -- this should exist regardless of failure
	Request  *Domain            // request represents what we were provided before the request
	Response Response           // Response represents the end result/data/status/etc.
	Error    error              // Error represents an error of a completely failed request
	Time     *utils.TimerResult // Time is the time it took to complete the request
}

func (r *Resource) String() string {
	if r.Response.URL != nil && r.Time != nil {
		return fmt.Sprintf("<[Resource] request:%s response:%s ip:%q code:%d time:%dms err:%q>", r.Request.URL, r.Response.URL, r.Request.IP, r.Response.Code, r.Time.Milli, r.Error)
	}

	return fmt.Sprintf("<[Resource] request:%s ip:%q err:%q>", r.Request.URL, r.Request.IP, r.Error)
}

// FetchResult -- struct returned by Crawl() to represent the entire crawl process
type FetchResult struct {
	Resource                        // Inherit the Resource struct
	Assets       []*Resource        `json:"-"` // Assets containing the needed resources for the given URL
	ResourceTime *utils.TimerResult // ResourceTime is the time it took to fetch all resources
	TotalTime    *utils.TimerResult // TotalTime is the time it took to crawl the site
}

func (r *FetchResult) String() string {
	if r.Assets != nil && r.ResourceTime != nil && r.TotalTime != nil {
		return fmt.Sprintf("<[Results] request:%s response:%s ip:%q code:%d resources:%d resource-time:%dms total-time:%dms err:%q>", r.Request.URL, r.Response.URL, r.Request.IP, r.Response.Code, len(r.Assets), r.ResourceTime.Milli, r.TotalTime.Milli, r.Error)
	}

	return fmt.Sprintf("<[Results] request:%s response:%s ip:%q err:%q>", r.Request.URL, r.URL, r.Request.IP, r.Error)
}

// Domain represents a url we need to fetch, including the items needed to
// fetch said url. E.g: host, port, ip, scheme, path, etc.
type Domain struct {
	URL *url.URL `json:"-"`
	IP  string
}

func (d *Domain) String() string {
	return fmt.Sprintf("<[Domain] url:%q ip:%q>", d.URL, d.IP)
}

// Crawler is the higher level struct which wraps the entire threaded crawl process
type Crawler struct {
	Log     *log.Logger       // output log
	ipmap   map[string]string // domain -> ip map, to easily tell if something is local
	Results []*FetchResult    // scan results, should only be access when scan is complete
	Pool    sempool.Pool      // thread pool for fetching main resources
	ResPool sempool.Pool      // thread pool for fetching assets
	Cnf     CrawlerConfig
}

// CrawlerConfig is the configuration which changes Crawler
type CrawlerConfig struct {
	Domains       []*Domain     // list of domains to scan
	Assets        bool          // if we want to pull the assets for the page too
	NoRemote      bool          // ignore all resources that match a remote IP
	AllowInsecure bool          // if SSL errors should be ignored
	Delay         time.Duration // delay before each resource is crawled
	HTTPTimeout   time.Duration // http timeout before a request has become stale
	Threads       int           // total number of threads to run crawls in
}

// fetchResource fetches a singular resource from a page, returning a *Resource struct.
// As we don't care much about the body of the resource, that can safely be ignored. We
// must still close the body object, however.
func (c *Crawler) fetchResource(rsrc *Resource) {
	defer c.ResPool.Free()
	var err error

	rsrc.URL = rsrc.Request.URL.String()

	resp, err := c.Get(rsrc.Request.URL.String())
	if err != nil {
		rsrc.Error = err
		return
	}

	if resp.Body != nil {
		// we don't care about the body, but we want to know how large it is.
		// count the bytes but discard them.
		if resp.ContentLength < 1 {
			resp.ContentLength, _ = io.Copy(ioutil.Discard, resp.Body)
		}

		resp.Body.Close() // ensure the body stream is closed
	}

	rsrc.Response = Response{
		URL:           resp.URL,
		Code:          resp.StatusCode,
		ContentLength: resp.ContentLength,
		Headers:       resp.Header,
		TLS:           tlsToShort(resp.TLS),
	}

	if rsrc.Response.URL.Host != rsrc.Request.URL.Host {
		rsrc.Response.Remote = true
	}

	rsrc.URL = rsrc.Response.URL.String()
	if rsrc.URL != rsrc.Request.URL.String() {
		rsrc.URL = fmt.Sprintf("%s (-> %s)", rsrc.Request.URL, rsrc.URL)
	}

	rsrc.Time = resp.Time

	c.Log.Printf("fetched %s in %dms with status %d", rsrc.Response.URL, rsrc.Time.Milli, rsrc.Response.Code)
}

// Fetch manages the fetching of the main resource, as well as all child resources,
// providing a FetchResult struct containing the entire crawl data needed
func (c *Crawler) Fetch(res *FetchResult) {
	var err error

	crawlTimer := utils.NewTimer()
	defer func() {
		crawlTimer.End()
		res.TotalTime = crawlTimer.Result
	}()

	res.URL = res.Request.URL.String()

	// actually fetch the request
	resp, err := c.Get(res.Request.URL.String())
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
		TLS:           tlsToShort(resp.TLS),
	}

	if res.Response.URL.Host != res.Request.URL.Host {
		res.Response.Remote = true
	}

	res.URL = res.Response.URL.String()
	if res.URL != res.Request.URL.String() {
		res.URL = fmt.Sprintf("%s (-> %s)", res.Request.URL, res.URL)
	}

	res.Time = resp.Time

	buf, _ := ioutil.ReadAll(resp.Body)
	b := ioutil.NopCloser(bytes.NewReader(buf))
	defer b.Close()

	bbytes, err := ioutil.ReadAll(bytes.NewBuffer(buf))
	if err == nil && len(bbytes) != 0 {
		res.Response.Body = string(bbytes[:])
	}

	if res.Response.ContentLength < 1 {
		res.Response.ContentLength = int64(len(buf))
	}

	c.Log.Printf("fetched %s in %dms with status %d", res.Response.URL.String(), res.Time.Milli, res.Response.Code)

	resourceTime := utils.NewTimer()

	defer func() {
		resourceTime.End()
		res.ResourceTime = resourceTime.Result
	}()

	if c.Cnf.Assets {
		urls := []*url.URL{}

		for _, uri := range getSrc(b, res.Response.URL) {
			parsedURI, err := url.Parse(uri)
			if err != nil {
				c.Log.Printf("unable to parse asset uri [%s], resource: %s: %s", uri, res.Request, err)
				continue
			}

			urls = append(urls, parsedURI)
		}

		c.ResPool = sempool.New(4)

		for i := range urls {
			if c.Cnf.NoRemote {
				if c.IsRemote(urls[i].Host) {
					c.Log.Printf("host %s (url: %s) resolves to a unknown remote ip, skipping", urls[i].Host, urls[i])
					continue
				}
			}

			c.ResPool.Slot()

			asset := &Resource{Request: &Domain{URL: urls[i]}}
			res.Assets = append(res.Assets, asset)
			go c.fetchResource(res.Assets[len(res.Assets)-1])
		}

		c.ResPool.Wait()
	}

	return
}

// Crawl represents the higher level functionality of scraper. Crawl should
// concurrently request the needed resources for a list of domains, allowing
// the bypass of DNS lookups where necessary.
func (c *Crawler) Crawl() {
	c.Pool = sempool.New(c.Cnf.Threads)
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
			if c.Cnf.Delay.String() != "0s" {
				c.Log.Printf("delaying %s before starting crawl on %s", c.Cnf.Delay, domain)
				time.Sleep(c.Cnf.Delay)
			}

			result := &FetchResult{}
			result.Request = domain

			c.Fetch(result)
			// Check to see if there were errors here that we want to ignore.
			if c.Cnf.NoRemote && result.Error == ErrNotMatchOrigin {
				c.Log.Printf("skipping %s as skip remote was used (error: %s)", domain, result.Error)
				return
			}

			c.Results = append(c.Results, result)

			if result.Error != nil {
				c.Log.Printf("error scanning %s (error: %s)", domain, result.Error)
			} else {
				c.Log.Printf("finished scanning %s (%dms)", domain, result.TotalTime.Milli)
			}
		}(domain)
	}

	// wait for all workers to complete their tasks
	c.Pool.Wait()
	timer.End()

	c.Log.Printf("finished scanning %d urls in %d seconds", len(c.Results), timer.Result.Seconds)

	// give some extra details
	var resSuccess, resError int
	for i := 0; i < len(c.Results); i++ {
		if c.Results[i].Error != nil {
			resError++
			continue
		}

		resSuccess++
	}
	c.Log.Printf("%d successful, %d failed\n", resSuccess, resError)

	return
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
