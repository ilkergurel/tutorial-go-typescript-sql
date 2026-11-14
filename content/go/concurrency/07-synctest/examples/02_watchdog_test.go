package main

import (
	"context"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

// Watchdog declares a peer dead after timeout without a heartbeat.
type Watchdog struct {
	beats chan struct{}
	dead  atomic.Bool
}

func NewWatchdog(ctx context.Context, timeout time.Duration) *Watchdog {
	w := &Watchdog{beats: make(chan struct{})}
	go func() {
		timer := time.NewTimer(timeout)
		for {
			select {
			case <-w.beats:
				timer.Reset(timeout)
			case <-timer.C:
				w.dead.Store(true)
				return
			case <-ctx.Done():
				return
			}
		}
	}()
	return w
}

func (w *Watchdog) Beat()      { w.beats <- struct{}{} }
func (w *Watchdog) Dead() bool { return w.dead.Load() }

func TestWatchdog(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		start := time.Now()
		w := NewWatchdog(t.Context(), 30*time.Second)

		// synctest.Sleep (Go 1.27) = time.Sleep, then synctest.Wait: the
		// clock moves, then every other goroutine in the bubble settles.
		synctest.Sleep(20 * time.Second)
		w.Beat() // at 20s: the deadline moves to 50s
		synctest.Sleep(29 * time.Second)
		if w.Dead() {
			t.Fatal("dead at 49s, only 29s after a heartbeat")
		}
		synctest.Sleep(1 * time.Second)
		if !w.Dead() {
			t.Fatal("alive at 50s, after 30s of silence")
		}
		t.Logf("declared dead at %v of fake time", time.Since(start))
	})
}
