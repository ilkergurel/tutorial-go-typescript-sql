// LEARN(fullstack.tracelite.step-09-realtime): a trigger NOTIFYs at commit, one Go connection LISTENs, a hub fans out per project, and SSE carries it to the browser.
package main

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"log"
	"os"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
)

// Listen keeps one dedicated connection LISTENing, for the life of the server.
func Listen(ctx context.Context, dsn string, publish func(uuid.UUID, string), resync func()) {
	for ctx.Err() == nil {
		err := listen(ctx, dsn, publish, resync)
		log.Printf("listener: %v; reconnecting in 1 s", err)
		time.Sleep(time.Second)
	}
}

func listen(ctx context.Context, dsn string, publish func(uuid.UUID, string), resync func()) error {
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return err
	}
	defer conn.Close(context.WithoutCancel(ctx))
	if _, err := conn.Exec(ctx, "LISTEN tracelite_changes"); err != nil {
		return err
	}
	resync() // only now: whatever changed before this point was not heard
	for {
		n, err := conn.WaitForNotification(ctx)
		if err != nil {
			return err
		}
		var change struct {
			ProjectID uuid.UUID `json:"projectId"`
			Topic     string    `json:"topic"`
		}
		if err := json.Unmarshal([]byte(n.Payload), &change); err != nil {
			return err // unknown change: reconnecting resyncs everyone
		}
		publish(change.ProjectID, change.Topic)
	}
}

func main() {
	ctx, dsn := context.Background(), os.Getenv("DATABASE_URL")
	db, err := pgx.Connect(ctx, dsn)
	if err != nil {
		log.Fatal(err)
	}
	exec := func(sql string) {
		if _, err := db.Exec(ctx, sql); err != nil {
			log.Fatal(err)
		}
	}
	migration, _ := os.ReadFile("migrations/002_change_events.sql") // this example's fixture
	exec(schema + string(migration))
	calls := make(chan string) // stands in for the hub of example 2
	publish := func(p uuid.UUID, topic string) { calls <- "Publish(" + p.String() + ", " + topic + ")" }
	go Listen(ctx, dsn, publish, func() { calls <- "Resync()" })
	fmt.Println("LISTEN is active:", <-calls)
	exec("INSERT INTO trace_links SELECT '01937b6e-0000-7000-8000-00000000000a' FROM generate_series(1, 3)")
	fmt.Println("three links, one statement:", <-calls)
	exec("INSERT INTO test_runs SELECT id FROM test_cases")
	fmt.Println("a test run, whose project is found through test_cases:", <-calls)
	exec("SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE query = 'LISTEN tracelite_changes'")
	fmt.Println("the listener's connection was killed:", <-calls)
}

// Only the columns migration 002 reads; one test case of project Ödeme.
const schema = `CREATE TABLE requirements (project_id uuid); CREATE TABLE trace_links (project_id uuid);
CREATE TABLE test_cases (id uuid, project_id uuid); CREATE TABLE test_runs (test_case_id uuid);
INSERT INTO test_cases VALUES (uuidv7(), '01937b6e-0000-7000-8000-00000000000a');
`
