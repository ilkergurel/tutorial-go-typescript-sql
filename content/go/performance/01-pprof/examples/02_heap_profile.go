package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
)

func init() { runtime.MemProfileRate = 1 } // record every allocation, not a sample

func parse(csv string) []int {
	var ids []int
	for _, field := range strings.Split(csv, ",") {
		id, _ := strconv.Atoi(field)
		ids = append(ids, id)
	}
	return ids
}

func render(ids []int) string {
	out := ""
	for _, id := range ids {
		out += "lesson " + strconv.Itoa(id) + "\n"
	}
	return out
}

var sink string

func main() {
	sink = render(parse(strings.Repeat("42,", 1999) + "42"))

	dir, err := os.MkdirTemp("", "heap-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "heap.out")
	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	runtime.GC() // the heap profile reports as of the last completed GC
	if err := pprof.WriteHeapProfile(f); err != nil {
		panic(err)
	}
	f.Close()

	// Which functions allocated the most bytes? Rows for main.* only.
	row := regexp.MustCompile(`^\s*(\S+)\s+\S+\s+\S+\s+(\S+)\s+\S+\s+(main\..*)$`)
	for _, line := range pprofLines("-top", "-nodefraction=0", "-sample_index=alloc_space", "-unit=MB", path) {
		if m := row.FindStringSubmatch(line); m != nil {
			fmt.Printf("flat %-8s cum %-8s %s\n", m[1], m[2], m[3])
		}
	}
	// And which line? -list annotates the source of matching functions.
	src := regexp.MustCompile(`^\s*(\S+)\s+\S+\s+(\d+):(.*)$`)
	for _, line := range pprofLines("-list=main.render", "-sample_index=alloc_space", "-unit=MB", path) {
		if m := src.FindStringSubmatch(line); m != nil && m[1] != "." {
			fmt.Printf("line %s: %-8s %s\n", m[2], m[1], strings.TrimSpace(m[3]))
		}
	}
}

func pprofLines(args ...string) []string {
	cmd := exec.Command("go", append([]string{"tool", "pprof"}, args...)...)
	cmd.Env = append(os.Environ(), "GOTOOLCHAIN=local", "GOPROXY=off")
	out, err := cmd.CombinedOutput()
	if err != nil {
		panic(fmt.Sprint(err, string(out)))
	}
	return strings.Split(string(out), "\n")
}
