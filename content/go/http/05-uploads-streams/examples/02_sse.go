package main

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
)

// events streams Server-Sent Events. next paces the demo: the server only
// produces event n+1 after the client has received event n, which can only
// happen if every event is really flushed, not buffered.
func events(next <-chan bool, report chan<- string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		rc := http.NewResponseController(w)
		for id := 1; ; id++ {
			fmt.Fprintf(w, "id: %d\nevent: progress\ndata: {\"done\":%d}\n\n", id, id*25)
			if err := rc.Flush(); err != nil {
				report <- "flush failed: " + err.Error()
				return
			}
			select {
			case <-next:
			case <-r.Context().Done(): // the client went away
				report <- fmt.Sprintf("server: stopped after %d events: %v", id, r.Context().Err())
				return
			}
		}
	}
}

func main() {
	next, report := make(chan bool), make(chan string)
	srv := httptest.NewServer(events(next, report))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	req, _ := http.NewRequestWithContext(ctx, "GET", srv.URL, nil)
	req.Header.Set("Accept", "text/event-stream")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer resp.Body.Close()
	fmt.Printf("%s, length %d, transfer encoding %v\n",
		resp.Header.Get("Content-Type"), resp.ContentLength, resp.TransferEncoding)

	received := 0
	var event []string
	sc := bufio.NewScanner(resp.Body)
	for sc.Scan() {
		if line := sc.Text(); line != "" {
			event = append(event, line)
			continue
		}
		received++ // a blank line ends one event
		fmt.Println("client:", strings.Join(event, " | "))
		event = nil
		if received == 3 {
			cancel() // close the stream, as a browser tab does when it closes
			break
		}
		next <- true
	}
	fmt.Println(<-report)
}
