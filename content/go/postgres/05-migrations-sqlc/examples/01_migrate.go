package main

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"slices"
	"testing/fstest"

	"github.com/jackc/pgx/v5"
)

// migrate applies, in file-name order, every *.sql file in fsys that is not
// recorded yet; each file and its row in schema_migrations commit together.
func migrate(ctx context.Context, conn *pgx.Conn, fsys fs.FS) error {
	_, err := conn.Exec(ctx, "CREATE TABLE IF NOT EXISTS schema_migrations (name text PRIMARY KEY)")
	if err != nil {
		return err
	}
	names, _ := fs.Glob(fsys, "*.sql") // the only possible error is a bad pattern
	slices.Sort(names)                 // lexical order: 002 after 001, hence the zero padding
	for _, name := range names {
		body, err := fs.ReadFile(fsys, name)
		if err != nil {
			return err
		}
		err = pgx.BeginFunc(ctx, conn, func(tx pgx.Tx) error {
			tag, err := tx.Exec(ctx, "INSERT INTO schema_migrations VALUES ($1) ON CONFLICT DO NOTHING", name)
			if err != nil || tag.RowsAffected() == 0 {
				return err // nil when the file was applied before
			}
			fmt.Println("  applying", name)
			_, err = tx.Exec(ctx, string(body)) // no arguments: several statements allowed
			return err
		})
		if err != nil {
			return fmt.Errorf("migration %s: %w", name, err)
		}
	}
	return nil
}

func main() {
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close(ctx)

	// Stands in for //go:embed migrations/*.sql plus fs.Sub(embedded, "migrations").
	files := fstest.MapFS{"001_users.sql": {Data: []byte(`-- +goose Up
		CREATE TABLE users (id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY, email text NOT NULL);`)}}
	fmt.Println("deploy 1:", migrate(ctx, conn, files))
	// Between deploys the app stores data, which later migrations must keep.
	if _, err := conn.Exec(ctx, "INSERT INTO users (email) VALUES ('ayse@example.com')"); err != nil {
		log.Fatal(err)
	}

	files["002_users_name.sql"] = &fstest.MapFile{Data: []byte(`-- +goose Up
		ALTER TABLE users ADD COLUMN name text;
		UPDATE users SET name = split_part(email, '@', 1);
		ALTER TABLE users ALTER COLUMN name SET NOT NULL;`)}
	fmt.Println("deploy 2:", migrate(ctx, conn, files))
	fmt.Println("deploy 3:", migrate(ctx, conn, files)) // nothing new

	files["003_broken.sql"] = &fstest.MapFile{Data: []byte(`-- +goose Up
		ALTER TABLE users ADD COLUMN city text;
		ALTER TABLE users ADD COLUMN email text;`)}
	fmt.Println("deploy 4:", migrate(ctx, conn, files))

	var name string
	err = conn.QueryRow(ctx, "SELECT name FROM users").Scan(&name)
	fmt.Println("backfilled name:", name, err)
	_, err = conn.Exec(ctx, "SELECT city FROM users")
	fmt.Println("city after deploy 4:", err)
}
