package main

import (
	"fmt"
	htmltemplate "html/template"
	"io"
	"os"
	texttemplate "text/template"
)

// One template text, five contexts: element body, attribute, URL, URL query
// and JavaScript. Each needs a different escaping rule.
const page = `<p>Hi {{.Name}}</p>
<a title="{{.Name}}" href="{{.Site}}">site</a>
<a href="/search?q={{.Name}}">search</a>
<script>const user = {{.Name}};</script>
`

type User struct{ Name, Site string }

// Both template types have this method, so one loop runs both.
type executor interface {
	Execute(w io.Writer, data any) error
}

func main() {
	u := User{
		Name: `Ayşe"><script>steal()</script>`,
		Site: `javascript:steal()`,
	}
	for _, t := range []struct {
		label string
		tmpl  executor
	}{
		{"text/template: bytes in, bytes out", texttemplate.Must(texttemplate.New("p").Parse(page))},
		{"html/template: escaped per context", htmltemplate.Must(htmltemplate.New("p").Parse(page))},
	} {
		fmt.Println("==", t.label)
		if err := t.tmpl.Execute(os.Stdout, u); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
}
