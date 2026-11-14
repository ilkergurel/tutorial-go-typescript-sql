// LEARN(fullstack.tracelite.step-07-import): decode first, then split, then count characters: an Excel file from Turkish Windows read one validated record at a time.
package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
)

// check reads an upload record by record and returns every row problem. An
// error means the CSV itself is broken: nothing after it can be trusted.
func check(body io.Reader, charset string) ([]string, error) {
	if charset == "windows-1254" { // Excel's "CSV (Comma delimited)"
		body = charmap.Windows1254.NewDecoder().Reader(body)
	}
	br := bufio.NewReader(body)
	header, _ := br.ReadString('\n')
	header = strings.TrimPrefix(header, "\uFEFF") // "CSV UTF-8" starts with a BOM
	r := csv.NewReader(io.MultiReader(strings.NewReader(header), br))
	if strings.Count(header, ";") > strings.Count(header, ",") {
		r.Comma = ';'
	}
	r.FieldsPerRecord = -1 // a short row gets row problems, not a parse error
	names, err := r.Read()
	if err != nil {
		return nil, err
	}
	var problems []string
	for {
		rec, err := r.Read()
		if err == io.EOF {
			return problems, nil
		} else if err != nil {
			return problems, err
		}
		line, _ := r.FieldPos(0) // where the record starts; the header is line 1
		row := map[string]string{}
		for i := range min(len(names), len(rec)) {
			row[names[i]] = strings.TrimSpace(rec[i])
		}
		title := row["title"]
		if !utf8.ValidString(title) {
			problems = append(problems, fmt.Sprintf("line %d title: not UTF-8; is the file windows-1254?", line))
		} else if n := utf8.RuneCountInString(title); n < 1 || n > 200 {
			problems = append(problems, fmt.Sprintf("line %d title: must be 1-200 characters, got %d", line, n))
		}
	}
}

func main() {
	long := strings.Repeat("ş", 200) // 200 characters, 400 bytes of UTF-8
	sheet := "code;title;parent;status;description\r\n" +
		"AUTH;Kimlik doğrulama;;approved;\r\n" +
		"AUTH-RESET;;AUTH;;\r\n" +
		"LONG;" + long + ";;;\r\n" +
		"LONGER;" + long + "ş;;;\r\n"
	excelUTF8 := "\uFEFF" + sheet
	excelANSI, _ := charmap.Windows1254.NewEncoder().String(sheet)
	fmt.Printf("%d bytes as CSV UTF-8, %d as CSV (Comma delimited)\n", len(excelUTF8), len(excelANSI))
	for _, upload := range []struct{ name, body, charset string }{
		{"CSV UTF-8, labelled utf-8", excelUTF8, "utf-8"},
		{"CSV (Comma delimited), labelled windows-1254", excelANSI, "windows-1254"},
		{"CSV (Comma delimited), mislabelled utf-8", excelANSI, "utf-8"},
		{"a quote that never closes", "code;title\r\nAUTH;\"Kimlik\r\nEXPORT;Dışa\r\n", "utf-8"},
	} {
		fmt.Println(upload.name + ":")
		problems, err := check(strings.NewReader(upload.body), upload.charset)
		for _, problem := range problems {
			fmt.Println("  " + problem)
		}
		if err != nil {
			fmt.Println("  stopped:", err)
		}
	}
}
