package main

import (
	"context"
	"fmt"
	"slices"
	"sync"
)

// gen is the first stage. Each stage owns its output channel and closes it
// when it has nothing more to send.
func gen(ctx context.Context, nums ...int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for _, n := range nums {
			select {
			case out <- n:
			case <-ctx.Done(): // the consumer left: stop instead of blocking forever
				return
			}
		}
	}()
	return out
}

// square fans out to several workers reading one input channel, and fans
// their results back in to one output channel.
func square(ctx context.Context, in <-chan int, workers int) <-chan int {
	out := make(chan int)
	var wg sync.WaitGroup
	for range workers {
		wg.Go(func() {
			for n := range in {
				select {
				case out <- n * n:
				case <-ctx.Done():
					return
				}
			}
		})
	}
	// Close only after the last worker is done: closing earlier would make
	// a still-running worker panic on send.
	go func() { wg.Wait(); close(out) }()
	return out
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // on any early return, every stage unblocks and exits

	var got []int
	for v := range square(ctx, gen(ctx, 1, 2, 3, 4, 5, 6), 3) {
		got = append(got, v)
	}
	slices.Sort(got) // fan-out loses input order; sort for stable output
	fmt.Println(got)
}
