package main

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
)

// writeReport also returns Close's error: for a file you wrote, a failed
// Close can mean lost data. The named result lets the deferred call set it.
func writeReport(name string, rows int) (err error) {
	f, err := os.Create(name) // create or truncate
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, f.Close()) }()

	w := bufio.NewWriter(f) // 4096-byte buffer: one write call per 4 KB, not per row
	for i := range rows {
		fmt.Fprintf(w, "%d;ürün-%03d;%d,50\n", i, i, i*10)
	}
	if info, err := f.Stat(); err == nil {
		fmt.Printf("before Flush: %d bytes in the file, %d in the buffer\n", info.Size(), w.Buffered())
	}
	return w.Flush() // without it, the last partial buffer never reaches the file
}

func appendLine(name, line string) (err error) {
	f, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, f.Close()) }()
	_, err = f.WriteString(line + "\n")
	return err
}

func main() {
	if err := writeReport("report.csv", 300); err != nil {
		log.Fatal(err)
	}
	if info, err := os.Stat("report.csv"); err == nil {
		fmt.Println("after Flush and Close:", info.Size(), "bytes")
	}

	for _, line := range []string{"started", "stopped"} {
		if err := appendLine("app.log", line); err != nil {
			log.Fatal(err)
		}
	}
	data, _ := os.ReadFile("app.log")
	fmt.Printf("app.log: %q\n", data)

	_, err := os.OpenFile("app.log", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	fmt.Println("O_EXCL on an existing file is fs.ErrExist:", errors.Is(err, fs.ErrExist))
}
