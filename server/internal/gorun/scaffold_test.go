package gorun

import (
	"os"
	"regexp"
	"testing"
)

// The pgx lesson module must select exactly the versions the backend builds
// with: those are the ones in the module cache, and lessons build offline.
// When server/go.mod moves, regenerate pgxlesson.go.mod/.sum with go mod tidy.
func TestPgxScaffoldMatchesServer(t *testing.T) {
	server, err := os.ReadFile("../../go.mod")
	if err != nil {
		t.Fatalf("read server go.mod: %v", err)
	}
	for _, mod := range []string{"github.com/jackc/pgx/v5", "golang.org/x/sync", "golang.org/x/text", "github.com/jackc/puddle/v2"} {
		re := regexp.MustCompile(regexp.QuoteMeta(mod) + ` (v\S+)`)
		want := re.FindStringSubmatch(string(server))
		got := re.FindStringSubmatch(pgxGoMod)
		if want == nil || got == nil || got[1] != want[1] {
			t.Errorf("%s: lesson module has %v, server has %v", mod, got, want)
		}
	}
}

func TestUsesPgx(t *testing.T) {
	tests := []struct {
		files map[string]string
		want  bool
	}{
		{map[string]string{"main.go": "import \"github.com/jackc/pgx/v5\""}, true},
		{map[string]string{"main.go": "import \"github.com/jackc/pgx/v5/pgxpool\""}, true},
		{map[string]string{"main.go": "import \"database/sql\""}, false},
		{map[string]string{"notes.md": "\"github.com/jackc/pgx/v5\""}, false},
	}
	for _, tc := range tests {
		if got := UsesPgx(tc.files); got != tc.want {
			t.Errorf("UsesPgx(%v) = %v, want %v", tc.files, got, tc.want)
		}
	}
}
