package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"
	"unicode/utf8"
)

// LEARN(inside.codebase.runner-sandbox): runProcess and cappedBuffer from internal/gorun/gorun.go.

type cappedBuffer struct {
	buf       bytes.Buffer
	cap       int
	truncated bool
}

func (c *cappedBuffer) Write(p []byte) (int, error) {
	if c.truncated {
		return len(p), nil // discard, but report success so the child never blocks
	}
	room := c.cap - c.buf.Len()
	if len(p) <= room {
		return c.buf.Write(p)
	}
	c.buf.Write(p[:max(room, 0)])
	c.buf.Truncate(c.buf.Len() - partialRuneLen(c.buf.Bytes()))
	c.buf.WriteString("\n[output truncated]")
	c.truncated = true
	return len(p), nil
}

// partialRuneLen counts the bytes at the end of b that start a character but don't finish it.
func partialRuneLen(b []byte) int {
	for i := len(b) - 1; i >= 0 && i >= len(b)-(utf8.UTFMax-1); i-- {
		if utf8.RuneStart(b[i]) {
			if utf8.FullRune(b[i:]) {
				return 0
			}
			return len(b) - i
		}
	}
	return 0
}

func main() {
	if len(os.Args) > 1 { // the "lesson program": an endless loop that prints
		for i := 1; ; i++ {
			fmt.Printf("satır %d\n", i)
		}
	}
	self, err := os.Executable()
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	cmd := exec.CommandContext(ctx, self, "flood")
	stdout := &cappedBuffer{cap: 40} // gorun: 64 KB per stream
	cmd.Stdout = stdout
	cmd.WaitDelay = 2 * time.Second // gorun's pipeGrace
	runErr := cmd.Run()

	exitCode, timedOut := 0, errors.Is(ctx.Err(), context.DeadlineExceeded)
	var exitErr *exec.ExitError
	switch {
	case timedOut:
		exitCode = -1
	case errors.As(runErr, &exitErr):
		exitCode = exitErr.ExitCode()
	}
	fmt.Printf("timedOut=%v exitCode=%d truncated=%v\n", timedOut, exitCode, stdout.truncated)
	fmt.Println(stdout.buf.String())
}
