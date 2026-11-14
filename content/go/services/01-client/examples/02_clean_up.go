package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"time"
)

func main() {
	small, big := strings.Repeat("x", 2<<10), strings.Repeat("x", 1<<20)
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/big" {
			io.WriteString(w, big)
			return
		}
		io.WriteString(w, small)
	}))
	var opened atomic.Int32 // TCP connections the server accepted
	srv.Config.ConnState = func(_ net.Conn, s http.ConnState) {
		if s == http.StateNew {
			opened.Add(1)
		}
	}
	srv.Start()
	defer srv.Close()

	client := &http.Client{Timeout: 5 * time.Second} // one client for the whole program
	for _, mode := range []string{"leak", "close-big", "drain"} {
		before := opened.Load()
		for range 3 {
			path := "/small"
			if mode == "close-big" {
				path = "/big"
			}
			resp, err := client.Get(srv.URL + path)
			if err != nil {
				log.Fatal(err)
			}
			switch mode {
			case "leak": // forgot resp.Body.Close()
			case "close-big": // closed after reading nothing of 1 MiB
				resp.Body.Close()
			case "drain": // read to EOF, then closed
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}
		}
		fmt.Printf("%-9s 3 requests, connections opened: %d\n", mode, opened.Load()-before)
	}
}
