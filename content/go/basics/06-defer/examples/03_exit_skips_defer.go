package main

import (
	"fmt"
	"os"
)

func main() {
	defer fmt.Println("cleanup: flush the buffer, delete the temp file")

	fmt.Println("work done, calling os.Exit(0)")
	// os.Exit ends the process on the spot: deferred calls never run.
	// log.Fatal and log.Fatalf call os.Exit(1), so they skip them too.
	os.Exit(0)
}
