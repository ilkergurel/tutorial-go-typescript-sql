package main

import (
	"bytes"
	"debug/buildinfo"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const app = `package main

import "fmt"

var version = "dev" // overwritten at link time by -X main.version=...

func main() { fmt.Println("stacklab", version) }
`

func main() {
	os.MkdirAll("ship", 0o755) // the runner starts in a fresh directory
	write("ship/go.mod", "module example.com/stacklab\n\ngo 1.27\n")
	write("ship/main.go", app)

	ldflags := "-s -X main.version=v1.4.2"
	for _, target := range []string{"linux/amd64", "linux/arm64", "darwin/arm64", "windows/amd64"} {
		goos, goarch, _ := strings.Cut(target, "/")
		out := "dist/stacklab-" + goos + "-" + goarch
		gocmd([]string{"GOOS=" + goos, "GOARCH=" + goarch, "CGO_ENABLED=0"},
			"build", "-trimpath", "-ldflags="+ldflags, "-o", out, ".")

		bin, err := os.ReadFile("ship/" + out)
		if err != nil {
			panic(err)
		}
		info, err := buildinfo.Read(bytes.NewReader(bin)) // any target's format
		if err != nil {
			panic(err)
		}
		set := map[string]string{}
		for _, s := range info.Settings {
			set[s.Key] = s.Value
		}
		fmt.Printf("%-14s %-6s CGO_ENABLED=%s -trimpath=%s -ldflags recorded: %v\n", target,
			format(bin), set["CGO_ENABLED"], set["-trimpath"], set["-ldflags"] != "")
	}

	// Only a build for this machine can run here; -X works the same for go run.
	fmt.Print("go run here: ", gocmd(nil, "run", "-ldflags="+ldflags, "."))
}

// format names the executable format by the file's first bytes.
func format(bin []byte) string {
	for magic, name := range map[string]string{"\x7fELF": "ELF", "MZ": "PE", "\xcf\xfa\xed\xfe": "Mach-O"} {
		if bytes.HasPrefix(bin, []byte(magic)) {
			return name
		}
	}
	return "?"
}

func gocmd(env []string, args ...string) string {
	cmd := exec.Command("go", args...)
	cmd.Dir = "ship"
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		panic(fmt.Sprint(err, string(out)))
	}
	return string(out)
}

func write(path, body string) {
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		panic(err)
	}
}
