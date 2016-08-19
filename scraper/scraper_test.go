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
		want bool
	}{
		{"https://liamstanley.io", "", false},
		{"https://liamstanley.io", "0.0.0.0", false},
		{"https://liamstanley.io", "000.000.00.0", false},
		{"https://liamstanley.io", "111.1111.11.1", false},
		{"https://liamstanley.io", "1.1.1.1.", false},
		{"https://google.com/", "", false},
		{"htps://google.com", "", true},
		{"http://liamstanley.io", "", false},
		{"http://liamstanley.io", "0.0.0.0", false},
		{"http://liamstanley.io", "000.000.00.0", false},
		{"http://liamstanley.io", "111.1111.11.1", false},
		{"http://liamstanley.io", "1.1.1.1.", false},
		{"http://some-domains-that-doesnt-exist.com", "", true},
		{"https://some-domains-that-doesnt-exist.com", "", true},
	}

	for _, c := range cases {
		got := FetchURL(c.in, "", logger)

		if got.Error != nil && !c.want {
			t.Errorf("fetchURL(%q, %q) == %q, wanted error: %v", c.in, c.inx, got.Error, c.want)
		}

		if got.Error == nil && c.want && got.Code == 200 {
			t.Errorf("fetchURL(%q, %q) == %q (%#v), though no errors", c.in, c.inx, got.Error, got)
		}
	}

	return
}