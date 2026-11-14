// LEARN(inside.codebase.contract): the error body is half the contract; a client that decodes only the success shape loses it.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
)

// runapi.Error and an abridged runapi.Result.
type Error struct {
	Status int    `json:"status"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
}
type Result struct {
	Stdout string `json:"stdout"`
}

// before decodes like runclient.Run did: always into Result, then the status.
func before(resp *http.Response) (*Result, *Error) {
	var res Result
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, &Error{Status: http.StatusBadGateway, Detail: err.Error()}
	}
	if resp.StatusCode != http.StatusOK {
		return nil, &Error{Status: resp.StatusCode, Title: "runner error"}
	}
	return &res, nil
}

// fixed picks the shape by status before decoding, as runclient.Run does now.
func fixed(resp *http.Response) (*Result, *Error) {
	if resp.StatusCode != http.StatusOK {
		e := &Error{Status: resp.StatusCode, Title: "runner error"}
		_ = json.NewDecoder(resp.Body).Decode(e) // a non-JSON body keeps status and title
		return nil, e
	}
	var res Result
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, &Error{Status: http.StatusBadGateway, Detail: err.Error()}
	}
	return &res, nil
}

func main() {
	// cmd/runner refusing a ts-test run in Docker mode.
	runner := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusNotImplemented)
		_ = json.NewEncoder(w).Encode(Error{501, "runner error", "ts-test lessons run only in local mode"})
	}))
	defer runner.Close()

	for _, c := range []struct {
		name   string
		decode func(*http.Response) (*Result, *Error)
	}{{"before", before}, {"fixed", fixed}} {
		resp, err := http.Post(runner.URL+"/run", "application/json", nil)
		if err != nil {
			log.Fatal(err)
		}
		_, e := c.decode(resp)
		resp.Body.Close()
		shown := e.Detail // ServerRunner shows body.detail, or else res.statusText
		if shown == "" {
			shown = http.StatusText(e.Status)
		}
		fmt.Printf("%s: detail %q -> the learner reads %q\n", c.name, e.Detail, shown)
	}
}
