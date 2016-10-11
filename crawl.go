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
			return nil, NewErr{Code: ErrDomains, deepErr: err}
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
			out.Printf("found %d procs matching a webserver", len(finder.Procs))
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

	res.crawler.Cnf.Resources = conf.scan.resources
	res.crawler.Cnf.NoRemote = conf.scan.ignoreRemote
	res.crawler.Cnf.Delay = conf.scan.delay
	res.crawler.Cnf.AllowInsecure = conf.scan.allowInsecure

	logger.Print("starting crawler...")
	out.Printf("starting scan on %d domains", len(res.crawler.Cnf.Domains))
	res.crawler.Crawl()
	out.Println("{lightgreen}scan complete{c}")

	// print out a fairly large amount of debugging information here
	for i := 0; i < len(res.crawler.Results); i++ {
		logger.Print(res.crawler.Results[i])

		for r := 0; r < len(res.crawler.Results[i].Resources); r++ {
			logger.Printf("%s => %s", res.crawler.Results[i].URL, res.crawler.Results[i].Resources[r])
		}
	}

	out.Println("{lightblue}starting tests{c}")
	res.results = checkTests(res.crawler.Results, res.tests)

	for i := 0; i < len(res.results); i++ {
		if res.results[i].Domain.Error != nil {
			res.failed++
			continue
		}

		res.successful++
	}

	out.Printf("%d successful, %d failed", res.successful, res.failed)

	return res, nil
}
