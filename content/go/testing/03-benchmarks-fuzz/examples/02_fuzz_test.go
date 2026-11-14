package main

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// Truncate shortens s to at most n bytes without splitting a character,
// e.g. to fit a column whose limit is in bytes.
func Truncate(s string, n int) string {
	if n <= 0 {
		return "" // the fuzzer found Truncate("0", -42) panicking
	}
	if len(s) <= n {
		return s
	}
	for n > 0 && !utf8.RuneStart(s[n]) {
		n-- // back up to the first byte of the character that doesn't fit
	}
	return s[:n]
}

func FuzzTruncate(f *testing.F) {
	f.Add("Iğdır", 2) // seeds: run by plain go test, mutated by go test -fuzz
	f.Add("şçöğü", 5)
	f.Add("", 0)
	f.Fuzz(func(t *testing.T, s string, n int) {
		got := Truncate(s, n)
		// Properties, not examples: true for every input.
		if len(got) > max(n, 0) || !strings.HasPrefix(s, got) {
			t.Fatalf("Truncate(%q, %d) = %q: too long or not a prefix", s, n, got)
		}
		if utf8.ValidString(s) && !utf8.ValidString(got) {
			t.Fatalf("Truncate(%q, %d) = %q: split a character", s, n, got)
		}
		t.Logf("Truncate(%q, %d) = %q", s, n, got)
	})
}
