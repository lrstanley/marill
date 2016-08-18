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

	logger.Println("Initializing logger")

	runtime.GOMAXPROCS(runtime.NumCPU() * 2)
	logger.Printf("Limiting max threads to %d", runtime.NumCPU()*2)

	logger.Println("Checking for running webservers...")
	ps := domfinder.GetProcs()

	if out := ""; len(ps) > 0 {
		for _, proc := range ps {
			out += fmt.Sprintf("[%s:%s] ", proc.Name, proc.PID)
		}
		logger.Printf("Found %d procs matching a webserver: %s", len(ps), out)
	}

	ws, domains, err := domfinder.GetDomains(ps)
	logger.Printf("Found %d domains on webserver %s (exe: %s, pid: %s)", len(domains), ws.Name, ws.Exe, ws.PID)

	if err != nil {
		logger.Fatal(err)
	}

	tmplist := []*scraper.Domain{}
	for _, domain := range domains {
		tmplist = append(tmplist, &scraper.Domain{URL: domain.URL, IP: domain.IP})
	}
	scraper.Crawl(tmplist)
}
