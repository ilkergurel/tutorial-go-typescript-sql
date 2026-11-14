// LEARN(fullstack.tracelite.step-15-packaging): one binary carries the built app; the API, the files and the client-side routes share one origin.
package main

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/http/httptest"
	"path"
	"strings"
)

// In TraceLite this directive lives in web/web.go, next to dist, because a
// pattern can't climb out of its package's directory with "..".
//
//go:embed all:web/dist
var webFiles embed.FS

// spa serves the built app: real files as they are, client-side routes as
// index.html, and a missing file as a 404 rather than as the app.
func spa(dist fs.FS) http.Handler {
	files := http.FileServerFS(dist)
	index, err := fs.ReadFile(dist, "index.html")
	if err != nil {
		log.Fatal("web/dist has no index.html: run npm run build before go build")
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if info, err := fs.Stat(dist, name); err == nil && !info.IsDir() {
			if strings.HasPrefix(name, "assets/") { // Vite puts a content hash in these names
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}
			files.ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(name, "assets/") || path.Ext(name) != "" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(index)
	})
}

func main() {
	dist, _ := fs.Sub(webFiles, "web/dist") // embedded names keep their directory; a valid name can't fail
	mux := http.NewServeMux()
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) { // what no API route matched
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusNotFound)
		io.WriteString(w, `{"type": "about:blank", "title": "Not Found", "status": 404}`)
	})
	mux.Handle("/", spa(dist))
	srv := httptest.NewServer(mux)
	defer srv.Close()

	for _, p := range []string{"/", "/projects/01937b6e-0000-7000-8000-00000000000a",
		"/assets/index-B2xk91fa.js", "/assets/_commonjsHelpers-Dk2Y9xQb.js", "/assets/index-A1old000.js",
		"/index.html", "/api/no-such-endpoint"} {
		resp, err := http.Get(srv.URL + p)
		if err != nil {
			log.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		what := "file"
		if strings.HasPrefix(string(body), "<!doctype html>") {
			what = "the app"
		} else if resp.StatusCode == http.StatusNotFound {
			what = resp.Header.Get("Content-Type")
		}
		fmt.Println(strings.TrimSpace(fmt.Sprintf("%-47s %d  %-25s %s", p, resp.StatusCode, what, resp.Header.Get("Cache-Control"))))
	}
}
