package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"
)

var ErrBadAmount = errors.New("bad amount")

// ParseKurus parses a lira amount written the Turkish way, "1.234,56",
// into kuruş (hundredths), so money never touches a float.
func ParseKurus(s string) (int64, error) {
	whole, frac, _ := strings.Cut(s, ",")
	whole = strings.ReplaceAll(whole, ".", "") // thousands separators
	if whole == "" || len(frac) > 2 {
		return 0, fmt.Errorf("%w: %q", ErrBadAmount, s)
	}
	frac = (frac + "00")[:2] // ",5" is 50 kuruş
	n, err := strconv.ParseUint(whole+frac, 10, 63)
	if err != nil {
		return 0, fmt.Errorf("%w: %q", ErrBadAmount, s)
	}
	return int64(n), nil
}

func TestParseKurus(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want int64
		err  error
	}{
		{"whole lira", "12", 1200, nil},
		{"one decimal", "12,5", 1250, nil},
		{"thousands", "1.234,56", 123456, nil},
		{"only kuruş", "0,07", 7, nil},
		{"empty", "", 0, ErrBadAmount},
		{"english decimal point", "1,234", 0, ErrBadAmount},
		{"negative", "-5", 0, ErrBadAmount},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel() // safe: since Go 1.22 each iteration has its own tt
			got, err := ParseKurus(tt.in)
			if !errors.Is(err, tt.err) {
				t.Fatalf("ParseKurus(%q) error = %v, want %v", tt.in, err, tt.err)
			}
			if got != tt.want {
				t.Errorf("ParseKurus(%q) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}
