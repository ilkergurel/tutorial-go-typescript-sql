package main

import "fmt"

// tryPublish never blocks: when the buffer is full the event is dropped
// and the caller learns about it, instead of stalling the request path.
func tryPublish(events chan<- string, e string) bool {
	select {
	case events <- e:
		return true
	default:
		return false
	}
}

func main() {
	events := make(chan string, 2) // room for two events, no consumer yet
	for _, e := range []string{"login", "view", "click"} {
		fmt.Println(e, "queued:", tryPublish(events, e))
	}

	// Non-blocking receive: drain what is buffered, stop when it is empty.
	for drained := false; !drained; {
		select {
		case e := <-events:
			fmt.Println("handled", e)
		default:
			drained = true
		}
	}

	// A closed channel is always ready to receive. With two ready cases,
	// select picks one at random: the first case gets no priority.
	a, b := make(chan struct{}), make(chan struct{})
	close(a)
	close(b)
	var fromA, fromB int
	for range 1000 {
		select {
		case <-a:
			fromA++
		case <-b:
			fromB++
		}
	}
	fmt.Println("both cases chosen:", fromA > 0 && fromB > 0)
}
