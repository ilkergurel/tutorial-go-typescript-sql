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

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// show prints what arrives on conn until it has been quiet for 300 ms.
func show(ctx context.Context, conn *pgx.Conn, label string) {
	fmt.Println(label)
	for {
		wctx, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
		n, err := conn.WaitForNotification(wctx)
		cancel()
		if pgconn.Timeout(err) {
			return // quiet; the connection is still usable
		}
		check(err)
		fmt.Println("  " + n.Payload)
	}
}

func main() {
	ctx := context.Background()
	writer, err := pgx.Connect(ctx, os.Getenv("DATABASE_URL"))
	check(err)
	defer writer.Close(ctx)
	listener, err := pgx.Connect(ctx, os.Getenv("DATABASE_URL"))
	check(err)
	defer listener.Close(ctx)
	exec := func(conn *pgx.Conn, sql string) {
		_, err := conn.Exec(ctx, sql)
		check(err)
	}

	// Channels are database-wide: name this run's after its own schema.
	var channel string
	check(writer.QueryRow(ctx, "SELECT current_schema() || '_changes'").Scan(&channel))
	exec(writer, "CREATE EXTENSION tcn") // trusted: no superuser needed
	exec(writer, "CREATE TABLE test_runs (id int PRIMARY KEY, status text NOT NULL)")
	exec(writer, `CREATE TRIGGER test_runs_tcn AFTER INSERT OR UPDATE OR DELETE ON test_runs
		FOR EACH ROW EXECUTE FUNCTION triggered_change_notification('`+channel+`')`)
	exec(listener, "LISTEN "+pgx.Identifier{channel}.Sanitize())

	tx, err := writer.Begin(ctx)
	check(err)
	_, err = tx.Exec(ctx, "INSERT INTO test_runs VALUES (1, 'running'), (2, 'running')")
	check(err)
	show(ctx, listener, "INSERT, not yet committed:")
	check(tx.Commit(ctx))
	show(ctx, listener, "after COMMIT:")

	exec(writer, "UPDATE test_runs SET status = 'passed' WHERE id = 1")
	exec(writer, "DELETE FROM test_runs WHERE id = 2")
	show(ctx, listener, "UPDATE, then DELETE:")

	exec(writer, "BEGIN; UPDATE test_runs SET status = 'failed'; ROLLBACK")
	show(ctx, listener, "rolled-back UPDATE:")
}
