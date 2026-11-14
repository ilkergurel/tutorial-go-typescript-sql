package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// The app prints its own version: go build stamps it, no -ldflags needed.
const app = `package main

import (
	"fmt"
	"runtime/debug"
)

func main() {
	info, _ := debug.ReadBuildInfo()
	fmt.Println(info.Main.Version)
}
`

func main() {
	os.MkdirAll("repo", 0o755) // the runner starts in a fresh directory
	write("repo/go.mod", "module example.com/stacklab\n\ngo 1.27\n")
	write("repo/main.go", app)

	run("git", "init", "-q")
	run("git", "add", ".")
	run("git", "commit", "-q", "-m", "first release")
	run("git", "tag", "v1.4.2")
	fmt.Println("tagged commit:       ", buildAndRun())

	write("repo/main.go", app+"\n// work in progress\n")
	fmt.Println("uncommitted change:  ", buildAndRun())

	run("git", "commit", "-q", "-am", "after the release")
	fmt.Println("commit after the tag:", buildAndRun())
}

var rev = regexp.MustCompile(`-[0-9a-f]{12}$`) // the commit hash part

// buildAndRun writes the binary outside the repository: inside, it would be
// an untracked file, and that counts as a modification.
func buildAndRun() string {
	run("go", "build", "-o", "../bin/app.exe", ".")
	out, err := exec.Command("bin/app.exe").Output()
	if err != nil {
		panic(err)
	}
	return rev.ReplaceAllString(strings.TrimSpace(string(out)), "-<rev12>")
}

// run executes a command in repo with a fixed git identity and clock and
// without the user's git configuration, so every run makes the same history.
func run(name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Dir = "repo"
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=../no-gitconfig", "GIT_CONFIG_NOSYSTEM=1",
		"GIT_AUTHOR_NAME=ayse", "GIT_AUTHOR_EMAIL=ayse@example.com",
		"GIT_COMMITTER_NAME=ayse", "GIT_COMMITTER_EMAIL=ayse@example.com",
		"GIT_AUTHOR_DATE=2026-11-02T09:00:00Z", "GIT_COMMITTER_DATE=2026-11-02T09:00:00Z")
	if out, err := cmd.CombinedOutput(); err != nil {
		panic(fmt.Sprint(name, args, err, string(out)))
	}
}

func write(path, body string) {
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		panic(err)
	}
}
