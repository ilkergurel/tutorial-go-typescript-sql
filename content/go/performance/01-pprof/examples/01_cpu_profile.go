package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/pprof"
	"slices"
	"strings"
)

// load fakes reading run records: lesson numbers with many repeats.
func load(n int) []int {
	ids := make([]int, n)
	for i := range ids {
		ids[i] = i * 7 % (n / 2)
	}
	return ids
}

// dedupe looks linear, but slices.Contains scans out on every call: O(n²).
func dedupe(ids []int) []int {
	var out []int
	for _, id := range ids {
		if !slices.Contains(out, id) {
			out = append(out, id)
		}
	}
	return out
}

func format(ids []int) string {
	return fmt.Sprint(len(ids), " unique lessons")
}

func main() {
	dir, err := os.MkdirTemp("", "cpu-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "cpu.out")
	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}

	if err := pprof.StartCPUProfile(f); err != nil { // samples the stacks ~100 times a second
		panic(err)
	}
	fmt.Println(format(dedupe(load(60000))))
	pprof.StopCPUProfile()
	f.Close()

	cmd := exec.Command("go", "tool", "pprof", "-top", "-cum", "-nodecount=6", path)
	cmd.Env = append(os.Environ(), "GOTOOLCHAIN=local", "GOPROXY=off")
	out, err := cmd.CombinedOutput()
	if err != nil {
		panic(fmt.Sprint(err, string(out)))
	}
	// Skip the header (binary path, date, duration): it differs on every run.
	_, table, _ := strings.Cut(string(out), "      flat  flat%")
	fmt.Print("      flat  flat%", table)
}
