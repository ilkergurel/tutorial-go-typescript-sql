package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"text/tabwriter"
)

type Line struct {
	Name  string
	Qty   int
	Kurus int64 // unit price
}

// RenderReceipt writes a plain-text receipt: output that is tedious to
// assert piece by piece and easy to review as a whole file.
func RenderReceipt(w io.Writer, lines []Line) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	var total int64
	for _, l := range lines {
		sum := int64(l.Qty) * l.Kurus
		total += sum
		fmt.Fprintf(tw, "%s\tx%d\t%7s TL\n", l.Name, l.Qty, lira(sum))
	}
	fmt.Fprintf(tw, "TOPLAM\t\t%7s TL\n", lira(total))
	return tw.Flush()
}

func lira(k int64) string { return fmt.Sprintf("%d,%02d", k/100, k%100) }

var update = flag.Bool("update", false, "rewrite golden files")

func TestReceipt(t *testing.T) {
	var got bytes.Buffer
	lines := []Line{{"Çay", 2, 1500}, {"Simit", 3, 1250}, {"Kaşar peyniri", 1, 18990}}
	if err := RenderReceipt(&got, lines); err != nil {
		t.Fatal(err)
	}

	golden := filepath.Join("testdata", t.Name()+".golden")
	if *update { // go test -run TestReceipt -update
		if err := os.WriteFile(golden, got.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got.Bytes(), want) {
		t.Fatalf("output differs from %s; if intended, run with -update and review the diff\ngot:\n%s\nwant:\n%s",
			filepath.ToSlash(golden), got.Bytes(), want)
	}
	t.Logf("matches %s (%d bytes)", filepath.ToSlash(golden), len(want))
}
