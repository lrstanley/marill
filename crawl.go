// Author: Liam Stanley <me@liamstanley.io>
// Docs: https://marill.liam.sh/
// Repo: https://github.com/Liamraystanley/marill

package main

import (
	"fmt"

	"github.com/Liamraystanley/marill/domfinder"
	"github.com/Liamraystanley/marill/scraper"
)

type Scan struct {
	finder     *domfinder.Finder
	crawler    *scraper.Crawler
	results    []*TestResult
	tests      []*Test
	successful int
	failed     int
}

func crawl() (*Scan, error) {
	var err error
	res := &Scan{}

	// fetch the tests ahead of time to ensure there are no syntax errors or anything
	res.tests = genTests()

	res.crawler = &scraper.Crawler{Log: logger}

	if conf.scan.manualList != "" {
		logger.Println("manually supplied url list")
		res.crawler.Cnf.Domains, err = parseManualList()
		if err != nil {
			return nil, NewErr{Code: ErrDomainFlag, deepErr: err}
		}
	} else {
		logger.Println("checking for running webservers")

		finder := &domfinder.Finder{Log: logger}
		if err := finder.GetWebservers(); err != nil {
			return nil, NewErr{Code: ErrProcList, deepErr: err}
		}

		if outlist := ""; len(finder.Procs) > 0 {
			for _, proc := range finder.Procs {
				outlist += fmt.Sprintf("[%s:%s] ", proc.Name, proc.PID)
			}
			logger.Printf("found %d procs matching a webserver: %s", len(finder.Procs), outlist)
			out.Printf("found %d procs matching a webserver\n", len(finder.Procs))
		}

		// start crawling for domains
		if err := finder.GetDomains(); err != nil {
			return nil, NewErr{Code: ErrGetDomains, deepErr: err}
		}

		finder.Filter(domfinder.DomainFilter{
			IgnoreHTTP:  conf.scan.ignoreHTTP,
			IgnoreHTTPS: conf.scan.ignoreHTTPS,
			IgnoreMatch: conf.scan.ignoreMatch,
			MatchOnly:   conf.scan.matchOnly,
		})

		if len(finder.Domains) == 0 {
			return nil, NewErr{Code: ErrNoDomainsFound}
		}

		logger.Printf("found %d domains on webserver %s (exe: %s, pid: %s)", len(finder.Domains), finder.MainProc.Name, finder.MainProc.Exe, finder.MainProc.PID)

		for _, domain := range finder.Domains {
			res.crawler.Cnf.Domains = append(res.crawler.Cnf.Domains, &scraper.Domain{URL: domain.URL, IP: domain.IP})
		}
	}

	res.crawler.Cnf.Recursive = conf.scan.recursive
	res.crawler.Cnf.NoRemote = conf.scan.ignoreRemote
	res.crawler.Cnf.Delay = conf.scan.delay

	logger.Printf("starting crawler...")
	out.Printf("starting scan on %d domains\n", len(res.crawler.Cnf.Domains))
	res.crawler.Crawl()
	out.Println("{lightgreen}scan complete{c}")

	out.Println("{lightblue}starting tests{c}")
	res.results = checkTests(res.crawler.Results, res.tests)

	for _, res := range res.results {
		if res.Domain.Error != nil {
			out.Printf("{red}[FAILURE]{c} %5.1f/10 [code: ---] [%15s] [{cyan}  0 resources{c}] [{green}     0ms{c}] %s ({red}%s{c})\n", res.Score, res.Domain.Request.IP, res.Domain.Request.URL, res.Domain.Error)
		} else {
			url := res.Domain.Resource.Response.URL.String()
			if url != res.Domain.Request.URL {
				url = fmt.Sprintf("%s (result: %s)", res.Domain.Request.URL, url)
			}

			out.Printf("{green}[SUCCESS]{c} %5.1f/10 [code: {yellow}%d{c}] [%15s] [{cyan}%3d resources{c}] [{green}%6dms{c}] %s\n", res.Score, res.Domain.Resource.Response.Code, res.Domain.Request.IP, len(res.Domain.Resources), res.Domain.Resource.Time.Milli, url)
		}
	}

	for i := 0; i < len(res.results); i++ {
		if res.results[i].Domain.Error != nil {
			res.failed++
			continue
		}

		res.successful++
	}

	out.Printf("%d successful, %d failed\n", res.successful, res.failed)

	return res, nil
}
