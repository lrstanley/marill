// Author: Liam Stanley <me@liamstanley.io>
// Docs: https://marill.liam.sh/
// Repo: https://github.com/lrstanley/marill

package utils

import (
	"testing"
	"time"
)

func TestTimer(t *testing.T) {
	tt := NewTimer()
	time.Sleep(1 * time.Second)
	tt.End()

	if tt.Result.Seconds != 1 {
		t.Fatalf("Timer ran for 1 second, but: %#v\n", tt.Result)
	}

	return
}
