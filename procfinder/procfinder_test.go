package procfinder

import (
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestRemoveEmpty(t *testing.T) {
	cases := []struct {
		in   []string
		want []string
	}{
		{in: []string{"test", "", "test"}, want: []string{"test", "test"}},
		{in: []string{"test", "test", ""}, want: []string{"test", "test"}},
		{in: []string{"", "test", "test"}, want: []string{"test", "test"}},
		{in: []string{"test", "test", "test"}, want: []string{"test", "test", "test"}},
	}

	for _, c := range cases {
		out := removeEmpty(c.in)
		if !reflect.DeepEqual(out, c.want) {
			t.Fatalf("remoteEmpty(%q) == %#v, wanted %#v\n", c.in, out, c.want)
		}
	}

	return
}

func TestHexToDec(t *testing.T) {
	cases := []struct {
		in   string
		want int64
	}{
		{in: "DF10", want: 57104},
		{in: "34", want: 52},
		{in: "01BB", want: 443},
		{in: "E6", want: 230},
		{in: "0", want: 0},           // should fail with 0
		{in: "1000000AAAA", want: 0}, // should fail with 0
	}

	for _, c := range cases {
		out := hexToDec(c.in)

		if out != c.want {
			t.Fatalf("hexToDec(%q) == %q, wanted %q", c.in, out, c.want)
		}
	}

	return
}

func TestIP(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{in: "0100007F", want: "127.0.0.1"},
		{in: "00000000", want: "0.0.0.0"},
		{in: "8706140A", want: "10.20.6.135"},
		{in: "069AA1C0", want: "192.161.154.6"},
		{in: "A0AB3448", want: "72.52.171.160"},
		{in: "111111111111111111111111111111111111", want: "0.0.0.0"}, // should fail
		{in: "", want: "0.0.0.0"},                                     // should fail
	}

	for _, c := range cases {
		out := ip(c.in)

		if out != c.want {
			t.Fatalf("ip(%q) == %q, wanted %q", c.in, out, c.want)
		}
	}

	return
}

func TestGetProcessExe(t *testing.T) {
	// proc := os.Args[0]
	pid := strconv.Itoa(os.Getppid())
	out := getProcessExe(pid)

	if !strings.HasSuffix(out, "/go") {
		t.Fatalf("getProcessExe(%q) == %q, not go", pid, out)
	}

	return
}
