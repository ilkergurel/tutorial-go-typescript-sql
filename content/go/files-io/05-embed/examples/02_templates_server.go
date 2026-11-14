package main

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing/fstest"
)

//go:embed templates/*.tmpl
var templateFiles embed.FS

//go:embed static
var staticFiles embed.FS

// newHandler takes an fs.FS, not an embed.FS: production passes the embedded
// files, tests a fstest.MapFS, development maybe os.DirFS("static").
func newHandler(assets fs.FS, pages *template.Template) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(assets)))
	mux.HandleFunc("GET /hello/{name}", func(w http.ResponseWriter, r *http.Request) {
		if err := pages.ExecuteTemplate(w, "welcome.tmpl", r.PathValue("name")); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	return mux
}

func get(h http.Handler, url string) {
	rec := httptest.NewRecorder() // no network: call the handler directly
	h.ServeHTTP(rec, httptest.NewRequest("GET", url, nil))
	fmt.Printf("GET %-22s %d %s\n", url, rec.Code, strings.TrimSpace(rec.Body.String()))
}

func main() {
	pages := template.Must(template.ParseFS(templateFiles, "templates/*.tmpl"))
	assets, err := fs.Sub(staticFiles, "static") // serve static/app.css as /app.css
	if err != nil {
		log.Fatal(err)
	}

	h := newHandler(assets, pages)
	get(h, "/hello/Ay%C5%9Fe")
	get(h, "/hello/%3Cscript%3E")
	get(h, "/static/app.css")
	get(h, "/static/_draft.html")

	test := newHandler(fstest.MapFS{"app.css": {Data: []byte("/* test */")}}, pages)
	get(test, "/static/app.css")
}
