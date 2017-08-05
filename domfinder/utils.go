// Author: Liam Stanley <me@liamstanley.io>
// Docs: https://marill.liam.sh/
// Repo: https://github.com/lrstanley/marill

package domfinder

import (
	"strings"
)

// stripDups strips all domains that have the same resulting URL
func stripDups(domains *[]*Domain) {
	var tmp []*Domain

	for _, dom := range *domains {
		var isIn bool
		for _, other := range tmp {
			if dom.URL.String() == other.URL.String() {
				isIn = true
				break
			}
		}
		if !isIn {
			tmp = append(tmp, dom)
		}
	}

	*domains = tmp

	return
}

var predefined = [...]string{
	"cpanel", "webmail", "mail", "whm", "cpcalendars", "cpcontacts",
	"_wildcard_",
}

func stripPredefined(domains *[]*Domain) {
	var tmp []*Domain

	for _, dom := range *domains {
		var in bool
		for i := 0; i < len(predefined); i++ {
			if strings.HasPrefix(dom.URL.Hostname(), predefined[i]+".") {
				in = true
				break
			}
		}

		if !in {
			tmp = append(tmp, dom)
		}
	}

	*domains = tmp
}
