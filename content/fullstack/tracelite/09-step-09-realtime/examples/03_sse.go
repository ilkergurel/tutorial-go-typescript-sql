package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"time"
	"uuid"
)

// GET /api/projects/{projectId}/events
func eventsHandler(exists func(uuid.UUID) bool, subscribe func(uuid.UUID) (<-chan string, func())) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		project, err := uuid.Parse(r.PathValue("projectId"))
		if err != nil || !exists(project) {
			http.Error(w, "no such project", http.StatusNotFound) // writeError in TraceLite
			return
		}
		events, unsubscribe := subscribe(project)
		defer unsubscribe()
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		rc := http.NewResponseController(w)
		rc.SetWriteDeadline(time.Time{}) // a stream must outlive any server WriteTimeout
		io.WriteString(w, "retry: 3000\n\n")
		ping := time.NewTicker(25 * time.Second)
		defer ping.Stop()
		for {
			if err := rc.Flush(); err != nil {
				return
			}
			select {
			case topic, ok := <-events:
				if !ok {
					return // dropped by the hub: EventSource reconnects and refetches
				}
				fmt.Fprintf(w, "event: %s\ndata: {\"topic\":%q}\n\n", topic, topic)
			case <-ping.C:
				io.WriteString(w, ": ping\n\n")
			case <-r.Context().Done():
				return // the tab closed
			}
		}
	}
}

func main() {
	odeme := uuid.MustParse("01937b6e-0000-7000-8000-00000000000a")
	events, unsubscribed := make(chan string), make(chan bool)
	subscribe := func(uuid.UUID) (<-chan string, func()) { return events, func() { close(unsubscribed) } }
	mux := http.NewServeMux()
	mux.Handle("GET /api/projects/{projectId}/events", eventsHandler(func(p uuid.UUID) bool { return p == odeme }, subscribe))
	srv := httptest.NewServer(mux)
	defer srv.Close()
	open := func(ctx context.Context, project uuid.UUID) *http.Response {
		req, _ := http.NewRequestWithContext(ctx, "GET", srv.URL+"/api/projects/"+project.String()+"/events", nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(resp.Status, resp.Header.Get("Content-Type"))
		return resp
	}
	open(context.Background(), uuid.New()) // an unknown project
	ctx, closeTab := context.WithCancel(context.Background())
	stream := bufio.NewReader(open(ctx, odeme).Body)
	events <- "trace-links"
	events <- "resync"
	for range 8 {
		line, _ := stream.ReadString('\n')
		fmt.Printf("%q\n", line)
	}
	closeTab()
	<-unsubscribed // the handler saw the request end and unsubscribed
	fmt.Println("tab closed, unsubscribed")
}
