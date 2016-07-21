package main

import (
	"os"
	"runtime"
	"sync"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	urlsToCheck := os.Args[1:]
	results := []*Results{}
	var wg sync.WaitGroup

	// loop through all supplied urls and send them to a worker to be fetched
	for _, url := range urlsToCheck {
		wg.Add(1)

		go func(url string) {
			defer wg.Done()

			results = append(results, Crawl(url, ""))
		}(url)
	}

	// wait for all workers to complete their tasks
	wg.Wait()

	// print out cool stuff here based on results
}
