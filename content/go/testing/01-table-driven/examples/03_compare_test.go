package main

import (
	"reflect"
	"slices"
	"strings"
	"testing"
	"unicode"
)

// Words lowercases Turkish text and splits it into words.
func Words(s string) []string {
	return strings.Fields(strings.ToLowerSpecial(unicode.TurkishCase, s))
}

func TestWords(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{"İSTANBUL  Iğdır", []string{"istanbul", "ığdır"}},
		{"ILIK su", []string{"ılık", "su"}},
		{"   ", nil},
	}
	for _, tt := range tests {
		got := Words(tt.in)
		if !slices.Equal(got, tt.want) { // element-wise ==; nil equals empty
			t.Errorf("Words(%q) = %+q, want %+q", tt.in, got, tt.want)
		}
	}
}

func TestNilVersusEmpty(t *testing.T) {
	got := Words("   ")
	t.Logf("got = %#v, got == nil: %v", got, got == nil)
	t.Logf("reflect.DeepEqual(got, nil) = %v", reflect.DeepEqual(got, []string(nil)))
	t.Logf("slices.Equal(got, nil)      = %v", slices.Equal(got, nil))
}

func TestInvisibleDifference(t *testing.T) {
	fromJS := "i\u0307stanbul" // "İSTANBUL".toLowerCase() in JavaScript
	fromGo := Words("İSTANBUL")[0]
	t.Logf("equal: %v", fromJS == fromGo)
	t.Logf("%%q:  %q vs %q", fromJS, fromGo)
	t.Logf("%%+q: %+q vs %+q", fromJS, fromGo)
}
