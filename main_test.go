package main

import (
	"testing"

	"./scraper"
)

func TestFetch(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"https://liamstanley.io", false},
		{"https://google.com/", false},
		{"htps://google.com", true},
		{"http://liamstanley.io", false},
		{"http://some-domains-that-doesnt-exist.com", true},
		{"https://some-domains-that-doesnt-exist.com", true},
	}

	for _, c := range cases {
		got := scraper.Crawl(c.in, "")

		if got.Error != nil && !c.want {
			t.Errorf("scraper.Crawl(%q) == %q, wanted error: %v", c.in, got.Error, c.want)
		}

		if got.Error == nil && c.want {
			t.Errorf("scraper.Crawl(%s) == %q (%q), though no errors", c.in, got.Error, got)
		}
	}

	return
}
