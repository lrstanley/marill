// Author: Liam Stanley <me@liamstanley.io>
// Docs: https://marill.liam.sh/
// Repo: https://github.com/lrstanley/marill

package scraper

import (
	"testing"

	"github.com/lrstanley/marill/utils"
)

func TestFmtTagLinks(t *testing.T) {
	cases := []struct {
		uri  string
		in   string
		want string // if ""; assume error
	}{
		// absolute
		{"http://example.com/", "http://example.com/test.css", "http://example.com/test.css"},
		{"https://example.com/", "http://example.com/test.css", "http://example.com/test.css"},
		{"http://example.com/", "https://example.com/test.css", "https://example.com/test.css"},
		// absolute remote
		{"http://example.com/", "//example1.com/test.css", "http://example1.com/test.css"},
		{"https://example.com/", "//example1.com/test.css", "https://example1.com/test.css"},
		// relatively absolute
		{"http://example.com/", "/test.css", "http://example.com/test.css"},
		{"http://example.com/test/", "/test.css", "http://example.com/test.css"},
		{"http://example.com/", "./test.css", "http://example.com/test.css"},
		{"http://example.com/test", "./test.css", "http://example.com/test.css"},
		{"http://example.com/test/", "./test.css", "http://example.com/test.css"},
		// relative
		{"http://example.com/", "test.css", "http://example.com/test.css"},

		// some erronous ones
		{"http://example.com/", "", ""},
		{"http://example.com/", "ht://example.com/test.css", ""},
		{"http://example.com/", "ftp://example.com/test.css", ""},
		{"http://example.com/", "://example.com/test.css", ""},
	}

	for _, c := range cases {
		uri := utils.MustURL(c.uri, "")
		out := fmtTagLinks(c.in, uri)

		if out != c.want {
			t.Fatalf("fmtTagLinks(%q, %q) == %q, wanted: %q", c.in, c.uri, out, c.want)
		}
	}
}
