package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync/atomic"
)

type middleware func(http.Handler) http.Handler

// chain wraps h so that the first middleware listed is the outermost: it
// sees the request first and the response last.
func chain(h http.Handler, mws ...middleware) http.Handler {
	for _, mw := range slices.Backward(mws) {
		h = mw(h)
	}
	return h
}

// trace prints when the request passes through, on the way in and out.
func trace(name string) middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Println("  ->", name)
			next.ServeHTTP(w, r)
			fmt.Println("  <-", name)
		})
	}
}

type ctxKey struct{} // unexported: no other package can collide with it

// requestID gives each request an ID from newID and passes it on, in the
// context for handlers and in the response for the client.
func requestID(newID func() string) middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := newID()
			w.Header().Set("X-Request-ID", id)
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, id)))
		})
	}
}

func requestIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(ctxKey{}).(string)
	return id
}

func main() {
	app := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("  handler sees", requestIDFrom(r.Context()))
	})
	// Predictable IDs for the output; real code passes crypto/rand.Text.
	// Atomic, because handlers run concurrently.
	var n atomic.Int64
	seq := func() string { return fmt.Sprintf("req-%d", n.Add(1)) }
	h := chain(app, requestID(seq), trace("outer"), trace("inner"))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	fmt.Println("response X-Request-ID:", rec.Header().Get("X-Request-ID"))
}
