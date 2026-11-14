package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"
)

type counts struct{ lines, words, chars, bytes int }

// count takes an io.Reader: os.Stdin in main, strings.NewReader in a test,
// an *os.File when the user names a file instead of piping one in.
func count(r io.Reader) (counts, error) {
	var c counts
	sc := bufio.NewScanner(r)
	for sc.Scan() { // strips "\n" and a preceding "\r"
		line := sc.Text()
		c.lines++
		c.words += len(strings.Fields(line))
		c.chars += utf8.RuneCountInString(line)
		c.bytes += len(line)
	}
	return c, sc.Err()
}

func main() {
	c, err := count(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "wc: reading stdin: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("newlines are not counted in chars or bytes")
	fmt.Printf("lines=%d words=%d chars=%d bytes=%d\n", c.lines, c.words, c.chars, c.bytes)
}
