package main

import (
	"fmt"
	"log"
	"runtime"
	"sync"

	df "github.com/Liamraystanley/marill/domfinder"
	"github.com/Liamraystanley/marill/scraper"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	ps := df.GetProcs()
	for _, proc := range ps {
		fmt.Printf("%#v\n", proc)
	}

	domains, err := df.GetDomains(ps)

	if err != nil {
		log.Fatal(err)
	}

	results := []*scraper.Results{}
	var wg sync.WaitGroup

	// loop through all supplied urls and send them to a worker to be fetched
	for _, domain := range domains {
		wg.Add(1)

		go func(domain *df.Domain) {
			defer wg.Done()

			fmt.Printf("[\033[1;36m---\033[0;m] [\033[0;32m------\033[0;m] \033[0;95mStarting to scan %s\033[0;m\n", domain.URL.String())
			result := scraper.Crawl(domain.URL.String(), "")
			results = append(results, result)

			fmt.Printf("[\033[1;36m---\033[0;m] [\033[0;32m%4dms\033[0;m] \033[0;95mFinished scanning %s\033[0;m\n", result.TotalTime.Milli, domain.URL.String())
		}(domain)
	}

	// wait for all workers to complete their tasks
	wg.Wait()

	// print out cool stuff here based on results
}
