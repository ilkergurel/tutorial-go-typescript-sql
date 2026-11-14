package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"
)

// payments charges once per Idempotency-Key and replays the stored result
// for every repeat. The first response it sends is lost on the way back.
type payments struct {
	mu      sync.Mutex
	byKey   map[string]string // key -> payment id
	charges int
	calls   int
}

func (p *payments) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.calls++
	key := r.Header.Get("Idempotency-Key")
	id, seen := p.byKey[key]
	if key == "" || !seen {
		p.charges++ // in real life: one database transaction
		id = fmt.Sprintf("pay_%d", p.charges)
		p.byKey[key] = id
	}
	if p.calls == 1 {
		panic(http.ErrAbortHandler) // committed, but the connection dies before the answer
	}
	w.WriteHeader(http.StatusCreated)
	io.WriteString(w, id)
}

// post sends one attempt of the payment.
func post(ctx context.Context, c *http.Client, url, key string) (string, error) {
	body := bytes.NewReader([]byte(`{"amount_kurus":125000,"currency":"TRY"}`))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return "", err
	}
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	resp, err := c.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	id, err := io.ReadAll(resp.Body)
	return string(id), err
}

func main() {
	ctx, client := context.Background(), &http.Client{Timeout: 5 * time.Second}
	for _, key := range []string{"", rand.Text()} {
		fmt.Printf("with Idempotency-Key: %t\n", key != "")
		p := &payments{byKey: map[string]string{}}
		srv := httptest.NewServer(p)
		id, err := post(ctx, client, srv.URL, key)
		if err != nil {
			fmt.Println("  no response: charged or not? unknown, so retry")
			id, err = post(ctx, client, srv.URL, key) // the same key: same operation
		}
		srv.Close()
		fmt.Printf("  got %s (err=%v); customer charged %d time(s)\n", id, err, p.charges)
	}
}
