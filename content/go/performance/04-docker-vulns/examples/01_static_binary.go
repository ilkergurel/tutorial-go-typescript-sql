package main

import (
	"debug/elf"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// A tiny HTTP service. It imports net, which links libc when cgo is on.
const app = `package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "ok")
	})
	log.Fatal(http.ListenAndServe(":8080", nil))
}
`

func main() {
	dir, err := os.MkdirTemp("", "image-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)
	write(filepath.Join(dir, "go.mod"), "module example.com/svc\n\ngo 1.27\n")
	write(filepath.Join(dir, "main.go"), app)

	// What the build stage of a multi-stage Dockerfile runs.
	bin := filepath.Join(dir, "svc")
	cmd := exec.Command("go", "build", "-trimpath", "-ldflags=-s", "-o", bin, ".")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOTOOLCHAIN=local", "GOPROXY=off",
		"GOOS=linux", "GOARCH=amd64", "CGO_ENABLED=0")
	if out, err := cmd.CombinedOutput(); err != nil {
		panic(fmt.Sprint(err, string(out)))
	}

	f, err := elf.Open(bin)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	interp := "none"
	for _, p := range f.Progs {
		if p.Type == elf.PT_INTERP { // the dynamic loader a dynamic binary needs
			interp = "present"
		}
	}
	libs, err := f.ImportedLibraries()
	if err != nil {
		panic(err)
	}
	fmt.Println("format:          ", f.Class, f.Machine)
	fmt.Println("dynamic loader:  ", interp)
	fmt.Println("shared libraries:", len(libs))
	fmt.Println("=> no loader, no libc: the file can run in FROM scratch or distroless/static")
}

func write(path, body string) {
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		panic(err)
	}
}
