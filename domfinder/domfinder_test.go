package domfinder

import "testing"

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
