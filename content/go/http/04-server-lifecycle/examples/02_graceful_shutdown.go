package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

func main() {
	ln, err := net.Listen("tcp", "127.0.0.1:0") // port 0: the OS picks a free one
	if err != nil {
		panic(err)
	}
	base := "http://" + ln.Addr().String()

	// Shutdown waits for requests but never cancels them. Long-lived ones
	// (streams, long polls) need a signal: this context, canceled on Shutdown.
	serverCtx, stopStreams := context.WithCancel(context.Background())
	started, release := make(chan bool, 2), make(chan bool)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /report", func(w http.ResponseWriter, r *http.Request) {
		started <- true
		<-release // slow work in progress
		io.WriteString(w, "report done")
	})
	mux.HandleFunc("GET /events", func(w http.ResponseWriter, r *http.Request) {
		started <- true
		<-r.Context().Done() // a stream ends when the client or the server does
		io.WriteString(w, "stream closed: server shutting down")
	})
	srv := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		BaseContext:       func(net.Listener) context.Context { return serverCtx },
	}
	srv.RegisterOnShutdown(stopStreams)
	served := make(chan error, 1)
	go func() { served <- srv.Serve(ln) }()

	get := func(path string) <-chan string {
		out := make(chan string, 1)
		go func() {
			resp, err := http.Get(base + path)
			if err != nil {
				out <- path + ": no response"
				return
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			out <- fmt.Sprintf("%s: %d %s", path, resp.StatusCode, body)
		}()
		return out
	}
	report, events := get("/report"), get("/events")
	<-started
	<-started // both requests are in flight

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	shutdown := make(chan error, 1)
	go func() { shutdown <- srv.Shutdown(ctx) }()

	fmt.Println("Serve returned:", <-served)
	_, err = net.Dial("tcp", ln.Addr().String())
	fmt.Println("new connections refused:", err != nil)
	fmt.Println(<-events)
	close(release)
	fmt.Println(<-report)
	fmt.Println("Shutdown returned:", <-shutdown)
}
