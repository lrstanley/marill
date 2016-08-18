package main

import (
	"fmt"
	"log"
	"runtime"

	df "github.com/Liamraystanley/marill/domfinder"
	"github.com/Liamraystanley/marill/scraper"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU() * 2)

	ps := df.GetProcs()
	for _, proc := range ps {
		fmt.Printf("%#v\n", proc)
	}

	domains, err := df.GetDomains(ps)

	if err != nil {
		log.Fatal(err)
	}

	tmplist := []*scraper.Domain{}
	for _, domain := range domains {
		tmplist = append(tmplist, &scraper.Domain{URL: domain.URL, IP: domain.IP})
	}
	scraper.Crawl(tmplist)
}
