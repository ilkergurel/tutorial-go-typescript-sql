package main

import (
	"context"
	"fmt"
	"time"
)

type userKey struct{}

func main() {
	// The request context: canceled when the client disconnects.
	request, cancelRequest := context.WithCancel(
		context.WithValue(context.Background(), userKey{}, "ayşe"))

	// Children derive from it. Canceling a child never touches its parent
	// or its siblings.
	dbCall, cancelDB := context.WithTimeout(request, time.Hour)
	defer cancelDB()
	cacheCall, cancelCache := context.WithCancel(request)
	cancelCache()
	fmt.Println("cache canceled; request:", request.Err(), "| db:", dbCall.Err())

	// Work that must outlive the request, like writing an audit record.
	// WithoutCancel (Go 1.21) keeps the values and drops the cancellation.
	audit := context.WithoutCancel(request)

	// AfterFunc (Go 1.21) runs a callback in its own goroutine on cancel.
	closed := make(chan struct{})
	context.AfterFunc(request, func() {
		fmt.Println("AfterFunc: closing the request's connection")
		close(closed)
	})

	cancelRequest() // cancels the whole subtree at once
	<-closed        // wait for the callback, so the output order is fixed

	fmt.Println("db:", dbCall.Err(), "| cache:", cacheCall.Err())
	fmt.Println("audit:", audit.Err(), "| user:", audit.Value(userKey{}))
}
