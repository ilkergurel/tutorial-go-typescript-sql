package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
)

// greeter is a Handler: any type with a ServeHTTP method. Its fields are its
// dependencies, set once at startup and shared by every request.
type greeter struct{ greeting string }

func (g greeter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "%s, %s!\n", g.greeting, r.PathValue("name"))
}

// createUser is a plain function with the handler signature.
func createUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json") // headers first,
	w.WriteHeader(http.StatusCreated)                  // then the status,
	json.NewEncoder(w).Encode(map[string]int{"id": 7}) // then the body
}

func main() {
	mux := http.NewServeMux()
	mux.Handle("GET /hello/{name}", greeter{greeting: "Merhaba"})
	mux.HandleFunc("POST /users", createUser) // wrapped in http.HandlerFunc

	srv := httptest.NewServer(mux) // the mux is itself a Handler
	defer srv.Close()

	show := func(resp *http.Response, err error) {
		if err != nil {
			fmt.Println("error:", err)
			return
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("%d %-25s %s", resp.StatusCode, resp.Header.Get("Content-Type"), body)
	}
	show(http.Get(srv.URL + "/hello/Ayşe"))
	show(http.Get(srv.URL + "/hello/%C4%B0zmir"))
	show(http.Post(srv.URL+"/users", "application/json", nil))
	show(http.Get(srv.URL + "/users/7"))
}
