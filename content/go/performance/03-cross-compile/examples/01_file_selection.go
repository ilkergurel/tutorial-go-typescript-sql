package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// The shape of StackLab's gorun: shared code plus one file per OS family.
var files = map[string]string{
	"go.mod": "module example.com/runner\n\ngo 1.27\n",
	"main.go": `package main

import "fmt"

func main() { fmt.Println(killTree(42)) }
`,
	// The _windows suffix is a build constraint by itself.
	"proc_windows.go": `package main

import "fmt"

func killTree(pid int) string { return fmt.Sprintf("TerminateJobObject (job of %d)", pid) }
`,
	// No suffix for "unix", so the constraint is spelled out.
	"proc_unix.go": `//go:build unix

package main

import "fmt"

func killTree(pid int) string { return fmt.Sprintf("kill -KILL -%d (process group)", pid) }
`,
}

func main() {
	dir, err := os.MkdirTemp("", "runner-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			panic(err)
		}
	}

	for _, target := range []string{"linux/amd64", "darwin/arm64", "windows/amd64", "plan9/amd64"} {
		goos, goarch, _ := strings.Cut(target, "/")
		cmd := exec.Command("go", "list", "-f", "{{.GoFiles}} ignored: {{.IgnoredGoFiles}}", ".")
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GOTOOLCHAIN=local", "GOPROXY=off",
			"GOOS="+goos, "GOARCH="+goarch, "CGO_ENABLED=0")
		out, err := cmd.CombinedOutput()
		if err != nil {
			panic(fmt.Sprint(err, string(out)))
		}
		fmt.Printf("%-14s %s", target, out)
	}
}
