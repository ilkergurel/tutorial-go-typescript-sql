package main

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"time"
)

// halfRequest sends a request line and one header, never the blank line that
// ends the headers, then waits. One such client per connection is enough to
// tie up a server that has no timeouts (the slowloris attack).
func halfRequest(addr string) string {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return "dial failed"
	}
	defer conn.Close()
	io.WriteString(conn, "GET / HTTP/1.1\r\nHost: example\r\n")
	conn.SetReadDeadline(time.Now().Add(time.Second)) // the client's patience
	_, err = conn.Read(make([]byte, 1))
	if ne, ok := errors.AsType[net.Error](err); ok && ne.Timeout() {
		return "still open; a server goroutine is still waiting for headers"
	}
	return "closed by the server"
}

func main() {
	written := make(chan error, 1)
	slow := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond) // slower than the WriteTimeout below
		_, err := io.WriteString(w, "report")
		written <- err
	})

	naive := httptest.NewServer(slow) // the zero http.Server: no timeouts at all
	defer naive.Close()
	guarded := httptest.NewUnstartedServer(slow)
	guarded.Config.ReadHeaderTimeout = 100 * time.Millisecond
	guarded.Config.WriteTimeout = 200 * time.Millisecond
	guarded.Start()
	defer guarded.Close()

	fmt.Println("half a request, no timeouts:       ", halfRequest(naive.Listener.Addr().String()))
	fmt.Println("half a request, ReadHeaderTimeout: ", halfRequest(guarded.Listener.Addr().String()))

	resp, err := http.Get(guarded.URL)
	if err == nil {
		resp.Body.Close()
		fmt.Println("slow handler: got a response")
	} else {
		fmt.Println("slow handler, WriteTimeout: client got no response at all")
	}
	fmt.Println("the handler's own write returned:", <-written)
}
