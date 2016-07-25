package main

import (
	"fmt"
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

			fmt.Printf("[START] Scanning %s\n", url)
			result := Crawl(url, "")
			results = append(results, result)

			fmt.Printf("[%s] %+v\n", result.connHostname, result)

			for i := range result.Resources {
				fmt.Printf("[%s] %+v\n", result.Resources[i].connHostname, result.Resources[i])
			}

			fmt.Printf("[FINISHED] Scanned %s\n", url)
		}(url)
	}

	// wait for all workers to complete their tasks
	wg.Wait()

	// print out cool stuff here based on results
}
