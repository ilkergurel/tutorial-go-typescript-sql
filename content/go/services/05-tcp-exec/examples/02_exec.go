package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// child is what this same program does when started with a role argument.
func child(role string, args []string) {
	switch role {
	case "echo":
		for i, a := range args {
			fmt.Printf("  arg %d: %q\n", i+1, a)
		}
	case "fail":
		fmt.Fprintln(os.Stderr, "config.toml: no such file")
		os.Exit(3)
	case "hang":
		time.Sleep(time.Hour)
	}
}

func main() {
	if len(os.Args) > 1 {
		child(os.Args[1], os.Args[2:])
		return
	}
	ctx := context.Background()

	// A real tool, looked up on PATH, run directly: no shell in between.
	out, err := exec.CommandContext(ctx, "go", "list", "-f", "{{.Doc}}", "os/exec").Output()
	fmt.Printf("go list: %q err=%v\n", strings.TrimSpace(string(out)), err)

	self, err := os.Executable()
	if err != nil {
		panic(err)
	}
	fmt.Println("user input as arguments:")
	out, _ = exec.CommandContext(ctx, self, "echo", "Ayşe Yılmaz", "x; rm -rf ~", "$(whoami)", "*.go").Output()
	fmt.Print(string(out))

	_, err = exec.CommandContext(ctx, self, "fail").Output()
	if ee, ok := errors.AsType[*exec.ExitError](err); ok {
		fmt.Printf("failing child: exit code %d, stderr %q\n", ee.ExitCode(), ee.Stderr)
	}

	tctx, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
	defer cancel()
	cmd := exec.CommandContext(tctx, self, "hang")
	cmd.WaitDelay = time.Second // after the kill, don't wait forever for its pipes
	err = cmd.Run()
	fmt.Println("hanging child: killed at the deadline:", err != nil && errors.Is(tctx.Err(), context.DeadlineExceeded))
}
