// Author: Liam Stanley <me@liamstanley.io>
// Docs: https://marill.liam.sh/
// Repo: https://github.com/lrstanley/marill

package utils

import (
	"os"
	"strings"
	"testing"
)

func TestGetHost(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"http://domain.com/", "domain.com"},
		{"https://domain.com/", "domain.com"},
		{"https://domain.com/path", "domain.com"},
		{"http://1.2.3.4", "1.2.3.4"},
	}

	for _, c := range cases {
		host, err := GetHost(c.in)
		if err != nil {
			t.Fatalf("getHost(%q) == %q, wanted %q", c.in, err, c.want)
		}
		if host != c.want {
			t.Fatalf("getHost(%q) == %q, wanted %q", c.in, host, c.want)
		}
	}

	return
}

func TestGetHostname(t *testing.T) {
	host, err := os.Hostname()
	if err != nil {
		t.Fatalf("os.Hostname() returned: %q", err)
	}

	newHost := GetHostname()

	if !strings.HasPrefix(newHost, host) {
		t.Fatalf("getHostname() == %q, wanted prefix: %q", newHost, host)
	}

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
		{host: "domain.com", port: "", want: "http://domain.com"},
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
