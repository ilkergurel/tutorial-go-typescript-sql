package main

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
)

type config struct{ dsn string }

func main() {
	var loads atomic.Int32 // how often the expensive work really ran

	// Usually a package-level var or a struct field. The function runs on
	// first use, once, however many goroutines call it at the same moment.
	loadConfig := sync.OnceValues(func() (config, error) {
		loads.Add(1)
		return config{dsn: "postgres://127.0.0.1/stacklab"}, nil
	})

	dsns := make([]string, 8) // one slot per goroutine: no lock needed
	var wg sync.WaitGroup
	for i := range dsns {
		wg.Go(func() {
			cfg, _ := loadConfig()
			dsns[i] = cfg.dsn
		})
	}
	wg.Wait()
	fmt.Println("callers:", len(dsns), "loads:", loads.Load())
	fmt.Println("all saw the same value:", dsns[0] == dsns[7])

	// The result is cached, and so is an error: a failed first attempt is
	// never retried.
	var dials atomic.Int32
	connect := sync.OnceValues(func() (string, error) {
		dials.Add(1)
		return "", errors.New("connection refused")
	})
	for range 3 {
		_, err := connect()
		fmt.Println("connect:", err)
	}
	fmt.Println("dial attempts:", dials.Load())
}
