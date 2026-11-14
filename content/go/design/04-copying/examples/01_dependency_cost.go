package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

// A "utility" module with the one function we want, next to one we don't;
// an app that depends on it; and the same app with that function copied in.
var files = map[string]string{
	"strutil/go.mod": "module example.com/strutil\n\ngo 1.27\n",
	"strutil/strutil.go": `package strutil

import "net/http"

func Truncate(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n]) + "…"
	}
	return s
}

func Fetch(url string) (*http.Response, error) { return http.Get(url) }
`,
	"app-dep/go.mod": "module example.com/app\n\ngo 1.27\n\nrequire example.com/strutil v1.0.0\n\nreplace example.com/strutil => ../strutil\n",
	"app-dep/main.go": `package main

import (
	"fmt"

	"example.com/strutil"
)

func main() { fmt.Println(strutil.Truncate("Merhaba dünya, nasılsın?", 12)) }
`,
	"app-copy/go.mod": "module example.com/app\n\ngo 1.27\n",
	"app-copy/main.go": `package main

import "fmt"

// truncate is copied from example.com/strutil v1.0.0 (MIT), minus the dependency.
func truncate(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n]) + "…"
	}
	return s
}

func main() { fmt.Println(truncate("Merhaba dünya, nasılsın?", 12)) }
`,
}

func main() {
	for name, body := range files { // the runner starts in a fresh directory
		os.MkdirAll(filepath.Dir(name), 0o755) // if this fails, WriteFile fails too
		if err := os.WriteFile(name, []byte(body), 0o644); err != nil {
			panic(err)
		}
	}
	for _, app := range []string{"app-dep", "app-copy"} {
		// Everything go build would compile. The target is pinned, so the
		// count is the same on every machine.
		cmd := exec.Command("go", "list", "-deps", "./...")
		cmd.Dir = app
		cmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH=amd64", "CGO_ENABLED=0")
		out, err := cmd.CombinedOutput()
		if err != nil {
			panic(fmt.Sprintf("%v\n%s", err, out))
		}
		deps := strings.Fields(string(out))
		fmt.Printf("%-9s %3d packages, net/http %v, crypto/tls %v\n", app+":",
			len(deps), slices.Contains(deps, "net/http"), slices.Contains(deps, "crypto/tls"))
	}
}
