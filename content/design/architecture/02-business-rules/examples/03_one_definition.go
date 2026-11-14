// LEARN(design.architecture.business-rules): a rule written twice drifts; give it one home and ask that home.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"unicode"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// The rule: "usernames are unique, ignoring case". Four implementations of
// "ignoring case" (JavaScript's toLowerCase would turn İ into i + U+0307, a fifth).
func main() {
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close(ctx)

	fmt.Printf("%-8s %-8s %-8s %-8s %s\n", "input", "Go", "Go tr", "PG", `PG "tr-x-icu"`)
	for _, name := range []string{"IŞIL", "Isparta", "İsmail"} {
		var pg, pgTR string
		err := conn.QueryRow(ctx, `SELECT lower($1), lower($1 COLLATE "tr-x-icu")`, name).Scan(&pg, &pgTR)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%-8s %-8s %-8s %-8s %s\n", name, strings.ToLower(name),
			strings.ToLowerSpecial(unicode.TurkishCase, name), pg, pgTR)
	}

	// One home: a unique index on the Turkish-aware expression. The app asks it
	// by inserting, and maps the violation to a domain error.
	_, err = conn.Exec(ctx, `CREATE TABLE users (username text NOT NULL);
		CREATE UNIQUE INDEX users_username_ci ON users (lower(username COLLATE "tr-x-icu"))`)
	if err != nil {
		log.Fatal(err)
	}
	for _, name := range []string{"ışıl", "IŞIL", "Işıl", "işil"} {
		fmt.Printf("sign up %-5s -> %v\n", name, signUp(ctx, conn, name))
	}
}

var ErrUsernameTaken = errors.New("username taken")

func signUp(ctx context.Context, conn *pgx.Conn, name string) error {
	_, err := conn.Exec(ctx, "INSERT INTO users (username) VALUES ($1)", name)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.ConstraintName == "users_username_ci" {
		return ErrUsernameTaken
	}
	return err
}
