package main

import (
	"fmt"
	"slices"
)

type message struct{ id, body string }

// queue is a toy at-least-once queue. A received message stays in flight
// until it is acked; when the visibility timeout expires, unacked messages
// are delivered again.
type queue struct{ ready, inFlight []message }

func (q *queue) receive() (message, bool) {
	if len(q.ready) == 0 {
		return message{}, false
	}
	m := q.ready[0]
	q.ready, q.inFlight = q.ready[1:], append(q.inFlight, m)
	return m, true
}

func (q *queue) ack(id string) {
	q.inFlight = slices.DeleteFunc(q.inFlight, func(m message) bool { return m.id == id })
}

func (q *queue) expire() { q.ready, q.inFlight = append(q.ready, q.inFlight...), nil }

// consume drains the queue. The consumer crashes once, after charging for
// m2 but before acking it.
func consume(dedupe bool) {
	q := &queue{ready: []message{{"m1", "order 1"}, {"m2", "order 2"}, {"m3", "order 3"}}}
	processed := map[string]bool{} // stands in for a table: it survives the crash
	charges := map[string]int{}
	crashed := false
	for {
		m, ok := q.receive()
		if !ok && len(q.inFlight) == 0 {
			break
		} else if !ok {
			fmt.Println("  visibility timeout: unacked messages are delivered again")
			q.expire()
			continue
		}
		if dedupe && processed[m.id] {
			fmt.Printf("  %s seen before: ack without charging\n", m.id)
			q.ack(m.id)
			continue
		}
		charges[m.body]++
		processed[m.id] = true // in production: same transaction as the charge
		fmt.Printf("  %s: charge %s\n", m.id, m.body)
		if m.id == "m2" && !crashed {
			crashed = true
			fmt.Println("  consumer crashes before acking m2")
			continue
		}
		q.ack(m.id)
	}
	fmt.Println("  charges:", charges)
}

func main() {
	fmt.Println("consumer without deduplication:")
	consume(false)
	fmt.Println("consumer that deduplicates by message ID:")
	consume(true)
}
