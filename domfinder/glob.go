package domfinder

import "strings"

// GLOB is the character which is treaded like a glob
const GLOB = "*"

// Glob will test a string pattern, potentially containing globs, against a string.
func Glob(subj, pattern string) bool {
	// empty pattern
	if pattern == "" {
		return subj == pattern
	}

	// if a glob, match all
	if pattern == GLOB {
		return true
	}

	parts := strings.Split(pattern, GLOB)

	if len(parts) == 1 {
		// no globs, test for equality
		return subj == pattern
	}

	leadingGlob, trailingGlob := strings.HasPrefix(pattern, GLOB), strings.HasSuffix(pattern, GLOB)
	last := len(parts) - 1

	// check prefix first
	if !leadingGlob && !strings.HasPrefix(subj, parts[0]) {
		return false
	}

	// check middle section
	for i := 1; i < last; i++ {
		if !strings.Contains(subj, parts[i]) {
			return false
		}

		// trim already-evaluated text from subj during loop over pattern
		idx := strings.Index(subj, parts[i]) + len(parts[i])
		subj = subj[idx:]
	}

	// check suffix last
	return trailingGlob || strings.HasSuffix(subj, parts[last])
}
