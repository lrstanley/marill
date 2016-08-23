package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/Liamraystanley/marill/domfinder"
	"github.com/Liamraystanley/marill/scraper"
)

func main() {
	// initialize the logger, just to stdout for now, in the future we will want to
	// provide users the option to choose the path they would like to log to. Can
	// also implement io.MultiWriter?
	// initLoggerToFile("marill.log")
	initLogger(os.Stdout)
	defer closeLogger() // ensure we're cleaning up the logger

	logger.Println("initializing logger")

	runtime.GOMAXPROCS(runtime.NumCPU() * 2)
	logger.Printf("limiting max threads to %d", runtime.NumCPU()*2)

	logger.Println("checking for running webservers...")

	finder := &domfinder.Finder{Log: logger}
	if err := finder.GetWebservers(); err != nil {
		logger.Fatalf("unable to get process list: %s", err)
	}

	if out := ""; len(finder.Procs) > 0 {
		for _, proc := range finder.Procs {
			out += fmt.Sprintf("[%s:%s] ", proc.Name, proc.PID)
		}
		logger.Printf("found %d procs matching a webserver: %s", len(finder.Procs), out)
	}

	// start crawling for domains
	if err := finder.GetDomains(); err != nil {
		logger.Fatalf("unable to auto-fetch domain list: %s", err)
	}

	logger.Printf("found %d domains on webserver %s (exe: %s, pid: %s)", len(finder.Domains), finder.MainProc.Name, finder.MainProc.Exe, finder.MainProc.PID)

	tmplist := []*scraper.Domain{}
	for _, domain := range finder.Domains {
		tmplist = append(tmplist, &scraper.Domain{URL: domain.URL, IP: domain.IP})
	}
	crawler := &scraper.Crawler{Log: logger, Domains: tmplist}
	crawler.Crawl()
}
