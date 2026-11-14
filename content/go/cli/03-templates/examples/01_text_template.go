package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"text/template"
	"unicode"
)

type Item struct {
	Name  string
	Qty   int
	Price float64
}

func (it Item) Total() float64 { return float64(it.Qty) * it.Price }

type Order struct {
	Customer, City, Note string
	Items                []Item
}

const report = `Order for {{.Customer}}, {{upper .City}}
{{range .Items -}}
- {{printf "%-6s" .Name}} x{{.Qty}} {{printf "%7.2f" .Total}} TL
{{else -}}
- (no items)
{{end -}}
{{with .Note}}Note: {{.}}
{{end -}}
`

func main() {
	funcs := template.FuncMap{
		"upper": func(s string) string { return strings.ToUpperSpecial(unicode.TurkishCase, s) },
	}
	t := template.Must(template.New("report").Funcs(funcs).Parse(report))

	orders := []Order{
		{Customer: "Ayşe", City: "izmir", Note: "ring twice",
			Items: []Item{{"çay", 2, 45.5}, {"simit", 3, 15}}},
		{Customer: "Mehmet", City: "Iğdır"},
	}
	for _, o := range orders {
		if err := t.Execute(os.Stdout, o); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}

	// A typo in a field name parses fine and fails during Execute, after the
	// text before it was written. Render into a buffer; copy it on success.
	bad := template.Must(template.New("bad").Parse("Hello {{.Custmer}}!\n"))
	var buf bytes.Buffer
	err := bad.Execute(&buf, orders[0])
	fmt.Printf("written before the error: %q\nerror: %v\n", buf.String(), err)
}
