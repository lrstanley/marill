package domfinder

import (
	"log"
	"testing"
)

func TestFetch(t *testing.T) {
	ps := GetProcs()

	if len(ps) > 0 {
		for _, proc := range ps {
			if !webservers[proc.Name] {
				log.Fatalf("GetProcs() returned len %d, proc %#v not webserver", len(ps), proc)
			}
		}
	}

	cases := []*Process{
		{
			PID:  "1",
			Name: "test",
			Exe:  "/usr/bin/doesntexist",
		},
		{
			PID:  "1",
			Name: "httpdtest",
			Exe:  "/usr/sbin/httpd", // may exist, but the name doesn't match
		},
	}

	// these should fail with ErrNotImplemented
	for _, c := range cases {
		proclist := []*Process{c}
		proc, domains, err := GetDomains(proclist)

		if err == nil {
			t.Fatalf("GetDomains(%#v) should have failed but got: (%#v :: %#v :: %q)", proclist, proc, domains, err)
		}

		if err.GetCode() != ErrNotImplemented {
			t.Fatalf("GetDomains(%#v) should have returned %v but returned %v", proclist, &NewErr{Code: ErrNotImplemented, value: c.Name}, err)
		}
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
		{host: "domain.com", port: "", want: ""},
	}

	for _, c := range cases {
		uri, err := isDomainURL(c.host, c.port)

		if err != nil && c.want != "" {
			t.Fatalf("isDomainURL(%s, %s) returned error %v", c.host, c.port, err)
		}

		if err == nil && c.want == "" {
			t.Fatalf("isDomainURL(%s, %s) returned no errors", c.host, c.port)
		}

		if err == nil && len(c.want) > 0 {
			if uri.String() != c.want {
				t.Fatalf("isDomainURL(%s, %s) returned no errors. wanted: %s, got: %s", c.host, c.port, c.want, uri.String())
			}
		}
	}

	return
}
