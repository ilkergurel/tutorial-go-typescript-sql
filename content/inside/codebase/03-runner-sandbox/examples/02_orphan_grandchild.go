package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func main() {
	self, _ := os.Executable()
	if len(os.Args) > 1 && os.Args[1] == "child" {
		gc := exec.Command(self, "grandchild")
		gc.Stdout = os.Stdout // the grandchild inherits the same output pipe
		_ = gc.Start()
		time.Sleep(time.Minute) // hangs until the timeout kills it
		return
	}
	if len(os.Args) > 1 { // grandchild: report its PID, then outlive the child
		fmt.Println(os.Getpid())
		time.Sleep(3 * time.Second)
		return
	}

	for _, grace := range []time.Duration{0, 300 * time.Millisecond} {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		cmd := exec.CommandContext(ctx, self, "child") // default Cancel kills the child only
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.WaitDelay = grace
		start := time.Now()
		_ = cmd.Run()
		elapsed := time.Since(start)
		cancel()

		waited := elapsed > 2500*time.Millisecond
		fmt.Printf("WaitDelay=%v: Run waited for the orphaned grandchild: %v\n", grace, waited)
		if pid, err := strconv.Atoi(strings.TrimSpace(out.String())); err == nil && !waited {
			p, _ := os.FindProcess(pid) // a manual tree kill, by PID
			fmt.Println("  grandchild still running, killed now:", p.Kill() == nil)
		}
	}
}
