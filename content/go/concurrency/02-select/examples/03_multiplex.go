package main

import (
	"fmt"
	"time"
)

// produce sends n events and closes its channel: the sender owns the close.
func produce(kind string, n int) <-chan string {
	ch := make(chan string)
	go func() {
		defer close(ch)
		for i := 1; i <= n; i++ {
			ch <- fmt.Sprintf("%s-%d", kind, i)
		}
	}()
	return ch
}

func main() {
	orders, refunds := produce("order", 3), produce("refund", 2)
	got := map[string][]string{}

	// One timer for the whole loop: an overall deadline, not per message.
	deadline := time.After(time.Second)

	// A nil channel is never ready, so setting a finished input to nil
	// switches its case off. The loop ends when both inputs are done.
	for orders != nil || refunds != nil {
		select {
		case o, ok := <-orders:
			if !ok {
				orders = nil
				continue
			}
			got["order"] = append(got["order"], o)
		case r, ok := <-refunds:
			if !ok {
				refunds = nil
				continue
			}
			got["refund"] = append(got["refund"], r)
		case <-deadline:
			fmt.Println("gave up waiting")
			return
		}
	}
	// How the two streams interleave varies per run; the order within
	// each channel never does.
	fmt.Println(got["order"])
	fmt.Println(got["refund"])
}
