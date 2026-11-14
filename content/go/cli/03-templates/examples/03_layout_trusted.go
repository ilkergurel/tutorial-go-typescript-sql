package main

import (
	"fmt"
	"html/template"
	"os"
)

// The layout declares blocks with defaults; pages override them.
const layout = `<title>{{block "title" .}}StackLab{{end}}</title>
<main>{{block "content" .}}nothing here{{end}}</main>
`

const profilePage = `{{define "title"}}Profile: {{.Name}}{{end}}
{{define "content"}}<p>{{.Bio}}</p> {{.Badge}}{{end}}`

type Profile struct {
	Name  string
	Bio   string        // typed by the user: escaped
	Badge template.HTML // built by our code: inserted verbatim
}

func main() {
	base := template.Must(template.New("layout").Parse(layout))
	// Clone before parsing the page, so other pages can reuse an untouched
	// base. Clone fails once a template has been executed.
	profile := template.Must(template.Must(base.Clone()).Parse(profilePage))

	p := Profile{
		Name:  "Ayşe",
		Bio:   "<b>Go</b> & SQL",
		Badge: template.HTML(`<span class="badge">admin</span>`),
	}
	for _, t := range []*template.Template{base, profile} {
		if err := t.Execute(os.Stdout, p); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}

	// Wrapping user input in template.HTML switches the protection off.
	p.Badge = template.HTML(`<img src=x onerror="steal()">`)
	if err := profile.ExecuteTemplate(os.Stdout, "content", p); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	fmt.Println()
}
