package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// run is the whole program. It returns the exit code instead of calling
// os.Exit, so deferred cleanup still runs and tests can pass buffers.
func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: sum N...")
		return 2 // usage error: the code the flag package uses
	}
	total := 0
	for _, a := range args {
		n, err := strconv.Atoi(a)
		if err != nil {
			fmt.Fprintf(stderr, "sum: %v\n", err) // tool name, then the cause
			return 1
		}
		total += n
	}
	fmt.Fprintln(stdout, total) // only the result goes to stdout
	return 0
}

func main() {
	// A real tool's main is one line:
	//   os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
	// Here run gets buffers, so we can see where each byte went.
	for _, args := range [][]string{{"40", "2"}, {"40", "two"}, {}} {
		var out, errs strings.Builder
		code := run(args, &out, &errs)
		fmt.Printf("sum %q -> exit %d\n", args, code)
		fmt.Printf("  stdout: %q\n  stderr: %q\n", out.String(), errs.String())
	}

	fmt.Fprintln(os.Stderr, "diagnostics go to stderr: you see them, a pipe doesn't")
	os.Exit(run([]string{"1", "2", "3"}, os.Stdout, os.Stderr))
}
