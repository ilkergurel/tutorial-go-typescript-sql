package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgconn/ctxwatch"
)

type setup struct {
	name      string
	configure func(*pgx.ConnConfig)
}

func main() {
	ctx := context.Background()
	setups := []setup{
		{"default", func(*pgx.ConnConfig) {}},
		{"cancel request", func(c *pgx.ConnConfig) {
			c.BuildContextWatcherHandler = func(pc *pgconn.PgConn) ctxwatch.Handler {
				return &pgconn.CancelRequestContextWatcherHandler{Conn: pc, DeadlineDelay: 2 * time.Second}
			}
		}},
		{"statement_timeout", func(c *pgx.ConnConfig) {
			c.RuntimeParams["statement_timeout"] = "100ms" // session default, set at connect
		}},
	}

	for _, s := range setups {
		cfg, err := pgx.ParseConfig(os.Getenv("DATABASE_URL"))
		if err != nil {
			log.Fatal(err)
		}
		s.configure(cfg)
		conn, err := pgx.ConnectConfig(ctx, cfg)
		if err != nil {
			log.Fatal(err)
		}

		short, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
		_, err = conn.Exec(short, "SELECT pg_sleep(3)")
		cancel()
		fmt.Printf("%s: %v\n", s.name, err)
		fmt.Printf("  errors.Is(err, context.DeadlineExceeded)=%t, connection closed=%t\n",
			errors.Is(err, context.DeadlineExceeded), conn.IsClosed())
		conn.Close(ctx)
	}
}
