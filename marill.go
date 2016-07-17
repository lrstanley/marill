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

	for _, url := range urlsToCheck {
		wg.Add(1)

		go func(url string) {
			defer wg.Done()

			results = append(results, Crawl(url))
		}(url)
	}

	wg.Wait()

	// print out cool stuff here based on results
}
