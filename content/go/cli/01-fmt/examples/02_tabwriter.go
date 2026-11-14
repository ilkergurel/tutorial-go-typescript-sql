package main

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"unicode/utf8"
)

var cities = []string{"Istanbul", "İstanbul", "Iğdır", "Ankara"}

func main() {
	fmt.Println("naive: pad to 10 using len(s)")
	for _, c := range cities {
		fmt.Println(c + strings.Repeat(" ", 10-len(c)) + "|")
	}

	fmt.Println("fmt: a width of 10 pads by runes")
	for _, c := range cities {
		fmt.Printf("%-10s|\n", c)
	}

	fmt.Println("tabwriter: widths computed from the data")
	// minwidth 0, tabwidth 0, padding 2, pad with spaces, no flags.
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "CITY\tBYTES\tRUNES")
	for _, c := range cities {
		fmt.Fprintf(tw, "%s\t%d\t%d\n", c, len(c), utf8.RuneCountInString(c))
	}
	tw.Flush() // nothing reaches stdout before Flush
}
