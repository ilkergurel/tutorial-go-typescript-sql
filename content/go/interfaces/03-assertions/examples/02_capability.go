package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// flusher names a capability only some writers have. It is declared here,
// at the consumer, and nobody has to implement it on purpose.
type flusher interface{ Flush() error }

func writeReport(w io.Writer, lines ...string) error {
	for _, l := range lines {
		if _, err := io.WriteString(w, l+"\n"); err != nil {
			return err
		}
	}
	// Upgrade: if this writer can also flush, do it. A writer without
	// Flush has nothing buffered to lose.
	if f, ok := w.(flusher); ok {
		fmt.Printf("(%T can flush)\n", w)
		return f.Flush()
	}
	fmt.Printf("(%T cannot flush)\n", w)
	return nil
}

func main() {
	var sb strings.Builder
	if err := writeReport(&sb, "ok: 3", "failed: 0"); err != nil {
		fmt.Println(err)
	}
	fmt.Print(sb.String())

	// *bufio.Writer keeps the bytes until Flush: without the upgrade above,
	// these lines would never reach stdout.
	bw := bufio.NewWriter(os.Stdout)
	if err := writeReport(bw, "ok: 5", "failed: 1"); err != nil {
		fmt.Println(err)
	}
}
