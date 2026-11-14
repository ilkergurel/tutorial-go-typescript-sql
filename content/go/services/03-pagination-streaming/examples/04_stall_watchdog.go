package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"
)

const idle = time.Second

var errStalled = errors.New("stalled: no data for 1s")

// progress pushes the stall deadline back whenever bytes arrive.
type progress struct {
	r     io.Reader
	timer *time.Timer
}

func (p progress) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	if n > 0 {
		p.timer.Reset(idle)
	}
	return n, err
}

func main() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for range 5 { // slow but healthy: 2 s in total, never 1 s without data
			io.WriteString(w, strings.Repeat("x", 1000))
			http.NewResponseController(w).Flush()
			time.Sleep(400 * time.Millisecond)
		}
		<-r.Context().Done() // then the upstream hangs mid-body
	}))
	defer srv.Close()

	t := http.DefaultTransport.(*http.Transport).Clone()
	t.ResponseHeaderTimeout = 5 * time.Second
	client := &http.Client{Transport: t} // no Client.Timeout: it would cap the body too

	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
	if err != nil {
		panic(err)
	}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	stall := time.AfterFunc(idle, func() { cancel(errStalled) })
	defer stall.Stop()

	n, err := io.Copy(io.Discard, progress{resp.Body, stall})
	fmt.Printf("read %d bytes, err=%v\n", n, err)
	fmt.Println("errors.Is(err, errStalled):", errors.Is(err, errStalled))
}
