package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// beginner is all WithTx needs: *pgx.Conn, *pgxpool.Pool and pgx.Tx have it.
type beginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// WithTx commits when fn returns nil and rolls back on an error or a panic.
func WithTx(ctx context.Context, db beginner, fn func(pgx.Tx) error) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback(ctx) // after a successful Commit this is a no-op
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

var ErrInsufficientFunds = errors.New("insufficient funds")

func transfer(ctx context.Context, db beginner, from, to string, amount int) error {
	return WithTx(ctx, db, func(tx pgx.Tx) error {
		const q = "UPDATE accounts SET balance = balance + $2 WHERE owner = $1"
		if _, err := tx.Exec(ctx, q, to, amount); err != nil {
			return fmt.Errorf("credit %s: %w", to, err)
		}
		_, err := tx.Exec(ctx, q, from, -amount)
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23514" {
			return fmt.Errorf("debit %s: %w", from, ErrInsufficientFunds) // check_violation
		}
		return err
	})
}

func main() {
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close(ctx)
	_, err = conn.Exec(ctx, `
		CREATE TABLE accounts (owner text PRIMARY KEY, balance int NOT NULL CHECK (balance >= 0));
		INSERT INTO accounts VALUES ('Ayşe', 100), ('Çağrı', 20)`)
	if err != nil {
		log.Fatal(err)
	}

	for _, amount := range []int{30, 500} {
		err := transfer(ctx, conn, "Ayşe", "Çağrı", amount)
		fmt.Printf("transfer %d: err=%v insufficient=%t\n", amount, err, errors.Is(err, ErrInsufficientFunds))
		rows, _ := conn.Query(ctx, "SELECT owner || '=' || balance FROM accounts ORDER BY owner")
		balances, err := pgx.CollectRows(rows, pgx.RowTo[string])
		fmt.Println("  balances:", balances, err)
	}
}
