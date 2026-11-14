package main

import (
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
)

// PanicError marks an error that started as a panic: a bug, not a failure.
type PanicError struct {
	Value any
	Stack []byte // the bug report: log it, never show it to users
}

func (e *PanicError) Error() string { return fmt.Sprintf("panic: %v", e.Value) }

// safely is the boundary of one goroutine. It has to run inside that
// goroutine: recover only sees panics of its own goroutine.
func safely(job func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = &PanicError{Value: r, Stack: debug.Stack()}
		}
	}()
	return job()
}

func main() {
	var cache map[string]int // a bug: never initialized
	jobs := []func() error{
		func() error { return nil },
		func() error { return errors.New("upstream returned 503") },
		func() error { cache["tr"]++; return nil },
	}

	errs := make([]error, len(jobs)) // one slot per job: no lock, stable order
	var wg sync.WaitGroup
	for i, job := range jobs {
		wg.Go(func() { errs[i] = safely(job) })
	}
	wg.Wait()

	for i, err := range errs {
		if pe, ok := errors.AsType[*PanicError](err); ok {
			fmt.Printf("job %d: bug: %v (stack captured: %t)\n", i, pe.Value, len(pe.Stack) > 0)
			continue
		}
		fmt.Printf("job %d: %v\n", i, err)
	}
	fmt.Println("process still running")
}
