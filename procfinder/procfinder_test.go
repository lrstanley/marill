package procfinder

import (
	"reflect"
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
