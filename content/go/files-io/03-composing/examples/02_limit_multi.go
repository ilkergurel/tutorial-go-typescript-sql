package main

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

var errTooLarge = errors.New("input too large")

// readAtMost reads all of r but refuses more than max bytes. Asking LimitReader
// for max+1 is what tells "exactly max" apart from "more than max".
func readAtMost(r io.Reader, max int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, max+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > max {
		return nil, errTooLarge
	}
	return data, nil
}

func main() {
	for _, body := range []string{"Ankara", "Kastamonu", "Eskişehir"} {
		data, err := readAtMost(strings.NewReader(body), 9)
		fmt.Printf("%-10s %2d bytes -> %q %v\n", body, len(body), data, err)
	}

	// A byte limit knows nothing about characters: "Ş" and "ı" are 2 bytes each.
	preview, _ := io.ReadAll(io.LimitReader(strings.NewReader("Şırnak"), 3))
	fmt.Printf("first 3 bytes of Şırnak: %q, valid UTF-8: %t\n", preview, utf8.Valid(preview))

	// MultiReader concatenates: a header, then the parts, read as one stream.
	all := io.MultiReader(
		strings.NewReader("id;city\n"),
		strings.NewReader("1;Muğla\n"),
		strings.NewReader("2;Çorum\n"),
	)
	data, err := readAtMost(all, 1<<20)
	fmt.Printf("%q %v\n", data, err)
}
