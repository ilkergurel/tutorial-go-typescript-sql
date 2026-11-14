// LEARN(design.architecture.styles): one module boundary, as a function call and as a network call.
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"time"
)

// Invoicer is what the orders module needs from billing, declared by orders.
type Invoicer interface {
	Invoice(ctx context.Context, orderID int) error
}

// ledger is billing's data: which orders have an invoice.
type ledger struct {
	mu       sync.Mutex
	invoiced []int
}

func (l *ledger) Invoice(_ context.Context, orderID int) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.invoiced = append(l.invoiced, orderID)
	return nil
}

// remote is the same port over HTTP, as it would be after extracting billing.
type remote struct{ url string }

func (r remote) Invoice(ctx context.Context, orderID int) error {
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/invoices?order=%d", r.url, orderID), nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	return resp.Body.Close()
}

func placeOrders(label string, billing Invoicer) {
	for id := 1; id <= 3; id++ {
		err := billing.Invoice(context.Background(), id)
		switch {
		case err == nil:
			fmt.Printf("%s: order %d invoiced\n", label, id)
		case errors.Is(err, context.DeadlineExceeded):
			fmt.Printf("%s: order %d: timeout; invoiced or not? unknown\n", label, id)
		default:
			fmt.Printf("%s: order %d: %v\n", label, id, err)
		}
	}
}

func main() {
	local := &ledger{}
	placeOrders("in-process", local)

	svc := &ledger{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var id int
		fmt.Sscan(r.URL.Query().Get("order"), &id)
		if id == 2 {
			time.Sleep(300 * time.Millisecond) // a GC pause, a deploy, a slow disk
		}
		svc.Invoice(r.Context(), id)
	}))
	placeOrders("over HTTP", remote{url: srv.URL})
	srv.Close() // waits for the slow request to finish
	fmt.Println("billing service's ledger:", slices.Sorted(slices.Values(svc.invoiced)))
}
