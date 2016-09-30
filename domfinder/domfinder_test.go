// Author: Liam Stanley <me@liamstanley.io>
// Docs: https://marill.liam.sh/
// Repo: https://github.com/Liamraystanley/marill

package domfinder

import (
	"strings"
	"testing"

	"github.com/Liamraystanley/marill/utils"
)

func TestFilter(t *testing.T) {
	pprint := func(domains []*Domain) string {
		out := make([]string, len(domains))

		for i, dom := range domains {
			out[i] = dom.String()
		}

		return strings.Join(out, ", ")
	}

	run := func(domains []*Domain, cnf DomainFilter, length int) {
		f := &Finder{Domains: domains}
		if f.Filter(cnf); len(f.Domains) != length {
			t.Fatalf("Filter(%s) == %s, wanted %d", pprint(domains), pprint(f.Domains), length)
		}
	}

	// should match all
	run([]*Domain{
		{IP: "1.2.3.4", Port: "80", URL: utils.MustURL("domain.com", "80")},
		{IP: "1.2.3.4", Port: "80", URL: utils.MustURL("domain.com", "80")},
	}, DomainFilter{IgnoreHTTP: true}, 0)

	// should match none
	run([]*Domain{
		{IP: "1.2.3.4", Port: "80", URL: utils.MustURL("domain.com", "80")},
		{IP: "1.2.3.4", Port: "80", URL: utils.MustURL("domain.com", "80")},
	}, DomainFilter{IgnoreHTTPS: true}, 2)

	// should match one
	run([]*Domain{
		{IP: "1.2.3.4", Port: "80", URL: utils.MustURL("domain.com", "80")},
		{IP: "1.2.3.4", Port: "443", URL: utils.MustURL("domain.com", "443")},
	}, DomainFilter{IgnoreHTTPS: true}, 1)

	// should match * (all)
	run([]*Domain{
		{IP: "1.2.3.4", Port: "80", URL: utils.MustURL("domain.com", "80")},
		{IP: "1.2.3.4", Port: "443", URL: utils.MustURL("domain.com", "443")},
	}, DomainFilter{MatchOnly: "*"}, 2)

	// should match none
	run([]*Domain{
		{IP: "1.2.3.4", Port: "80", URL: utils.MustURL("domain.com", "80")},
		{IP: "1.2.3.4", Port: "443", URL: utils.MustURL("domain.com", "443")},
	}, DomainFilter{IgnoreMatch: "*"}, 0)

	// should match none
	run([]*Domain{
		{IP: "1.2.3.4", Port: "80", URL: utils.MustURL("domain.com", "80")},
		{IP: "1.2.3.4", Port: "443", URL: utils.MustURL("domain.com", "443")},
	}, DomainFilter{IgnoreMatch: "*domain.com*"}, 0)

	// should match one
	run([]*Domain{
		{IP: "1.2.3.4", Port: "80", URL: utils.MustURL("domain.com", "80")},
		{IP: "1.2.3.4", Port: "443", URL: utils.MustURL("domain.com", "443")},
	}, DomainFilter{IgnoreMatch: "https://*"}, 1)

	return
}
