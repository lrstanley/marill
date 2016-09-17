package domfinder

import (
	"os"
	"strings"
	"testing"
)

func TestGetHostname(t *testing.T) {
	host, err := os.Hostname()
	if err != nil {
		t.Fatalf("os.Hostname() returned: %q", err)
	}

	newHost := getHostname()

	if !strings.HasPrefix(newHost, host) {
		t.Fatalf("getHostname() == %q, wanted prefix: %q", newHost, host)
	}

	return
}

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
		{IP: "1.2.3.4", Port: "80", URL: MustURL("domain.com", "80")},
		{IP: "1.2.3.4", Port: "80", URL: MustURL("domain.com", "80")},
	}, 1)

	// multiple duplicates
	run([]*Domain{
		{IP: "1.2.3.4", Port: "80", URL: MustURL("domain.com", "80")},
		{IP: "1.2.3.4", Port: "80", URL: MustURL("domain.com", "80")},
		{IP: "1.2.3.4", Port: "80", URL: MustURL("domain.com", "80")},
		{IP: "1.2.3.4", Port: "80", URL: MustURL("domain.com", "80")},
	}, 1)

	// "some" duplicates
	run([]*Domain{
		{IP: "1.2.3.4", Port: "80", URL: MustURL("domain.com", "80")},
		{IP: "1.2.3.4", Port: "80", URL: MustURL("domain1.com", "80")},
		{IP: "1.2.3.4", Port: "80", URL: MustURL("domain1.com", "80")},
		{IP: "1.2.3.4", Port: "80", URL: MustURL("domain.com", "80")},
	}, 2)

	// different ports
	run([]*Domain{
		{IP: "1.2.3.4", Port: "80", URL: MustURL("domain.com", "80")},
		{IP: "1.2.3.4", Port: "8080", URL: MustURL("domain.com", "8080")},
	}, 2)

	// different IPs
	run([]*Domain{
		{IP: "1.2.3.4", Port: "80", URL: MustURL("domain.com", "80")},
		{IP: "1.2.3.5", Port: "80", URL: MustURL("domain.com", "80")},
	}, 1)

	// different domains
	run([]*Domain{
		{IP: "1.2.3.4", Port: "80", URL: MustURL("domain.com", "80")},
		{IP: "1.2.3.4", Port: "80", URL: MustURL("domain1.com", "80")},
	}, 2)

	return
}

func TestIsDomainURL(t *testing.T) {
	cases := []struct {
		host string // host uri
		port string // port -- should always be supplied, but may not be valid
		want string // intended string return
	}{
		{host: "d omain .com", port: "80", want: ""},
		{host: "d omain .com", port: "443", want: ""},
		{host: "d omain .com", port: "12345", want: ""},
		{host: "domain.com", port: "80", want: "http://domain.com"},
		{host: "domain.com", port: "443", want: "https://domain.com"},
		{host: "domain.com", port: "8080", want: "http://domain.com:8080"},
		{host: "domain.com", port: "0123", want: ""},
		{host: "domain.com", port: "", want: ""},
	}

	for _, c := range cases {
		uri, err := IsDomainURL(c.host, c.port)

		if err != nil && c.want != "" {
			t.Fatalf("IsDomainURL(%s, %s) returned error %v", c.host, c.port, err)
		}

		if err == nil && c.want == "" {
			t.Fatalf("IsDomainURL(%s, %s) returned no errors", c.host, c.port)
		}

		if err == nil && len(c.want) > 0 {
			if uri.String() != c.want {
				t.Fatalf("IsDomainURL(%s, %s) returned no errors. wanted: %s, got: %s", c.host, c.port, c.want, uri.String())
			}
		}
	}

	return
}
