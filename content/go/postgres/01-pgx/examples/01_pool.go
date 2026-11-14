package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx := context.Background()

	cfg, err := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	cfg.MaxConns = 2 // tiny on purpose, so the limit is easy to hit
	cfg.MaxConnLifetime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg) // returns before any connection exists
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil { // the first real connection: fail fast here
		log.Fatal(err)
	}

	// Two queries whose rows stay open: each keeps its connection checked out.
	rows1, _ := pool.Query(ctx, "SELECT pg_backend_pid()")
	rows2, _ := pool.Query(ctx, "SELECT pg_backend_pid()")
	pid := func(r pgx.Rows) (p int32) { r.Next(); r.Scan(&p); return p }
	fmt.Println("two server processes:", pid(rows1) != pid(rows2))
	fmt.Println("acquired:", pool.Stat().AcquiredConns(), "of", pool.Stat().MaxConns())

	short, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()
	var n int
	err = pool.QueryRow(short, "SELECT 1").Scan(&n)
	fmt.Println("third query:", err)

	rows1.Close() // hands its connection back to the pool
	err = pool.QueryRow(ctx, "SELECT 1").Scan(&n)
	fmt.Println("after rows1.Close():", n, err)
	rows2.Close()
}
