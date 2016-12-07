// Author: Liam Stanley <me@liamstanley.io>
// Docs: https://marill.liam.sh/
// Repo: https://github.com/lrstanley/marill

package domfinder

import (
	"strings"
	"testing"

	"github.com/lrstanley/marill/utils"
)

func TestStripDups(t *testing.T) {
	pprint := func(domains []*Domain) string {
		out := make([]string, len(domains))

		for i, dom := range domains {
			out[i] = dom.String()
		}

		return strings.Join(out, ", ")
	}

	run := func(domains []*Domain, length int) {
		orig := domains
		if stripDups(&domains); len(domains) != length {
			t.Fatalf("stripDups(%s) == %s, wanted %d", pprint(orig), pprint(domains), length)
		}
	}

	// regular duplicate
	run([]*Domain{
		{IP: "1.2.3.4", Port: "80", URL: utils.MustURL("domain.com", "80")},
		{IP: "1.2.3.4", Port: "80", URL: utils.MustURL("domain.com", "80")},
	}, 1)

	// multiple duplicates
	run([]*Domain{
		{IP: "1.2.3.4", Port: "80", URL: utils.MustURL("domain.com", "80")},
		{IP: "1.2.3.4", Port: "80", URL: utils.MustURL("domain.com", "80")},
		{IP: "1.2.3.4", Port: "80", URL: utils.MustURL("domain.com", "80")},
		{IP: "1.2.3.4", Port: "80", URL: utils.MustURL("domain.com", "80")},
	}, 1)

	// "some" duplicates
	run([]*Domain{
		{IP: "1.2.3.4", Port: "80", URL: utils.MustURL("domain.com", "80")},
		{IP: "1.2.3.4", Port: "80", URL: utils.MustURL("domain1.com", "80")},
		{IP: "1.2.3.4", Port: "80", URL: utils.MustURL("domain1.com", "80")},
		{IP: "1.2.3.4", Port: "80", URL: utils.MustURL("domain.com", "80")},
	}, 2)

	// different ports
	run([]*Domain{
		{IP: "1.2.3.4", Port: "80", URL: utils.MustURL("domain.com", "80")},
		{IP: "1.2.3.4", Port: "8080", URL: utils.MustURL("domain.com", "8080")},
	}, 2)

	// different IPs
	run([]*Domain{
		{IP: "1.2.3.4", Port: "80", URL: utils.MustURL("domain.com", "80")},
		{IP: "1.2.3.5", Port: "80", URL: utils.MustURL("domain.com", "80")},
	}, 1)

	// different domains
	run([]*Domain{
		{IP: "1.2.3.4", Port: "80", URL: utils.MustURL("domain.com", "80")},
		{IP: "1.2.3.4", Port: "80", URL: utils.MustURL("domain1.com", "80")},
	}, 2)

	return
}
