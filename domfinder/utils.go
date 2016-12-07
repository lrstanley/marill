// Author: Liam Stanley <me@liamstanley.io>
// Docs: https://marill.liam.sh/
// Repo: https://github.com/lrstanley/marill

package domfinder

// stripDups strips all domains that have the same resulting URL
func stripDups(domains *[]*Domain) {
	var tmp []*Domain

	for _, dom := range *domains {
		isIn := false
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
