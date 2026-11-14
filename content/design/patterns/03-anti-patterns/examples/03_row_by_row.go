// LEARN(design.patterns.anti-patterns): row by row from Go fights SQL; a set-based statement doesn't.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
)

// statements is a pgx QueryTracer: it counts every Query, QueryRow and Exec.
type statements struct{ n int }

func (s *statements) TraceQueryStart(ctx context.Context, _ *pgx.Conn, _ pgx.TraceQueryStartData) context.Context {
	s.n++
	return ctx
}
func (s *statements) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}

// newPrice is the business rule: add 10 %, round to whole lira.
func newPrice(kurus int) int { return (kurus*110/100 + 50) / 100 * 100 }

func main() {
	ctx := context.Background()
	cfg, err := pgx.ParseConfig(os.Getenv("DATABASE_URL"))
	check(err)
	sent := &statements{}
	cfg.Tracer = sent
	conn, err := pgx.ConnectConfig(ctx, cfg)
	check(err)
	defer conn.Close(ctx)
	_, err = conn.Exec(ctx, `CREATE TABLE a (sku text PRIMARY KEY, kurus int NOT NULL);
		INSERT INTO a SELECT 'sku-' || i, 1999 * i FROM generate_series(1, 300) AS i;
		CREATE TABLE b AS TABLE a; CREATE TABLE c AS TABLE a`)
	check(err)

	// Java/ORM habit: load the rows, loop, save each one (and commit each one).
	sent.n = 0
	type price struct {
		SKU   string
		Kurus int
	}
	rows, _ := conn.Query(ctx, "SELECT sku, kurus FROM a")
	all, err := pgx.CollectRows(rows, pgx.RowToStructByPos[price])
	check(err)
	for _, p := range all {
		_, err := conn.Exec(ctx, "UPDATE a SET kurus = $1 WHERE sku = $2", newPrice(p.Kurus), p.SKU)
		check(err)
	}
	fmt.Println("row by row:     ", sent.n, "statements")

	// The rule stays in Go, but the write is one statement over two arrays.
	sent.n = 0
	skus, prices := make([]string, len(all)), make([]int, len(all))
	for i, p := range all {
		skus[i], prices[i] = p.SKU, newPrice(p.Kurus)
	}
	_, err = conn.Exec(ctx, `UPDATE b SET kurus = v.kurus
		FROM unnest($1::text[], $2::int[]) AS v(sku, kurus) WHERE b.sku = v.sku`, skus, prices)
	check(err)
	fmt.Println("arrays:         ", sent.n, "statement (the read is reused)")

	// The rule in SQL: say what must be true of every row, once.
	sent.n = 0
	_, err = conn.Exec(ctx, "UPDATE c SET kurus = (kurus * 110 / 100 + 50) / 100 * 100")
	check(err)
	fmt.Println("set-based:      ", sent.n, "statement")
	var diff int
	check(conn.QueryRow(ctx, `SELECT count(*) FROM a JOIN b USING (sku) JOIN c USING (sku)
		WHERE a.kurus <> b.kurus OR a.kurus <> c.kurus`).Scan(&diff))
	fmt.Println("rows that differ:", diff)
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
