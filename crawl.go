// Author: Liam Stanley <me@liamstanley.io>
// Docs: https://marill.liam.sh/
// Repo: https://github.com/lrstanley/marill

package main

import (
	"fmt"

	"github.com/lrstanley/marill/domfinder"
	"github.com/lrstanley/marill/scraper"
)

// Scan is a wrapper for all state related to the crawl process
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
	res.finder = &domfinder.Finder{Log: logger}

	if conf.scan.manualList != "" {
		logger.Println("manually supplied url list")
		res.crawler.Cnf.Domains, err = parseManualList()
		if err != nil {
			return nil, NewErr{Code: ErrDomains, deepErr: err}
		}
	} else {
		logger.Println("checking for running webservers")

		if err := res.finder.GetWebservers(); err != nil {
			return nil, NewErr{Code: ErrProcList, deepErr: err}
		}

		if outlist := ""; len(res.finder.Procs) > 0 {
			for _, proc := range res.finder.Procs {
				outlist += fmt.Sprintf("[%s:%s] ", proc.Name, proc.PID)
			}
			logger.Printf("found %d procs matching a webserver: %s", len(res.finder.Procs), outlist)
			out.Printf("found %d procs matching a webserver", len(res.finder.Procs))
		}

		// start crawling for domains
		if err := res.finder.GetDomains(); err != nil {
			return nil, NewErr{Code: ErrGetDomains, deepErr: err}
		}

		res.finder.Filter(domfinder.DomainFilter{
			IgnoreHTTP:  conf.scan.ignoreHTTP,
			IgnoreHTTPS: conf.scan.ignoreHTTPS,
			IgnoreMatch: conf.scan.ignoreMatch,
			MatchOnly:   conf.scan.matchOnly,
		})

		if len(res.finder.Domains) == 0 {
			return nil, NewErr{Code: ErrNoDomainsFound}
		}

		logger.Printf("found %d domains on webserver %s (exe: %s, pid: %s)", len(res.finder.Domains), res.finder.MainProc.Name, res.finder.MainProc.Exe, res.finder.MainProc.PID)

		for _, domain := range res.finder.Domains {
			res.crawler.Cnf.Domains = append(res.crawler.Cnf.Domains, &scraper.Domain{URL: domain.URL, IP: domain.IP})
		}
	}

	res.crawler.Cnf.Assets = conf.scan.assets
	res.crawler.Cnf.NoRemote = conf.scan.ignoreRemote
	res.crawler.Cnf.Delay = conf.scan.delay
	res.crawler.Cnf.AllowInsecure = conf.scan.allowInsecure
	res.crawler.Cnf.HTTPTimeout = conf.scan.httptimeout

	logger.Print("starting crawler...")
	out.Printf("starting scan on %d domains", len(res.crawler.Cnf.Domains))
	res.crawler.Crawl()
	out.Println("{lightgreen}scan complete{c}")

	// print out a fairly large amount of debugging information here
	for i := 0; i < len(res.crawler.Results); i++ {
		logger.Print(res.crawler.Results[i])

		for r := 0; r < len(res.crawler.Results[i].Assets); r++ {
			logger.Printf("%s => %s", res.crawler.Results[i].URL, res.crawler.Results[i].Assets[r])
		}
	}

	out.Println("{lightblue}starting tests{c}")
	res.results = checkTests(res.crawler.Results, res.tests)

	for i := 0; i < len(res.results); i++ {
		if res.results[i].Result.Error != nil {
			res.failed++
			continue
		}

		res.successful++
	}

	out.Printf("%d successful, %d failed", res.successful, res.failed)

	return res, nil
}
