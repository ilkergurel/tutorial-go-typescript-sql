package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"time"
)

func main() {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "backend: %s %s\n", r.Method, r.URL.Path)
		fmt.Fprintf(w, "  X-Forwarded-For: %q\n", r.Header.Get("X-Forwarded-For"))
		fmt.Fprintf(w, "  X-Gateway:       %q\n", r.Header.Get("X-Gateway"))
	}))
	target, err := url.Parse(backend.URL + "/api")
	if err != nil {
		log.Fatal(err)
	}

	proxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(target)                        // /orders/42 -> /api/orders/42
			pr.SetXForwarded()                       // the client's real address
			pr.Out.Header.Set("X-Gateway", "edge-1") // the backend may trust this
		},
		ModifyResponse: func(resp *http.Response) error {
			resp.Header.Set("Cache-Control", "no-store")
			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			http.Error(w, "upstream unavailable", http.StatusBadGateway) // log err, don't show it
		},
	}
	front := httptest.NewServer(proxy)
	defer front.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	call := func() {
		req, _ := http.NewRequest(http.MethodGet, front.URL+"/orders/42", nil)
		req.Header.Set("X-Forwarded-For", "203.0.113.7") // a client trying to look like someone else
		req.Header.Set("Connection", "X-Gateway")        // ...and to get our header stripped
		resp, err := client.Do(req)
		if err != nil {
			log.Fatal(err)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("%d Cache-Control=%q\n%s", resp.StatusCode, resp.Header.Get("Cache-Control"), body)
	}
	call()
	backend.Close() // the backend goes away
	call()
}
