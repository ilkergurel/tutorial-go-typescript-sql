package main

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type Pool struct{}

func NewPool(ctx context.Context) (*Pool, error) {
	fmt.Println("  pool: connected")
	return &Pool{}, nil
}

func (p *Pool) Close() { fmt.Println("  pool: closed") }

func Migrate(ctx context.Context, p *Pool, files []string) error {
	if len(files) == 0 {
		return errors.New("migrate: up: no migrations found")
	}
	fmt.Println("  migrations applied:", len(files))
	return nil
}

// run owns every resource. main only turns its error into an exit code.
func run(ctx context.Context, migrations []string) error {
	ctx, stop := context.WithCancel(ctx) // real: signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := NewPool(ctx)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := Migrate(ctx, pool, migrations); err != nil {
		return err
	}

	errCh := make(chan error, 1) // real: ListenAndServe runs in a goroutine and reports here
	fmt.Println("  serving")
	stop() // stands in for Ctrl+C
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		fmt.Println("  shutdown signal received, draining")
	}

	wrong, cancelWrong := context.WithTimeout(ctx, 10*time.Second)
	defer cancelWrong()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	fmt.Println("  drain budget derived from ctx:       ", wrong.Err())
	fmt.Println("  drain budget derived from Background:", shutdownCtx.Err())
	fmt.Println("  server drained") // real: srv.Shutdown(shutdownCtx)
	return nil
}

func main() {
	for _, migrations := range [][]string{nil, {"001_schema_info.sql", "002_runs.sql", "003_learning.sql"}} {
		fmt.Println("start:")
		if err := run(context.Background(), migrations); err != nil {
			fmt.Println("  startup failed:", err, "(real main: os.Exit(1))")
			continue
		}
		fmt.Println("  stopped")
	}
}
