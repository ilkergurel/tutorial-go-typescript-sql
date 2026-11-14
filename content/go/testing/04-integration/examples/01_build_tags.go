package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// A throwaway module with a unit test, a slow test and an integration test.
var files = map[string]string{
	"go.mod": "module example.com/store\n\ngo 1.27\n",
	"store_test.go": `package store

import "testing"

func TestParse(t *testing.T) {}

func TestReport(t *testing.T) {
	if testing.Short() {
		t.Skip("slow: skipped with -short")
	}
}
`,
	"store_integration_test.go": `//go:build integration

package store

import (
	"os"
	"testing"
)

func TestUsersOnPostgres(t *testing.T) {
	if os.Getenv("TEST_DATABASE_URL") == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
}
`,
}

func main() {
	dir, err := os.MkdirTemp("", "tags-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			panic(err)
		}
	}
	gotest(dir, "-short")
	gotest(dir, "-tags=integration")
}

var timing = regexp.MustCompile(` \([0-9.]+s\)$`)

// gotest runs go test -v with extra flags and prints the per-test lines.
func gotest(dir string, flags ...string) {
	args := append([]string{"test", "-v", "-count=1"}, flags...)
	cmd := exec.Command("go", append(args, "./...")...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOTOOLCHAIN=local", "GOPROXY=off", "GOFLAGS=", "TEST_DATABASE_URL=")
	out, err := cmd.CombinedOutput()
	fmt.Printf("$ go %s ./...\n", strings.Join(args, " "))
	for _, line := range strings.Split(strings.ReplaceAll(string(out), "\r\n", "\n"), "\n") {
		if strings.HasPrefix(line, "--- ") || strings.Contains(line, "_test.go:") {
			fmt.Println(timing.ReplaceAllString(line, ""))
		}
	}
	if err != nil {
		fmt.Printf("(%v)\n", err)
	}
}
