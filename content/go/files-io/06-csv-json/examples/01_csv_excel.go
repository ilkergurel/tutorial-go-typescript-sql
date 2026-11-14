package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"
)

const bom = "\xEF\xBB\xBF" // U+FEFF, the byte order mark, in UTF-8

// What Turkish Excel's "CSV UTF-8" export looks like: a BOM, ';' because ','
// is the decimal separator, CRLF line ends, quotes around a field with a ';'.
const export = bom + "Müşteri;Şehir;Tutar\r\n" +
	"Ayşe Yılmaz;İstanbul;1.250,50\r\n" +
	"\"Kaya; Ltd.\";Iğdır;99,90\r\n"

func main() {
	r := csv.NewReader(strings.NewReader(export))
	r.Comma = ';'
	records, err := r.ReadAll() // fine for small files; loop over r.Read for big ones
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("first header: %q\n", records[0][0])
	for _, rec := range records[1:] {
		fmt.Printf("%-11s | %-8s | %s\n", rec[0], rec[1], rec[2])
	}
	_, err = strconv.ParseFloat(records[1][2], 64)
	fmt.Println(err)

	// Plain "CSV (Comma delimited)" saves use the ANSI code page: windows-1254.
	legacy := []byte("\xddzmir;\xdeubat") // İzmir;Şubat in windows-1254
	asUTF8 := strings.ToValidUTF8(string(legacy), string(utf8.RuneError))
	fmt.Printf("valid UTF-8: %t, read as UTF-8: %q\n", utf8.Valid(legacy), asUTF8)

	// Writing for Excel: BOM first, then ';' and CRLF.
	var buf bytes.Buffer
	buf.WriteString(bom)
	w := csv.NewWriter(&buf)
	w.Comma, w.UseCRLF = ';', true
	err = w.WriteAll([][]string{{"Şehir", "Not"}, {"Muğla", `deniz; "tatil"`}})
	if err != nil { // WriteAll flushes and reports; after Write calls, Flush then check w.Error()
		log.Fatal(err)
	}
	if err := os.WriteFile("rapor.csv", buf.Bytes(), 0o644); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%q\n", buf.String())
}
