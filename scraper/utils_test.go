package scraper

import (
	"testing"
	"time"
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
		host, err := getHost(c.in)
		if err != nil {
			t.Fatalf("getHost(%q) == %q, wanted %q", c.in, err, c.want)
		}
		if host != c.want {
			t.Fatalf("getHost(%q) == %q, wanted %q", c.in, host, c.want)
		}
	}

	return
}

func TestTimer(t *testing.T) {
	tt := NewTimer()
	time.Sleep(1 * time.Second)
	tt.End()

	if tt.Result.Seconds != 1 {
		t.Fatalf("Timer ran for 1 second, but: %#v\n", tt.Result)
	}

	return
}
