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
}
