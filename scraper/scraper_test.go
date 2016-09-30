// Author: Liam Stanley <me@liamstanley.io>
// Docs: https://marill.liam.sh/
// Repo: https://github.com/Liamraystanley/marill

package scraper

import (
	"log"
	"net/url"
	"os"
	"testing"
)

func TestFetch(t *testing.T) {
	logger := log.New(os.Stdout, "", log.LstdFlags)

	cases := []struct {
		in   string
		inx  string // extra -- e.g. ip
		want bool   // true == error
	}{
		{"https://liquidweb.com", "", false},
		{"https://lw.liam.sh", "0.0.0.0", true},             // invalid ip
		{"https://test.lw.liam.sh", "000.000.00.0", true},   // invalid ip
		{"https://test2.lw.liam.sh", "111.1111.11.1", true}, // invalid ip
		{"https://test3.lw.liam.sh", "1.1.1.1.", true},      // invalid ip
		{"https://google.com/", "", false},
		{"htps://google.com", "", true}, // invalid schema
		{"http://liamstanley.io", "", false},
		{"http://1.liamstanley.io", "0.0.0.0", true},                            // invalid ip
		{"http://2.liamstanley.io", "000.000.00.0", true},                       // invalid ip
		{"http://3.liamstanley.io", "111.1111.11.1", true},                      // invalid ip
		{"http://4.liamstanley.io", "1.1.1.1.", true},                           // invalid ip
		{"http://some-domains-that-doesnt-exist.com/x", "", true},               // invalid domain/path
		{"https://some-domains-that-doesnt-exist.com/x", "", true},              // invalid domain/path
		{"https://httpbin.org/redirect/10", "", true},                           // we allow max of 3 redirects
		{"https://httpbin.org/links/10", "", false},                             // provide some html links
		{"https://httpbin.org/html", "", false},                                 // return some html
		{"https://httpbin.org/drip?duration=5&numbytes=5&code=200", "", false},  // drip for 5 seconds
		{"https://httpbin.org/drip?duration=11&numbytes=5&code=200", "", false}, // drip for 11 seconds, 10s is our timeout
		{"https://httpbin.org/delay/12", "", true},                              // 10s is our timeout
		{"https://httpbin.org/delay/3", "", false},
	}

	tmplist := []*Domain{}
	for _, c := range cases {
		uri, _ := url.Parse(c.in)

		tmplist = append(tmplist, &Domain{URL: uri, IP: c.inx})
	}

	crawler := Crawler{Log: logger}
	crawler.Cnf.Domains = tmplist
	crawler.Crawl()

	for _, c := range cases {
		dom := crawler.GetResults(c.in, c.inx)

		if dom == nil && !c.want {
			t.Fatalf("crawler.GetResults(%q, %q) == nil, wanted results", c.in, c.inx)
		}

		if dom == nil && c.want {
			t.Fatalf("crawler.GetResults(%q, %q) == nil, wanted error", c.in, c.inx)
		}

		if dom.Error != nil && !c.want {
			t.Fatalf("Crawl of (%q, %q) == %s, didn't want an error", c.in, c.inx, dom)
		} else {
			continue
		}

		// only success tests below this

		if dom.Error == nil && c.want && dom.Response.Code == 200 {
			t.Fatalf("Crawl of (%q, %q) == %s, though no errors", c.in, c.inx, dom)
		}
	}

	return
}
