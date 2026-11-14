package main

import (
	"fmt"
	"sync"
	"uuid"
)

// Hub (internal/events) hands each event, a topic to refetch, to its project's streams.
type Hub struct {
	mu   sync.Mutex
	subs map[chan string]uuid.UUID // every open stream, and its project
}

func (h *Hub) Subscribe(project uuid.UUID) (<-chan string, func()) {
	ch := make(chan string, 8)
	h.mu.Lock()
	defer h.mu.Unlock()
	h.subs[ch] = project
	return ch, func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		h.drop(ch)
	}
}

func (h *Hub) Publish(project uuid.UUID, topic string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch, p := range h.subs {
		if p == project {
			h.send(ch, topic)
		}
	}
}

func (h *Hub) Resync() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs {
		h.send(ch, "resync")
	}
}

// send and drop run with h.mu held.
func (h *Hub) send(ch chan string, topic string) {
	select {
	case ch <- topic:
	default: // eight unread events: end the stream; its browser reconnects and refetches
		h.drop(ch)
	}
}

func (h *Hub) drop(ch chan string) {
	if _, ok := h.subs[ch]; ok { // the hub and the stream's handler may both drop it
		delete(h.subs, ch)
		close(ch)
	}
}

func main() {
	odeme, raporlama := uuid.New(), uuid.New()
	hub := &Hub{subs: map[chan string]uuid.UUID{}}
	tab, closeTab := hub.Subscribe(odeme)
	stalled, _ := hub.Subscribe(odeme) // a tab that stopped reading
	other, _ := hub.Subscribe(raporlama)
	for range 9 {
		hub.Publish(odeme, "trace-links")
		<-tab // this tab keeps up
	}
	n := 0
	for range stalled { // ends because the hub closed the channel
		n++
	}
	fmt.Println("stalled tab:", n, "events, then closed; Raporlama's stream:", len(other), "events")
	hub.Resync()
	fmt.Println("after Resync:", <-tab, <-other)
	closeTab()
	fmt.Println("open streams:", len(hub.subs))
}
