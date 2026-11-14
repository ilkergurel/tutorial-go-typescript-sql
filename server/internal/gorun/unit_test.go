// unit_test.go: gorun's pure helpers (output cap, ts-test limits and
// environment). No build tag: they need no toolchain, so go test ./... runs
// them; everything that starts a process is in the integration files.
package gorun

import (
	"slices"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

const truncMark = "\n[output truncated]"

// The cap cuts on a character boundary, also when the character started in
// an earlier Write: half a ğ would reach the browser as U+FFFD.
func TestCappedBufferCutsOnCharacterBoundary(t *testing.T) {
	tests := []struct {
		name   string
		cap    int
		writes []string
		want   string
	}{
		{"fits", 8, []string{"Ağrı"}, "Ağrı"},
		{"cut inside ğ", 2, []string{"Ağrı"}, "A" + truncMark},
		{"cut after a whole ğ", 3, []string{"Ağrı"}, "Ağ" + truncMark},
		{"İ split across writes", 3, []string{"ab\xc4", "\xb0c"}, "ab" + truncMark},
		{"cap reached on a whole ş", 3, []string{"aş", "b"}, "aş" + truncMark},
		{"four-byte character", 3, []string{"a🙂"}, "a" + truncMark},
		{"later writes discarded", 2, []string{"şş", "ş", "ş"}, "ş" + truncMark},
		{"invalid bytes are cut where they are", 2, []string{"\xff\xfe\xfd"}, "\xff\xfe" + truncMark},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cb := cappedBuffer{cap: tc.cap}
			for _, w := range tc.writes {
				if n, err := cb.Write([]byte(w)); n != len(w) || err != nil {
					t.Fatalf("Write(%q) = %d, %v; want %d, nil", w, n, err, len(w))
				}
			}
			if got := cb.String(); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
			if cb.truncated != strings.HasSuffix(tc.want, truncMark) {
				t.Errorf("truncated = %v", cb.truncated)
			}
		})
	}
}

func TestTruncateKeepsCharactersWhole(t *testing.T) {
	if got := truncate("İstanbul"); got != "İstanbul" {
		t.Fatalf("short output changed: %q", got)
	}
	// "a" then two-byte İs: byte OutputCap-1 is the first half of an İ.
	got := truncate("a" + strings.Repeat("İ", OutputCap))
	if !utf8.ValidString(got) || !strings.HasSuffix(got, truncMark) || len(got) != OutputCap-1+len(truncMark) {
		t.Fatalf("len %d, valid UTF-8 %v, ends %q", len(got), utf8.ValidString(got), got[len(got)-24:])
	}
}

// ts-test honors timeout_s like the go runtime; without it, the maximum.
func TestTSTestTTL(t *testing.T) {
	for timeoutS, want := range map[int]time.Duration{0: maxRunTTL, 3: 3 * time.Second, 15: maxRunTTL, 99: maxRunTTL} {
		if got := testTTL(Request{TimeoutS: timeoutS}); got != want {
			t.Errorf("testTTL(timeout_s %d) = %v, want %v", timeoutS, got, want)
		}
	}
}

// An inherited NO_COLOR or FORCE_COLOR (any case: Windows ignores it) must
// not reach the tool next to the switch ts-test sets.
func TestWithoutEnv(t *testing.T) {
	env := []string{"PATH=/bin", "NO_COLOR=1", "force_color=1", "NO_COLOR_X=1", "=C:=C:\\"}
	got := withoutEnv(env, "NO_COLOR", "FORCE_COLOR")
	if want := []string{"PATH=/bin", "NO_COLOR_X=1", "=C:=C:\\"}; !slices.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}
