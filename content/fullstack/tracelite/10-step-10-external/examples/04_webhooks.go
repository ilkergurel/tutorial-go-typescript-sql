package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json/v2"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"
	"uuid"
)

// Event is the body of every delivery. Its id is the receivers' idempotency key.
type Event struct {
	ID        uuid.UUID `json:"id"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"createdAt"`
	Data      any       `json:"data"`
}

// mac is the scheme of go.services.webhooks-hmac: HMAC-SHA256 over "<unix time>.<body>".
func mac(secret string, unix int64, body []byte) []byte {
	m := hmac.New(sha256.New, []byte(secret))
	fmt.Fprintf(m, "%d.%s", unix, body)
	return m.Sum(nil)
}

// receiver is a subscriber's endpoint: the other half of the contract.
type receiver struct {
	secret string
	now    func() time.Time
	seen   sync.Map // event ids; in production, a table with the id as primary key
}

func (rc *receiver) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 64<<10))
	var unix int64
	var got []byte
	fmt.Sscanf(r.Header.Get("TraceLite-Signature"), "t=%d,v1=%x", &unix, &got) // a bad header leaves got empty
	age := rc.now().Sub(time.Unix(unix, 0)).Abs()
	if err != nil || !hmac.Equal(got, mac(rc.secret, unix, body)) || age > 5*time.Minute {
		http.Error(w, "bad or old signature", http.StatusUnauthorized)
		return
	}
	var e Event
	if err := json.Unmarshal(body, &e); err != nil {
		http.Error(w, "not an event", http.StatusBadRequest)
		return
	}
	if _, again := rc.seen.LoadOrStore(e.ID, true); again { // SQL: INSERT ... ON CONFLICT DO NOTHING
		fmt.Fprintln(w, "seen before, nothing done") // still 2xx, or TraceLite would keep retrying
		return
	}
	fmt.Fprintln(w, "processed", e.Type)
}

func main() {
	clock := time.Date(2026, 11, 13, 6, 2, 30, 0, time.UTC)
	ci := &receiver{secret: "ci-secret-26-characters-xx", now: func() time.Time { return clock }}
	body, _ := json.Marshal(Event{uuid.MustParse("01937b6e-0000-7000-8000-0000000000e1"), "test-run.failed", clock,
		map[string]any{"testCase": "TC-3", "requirements": []string{"AUTH-RESET"}}}) // once: every attempt sends these bytes
	deliver := func(label, secret string, signedAt time.Time) {
		req := httptest.NewRequest("POST", "/hooks/ci", bytes.NewReader(body))
		req.Header.Set("TraceLite-Signature", fmt.Sprintf("t=%d,v1=%x", signedAt.Unix(), mac(secret, signedAt.Unix(), body)))
		w := httptest.NewRecorder()
		ci.ServeHTTP(w, req)
		fmt.Printf("%-38s %d %s", label, w.Code, w.Body)
	}
	deliver("attempt 1 (its answer never arrives)", ci.secret, clock)
	clock = clock.Add(time.Minute)
	deliver("attempt 2: the same bytes, signed now", ci.secret, clock)
	deliver("signed with a guessed secret", "guessed", clock)
	deliver("a request signed 10 minutes ago", ci.secret, clock.Add(-10*time.Minute))
}
