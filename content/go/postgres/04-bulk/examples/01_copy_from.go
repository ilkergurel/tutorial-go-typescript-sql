package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
)

func main() {
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close(ctx)
	_, err = conn.Exec(ctx, `CREATE TABLE readings (
		id bigint PRIMARY KEY, city text NOT NULL, celsius numeric(4,1) NOT NULL)`)
	if err != nil {
		log.Fatal(err)
	}

	cities := []string{"İzmir", "Ankara", "Muğla", "İstanbul", "Diyarbakır"}
	const n = 100_000
	copied, err := conn.CopyFrom(ctx,
		pgx.Identifier{"readings"},
		[]string{"id", "city", "celsius"},
		pgx.CopyFromSlice(n, func(i int) ([]any, error) { // one row per call: no [][]any of 100k rows
			return []any{int64(i + 1), cities[i%len(cities)], float64(i%400) / 10}, nil
		}),
	)
	fmt.Println("copied:", copied, err)

	// One duplicate key in the next load: COPY is a single statement, all or nothing.
	more := [][]any{{int64(100_001), "Ankara", 12.5}, {int64(1), "İzmir", 30.0}}
	copied, err = conn.CopyFrom(ctx, pgx.Identifier{"readings"}, []string{"id", "city", "celsius"},
		pgx.CopyFromRows(more))
	fmt.Println("copied:", copied, err)

	rows, _ := conn.Query(ctx, `SELECT city || ' ' || count(*) || ' ' || round(avg(celsius), 2)
		FROM readings GROUP BY city ORDER BY city COLLATE "tr-x-icu"`)
	summary, err := pgx.CollectRows(rows, pgx.RowTo[string])
	for _, line := range summary {
		fmt.Println(" ", line)
	}
	fmt.Println(err)
}
