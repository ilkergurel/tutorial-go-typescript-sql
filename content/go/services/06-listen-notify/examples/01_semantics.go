package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// received collects notifications until the channel has been quiet for 300 ms.
func received(ctx context.Context, conn *pgx.Conn) []string {
	var got []string
	for {
		wctx, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
		n, err := conn.WaitForNotification(wctx)
		cancel()
		if pgconn.Timeout(err) {
			return got // quiet; the connection is still usable
		}
		check(err)
		got = append(got, n.Payload)
	}
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	ctx := context.Background()
	listener, err := pgx.Connect(ctx, os.Getenv("DATABASE_URL"))
	check(err)
	defer listener.Close(ctx)
	sender, err := pgx.Connect(ctx, os.Getenv("DATABASE_URL"))
	check(err)
	defer sender.Close(ctx)

	// Channel names are database-wide: derive one unique to this run's schema.
	var channel string
	check(listener.QueryRow(ctx, "SELECT current_schema() || '_orders'").Scan(&channel))
	_, err = listener.Exec(ctx, "LISTEN "+pgx.Identifier{channel}.Sanitize())
	check(err)
	notify := func(tx pgx.Tx, payloads ...string) {
		for _, p := range payloads {
			_, err := tx.Exec(ctx, "SELECT pg_notify($1, $2)", channel, p)
			check(err)
		}
	}

	tx, err := sender.Begin(ctx)
	check(err)
	notify(tx, "order 1 paid")
	fmt.Println("before COMMIT:", received(ctx, listener))
	check(tx.Commit(ctx))
	fmt.Println("after COMMIT: ", received(ctx, listener))

	tx, err = sender.Begin(ctx)
	check(err)
	notify(tx, "order 2 paid")
	check(tx.Rollback(ctx))
	fmt.Println("rolled back:  ", received(ctx, listener))

	tx, err = sender.Begin(ctx)
	check(err)
	notify(tx, "order 3 paid", "order 3 paid", "order 4 paid", "order 3 shipped")
	check(tx.Commit(ctx))
	fmt.Println("four in a tx: ", received(ctx, listener))
}
