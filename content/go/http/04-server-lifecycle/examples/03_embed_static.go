package main

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
)

// The files are compiled into the binary: one file to deploy, no paths to
// get wrong at run time.
//
//go:embed static
var embedded embed.FS

func main() {
	// Paths inside embedded keep the directory: "static/app.css". fs.Sub
	// strips it, so URLs don't have to repeat it.
	assets, err := fs.Sub(embedded, "static")
	if err != nil {
		panic(err)
	}
	mux := http.NewServeMux()
	mux.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServerFS(assets)))
	srv := httptest.NewServer(mux)
	defer srv.Close()

	for _, path := range []string{
		"/assets/", "/assets/app.css", "/assets/index.html",
		"/assets/_draft.html", "/assets/img/",
	} {
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			fmt.Println("error:", err)
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		fmt.Printf("%-20s %d %-24s Last-Modified=%q\n", path, resp.StatusCode,
			resp.Header.Get("Content-Type"), resp.Header.Get("Last-Modified"))
		if final := resp.Request.URL.Path; final != path {
			fmt.Println("    redirected to", final)
		}
		if strings.Contains(string(body), "<pre>") {
			fmt.Printf("    a directory listing: %q\n", body)
		}
	}
}
