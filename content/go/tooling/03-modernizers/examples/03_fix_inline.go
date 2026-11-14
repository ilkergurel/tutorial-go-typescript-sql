package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// A library renamed Format to Lira and kept the old name as a one-line
// wrapper marked //go:fix inline. The app still calls Format.
var files = map[string]string{
	"go.mod": "module example.com/shop\n\ngo 1.27\n",
	"money/money.go": `package money

import "fmt"

// Format formats an amount given in kuruş.
//
// Deprecated: use [Lira].
//
//go:fix inline
func Format(kurus int) string { return Lira(kurus) }

// Lira formats an amount given in kuruş as Turkish lira.
func Lira(kurus int) string { return fmt.Sprintf("%d,%02d TL", kurus/100, kurus%100) }
`,
	"main.go": `package main

import (
	"fmt"

	"example.com/shop/money"
)

func main() { fmt.Println(money.Format(1250)) }
`,
}

func main() {
	dir, err := os.MkdirTemp("", "shop-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)
	for name, body := range files {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			panic(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			panic(err)
		}
	}

	gocmd(dir, "fix", "-inline", "-diff", "./...") // only the inline fixer
	gocmd(dir, "fix", "-inline", "./...")
	gocmd(dir, "run", ".")
}

// gocmd runs the go command offline in dir. The diff's ---/+++ header lines
// hold absolute temp paths, so they are dropped.
func gocmd(dir string, args ...string) {
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOTOOLCHAIN=local", "GOPROXY=off")
	out, err := cmd.CombinedOutput()
	fmt.Printf("$ go %s\n", strings.Join(args, " "))
	for line := range strings.Lines(string(out)) {
		if !strings.HasPrefix(line, "--- ") && !strings.HasPrefix(line, "+++ ") {
			fmt.Print(line)
		}
	}
	if err != nil {
		fmt.Printf("(%v)\n", err)
	}
}
