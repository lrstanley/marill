package scraper

import (
	"log"
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
		{"https://liamstanley.io", "", false},
		{"https://liamstanley.io", "0.0.0.0", true},       // invalid ip
		{"https://liamstanley.io", "000.000.00.0", true},  // invalid ip
		{"https://liamstanley.io", "111.1111.11.1", true}, // invalid ip
		{"https://liamstanley.io", "1.1.1.1.", true},      // invalid ip
		{"https://google.com/", "", false},
		{"htps://google.com", "", true}, // invalid schema
		{"http://liamstanley.io", "", false},
		{"http://liamstanley.io", "0.0.0.0", true},                              // invalid ip
		{"http://liamstanley.io", "000.000.00.0", true},                         // invalid ip
		{"http://liamstanley.io", "111.1111.11.1", true},                        // invalid ip
		{"http://liamstanley.io", "1.1.1.1.", true},                             // invalid ip
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

	for _, c := range cases {
		got := FetchURL(c.in, c.inx, logger)

		if got.Error != nil && !c.want {
			t.Errorf("fetchURL(%q, %q) == %q, wanted error: %v", c.in, c.inx, got.Error, c.want)
		}

		if got.Error == nil && c.want && got.Code == 200 {
			t.Errorf("fetchURL(%q, %q) == %q (%#v), though no errors", c.in, c.inx, got.Error, got)
		}
	}

	return
}
