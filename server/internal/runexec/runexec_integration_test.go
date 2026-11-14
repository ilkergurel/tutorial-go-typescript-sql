//go:build integration

package runexec

import (
	"context"
	"os"
	"strings"
	"testing"

	"stacklab/server/internal/runapi"
	"stacklab/server/internal/sqlrun"
)

// A Go lesson that imports pgx builds offline, connects through DATABASE_URL
// into a schema of its own, commits, and leaves nothing behind.
func TestGoLessonWithPgxGetsItsOwnSchema(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	const program = `package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
)

func main() {
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		fmt.Println("connect:", err)
		os.Exit(1)
	}
	defer conn.Close(ctx)
	var schema string
	_ = conn.QueryRow(ctx, "SELECT current_schema()").Scan(&schema)
	_, err = conn.Exec(ctx, "CREATE TABLE t (n int); INSERT INTO t VALUES (1), (2)")
	var count int
	if err == nil {
		err = conn.QueryRow(ctx, "SELECT count(*) FROM t").Scan(&count)
	}
	fmt.Println(strings.HasPrefix(schema, "run_"), count, err)
	fmt.Println("schema:", schema)
}
`
	res, refusal, err := New(dsn, "").Run(context.Background(), runapi.Request{
		Runtime: runapi.RuntimeGo,
		Files:   map[string]string{"main.go": program},
	})
	if err != nil || refusal != nil {
		t.Fatalf("run: err=%v refusal=%+v", err, refusal)
	}
	if res.ExitCode != 0 || !strings.HasPrefix(res.Stdout, "true 2 <nil>\n") {
		t.Fatalf("unexpected result: stdout=%q stderr=%q exit=%d", res.Stdout, res.Stderr, res.ExitCode)
	}

	schema := strings.TrimSpace(strings.TrimPrefix(strings.SplitN(res.Stdout, "\n", 2)[1], "schema:"))
	s, err := sqlrun.Connect(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer s.Close(context.Background())
	check, err := sqlrun.Run(context.Background(), s, sqlrun.Request{
		Script: "SELECT count(*) FROM pg_namespace WHERE nspname = '" + schema + "'",
	})
	if err != nil || *check.Statements[0].Rows[0][0] != "0" {
		t.Fatalf("schema %s survived the run (err %v)", schema, err)
	}
}
