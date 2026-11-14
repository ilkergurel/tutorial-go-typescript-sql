package main

import (
	"errors"
	"fmt"
	"time"
)

var errTimeout = errors.New("timed out")

// call starts a slow operation and returns at once. The result channel has
// room for one value, so the goroutine can always deliver and exit, even
// after the caller has stopped waiting.
func call(reply <-chan struct{}) <-chan string {
	result := make(chan string, 1)
	go func() {
		<-reply // stands in for a network round trip
		result <- "200 OK"
	}()
	return result
}

// await waits for whichever happens first: the result or the timeout.
func await(result <-chan string, limit time.Duration) (string, error) {
	select {
	case r := <-result:
		return r, nil
	case <-time.After(limit):
		return "", errTimeout
	}
}

func main() {
	fast := make(chan struct{})
	close(fast) // the server answers immediately
	fmt.Println(await(call(fast), time.Second))

	slow := make(chan struct{}) // the server has not answered yet
	r, err := await(call(slow), 50*time.Millisecond)
	fmt.Printf("%q %v\n", r, err)

	// The late answer still arrives. It lands in the buffer and the
	// goroutine exits instead of blocking forever on a send.
	close(slow)
}
