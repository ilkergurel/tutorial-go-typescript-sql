package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
)

const schema = `
CREATE TABLE events (id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY, body text NOT NULL);
CREATE FUNCTION events_changed() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  PERFORM pg_notify(current_schema() || '_events', ''); -- a wake-up call, no data
  RETURN NULL;
END $$;
CREATE TRIGGER events_changed AFTER INSERT ON events
  FOR EACH STATEMENT EXECUTE FUNCTION events_changed();`

// consume listens on its own connection and sends out every new event.
func consume(ctx context.Context, url string, out chan<- string) error {
	conn, err := pgx.Connect(ctx, url)
	if err != nil {
		return err
	}
	defer conn.Close(context.Background())
	var channel string
	if err := conn.QueryRow(ctx, "SELECT current_schema() || '_events'").Scan(&channel); err != nil {
		return err
	}
	if _, err := conn.Exec(ctx, "LISTEN "+pgx.Identifier{channel}.Sanitize()); err != nil {
		return err
	}
	var last int64 // in production: persisted, so a restart resumes from here
	for {
		var id int64
		var body string
		rows, _ := conn.Query(ctx, "SELECT id, body FROM events WHERE id > $1 ORDER BY id", last)
		_, err := pgx.ForEachRow(rows, []any{&id, &body}, func() error {
			out <- fmt.Sprintf("#%d %s", id, body)
			last = id
			return nil
		})
		if err != nil {
			return err
		}
		if _, err := conn.WaitForNotification(ctx); err != nil {
			return err // cancelled, or the connection broke: reconnect and catch up
		}
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	writer, err := pgx.Connect(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer writer.Close(context.Background())
	exec := func(sql string) {
		if _, err := writer.Exec(ctx, sql); err != nil {
			log.Fatal(err)
		}
	}

	exec(schema)
	exec("INSERT INTO events (body) VALUES ('order 1 created')") // nobody is listening yet
	handled, stopped := make(chan string), make(chan error, 1)
	go func() { stopped <- consume(ctx, os.Getenv("DATABASE_URL"), handled) }()
	fmt.Println("catch-up at start:", <-handled)

	exec("INSERT INTO events (body) VALUES ('order 2 created'), ('order 2 paid'), ('order 3 created')")
	fmt.Println("one INSERT of three rows:", <-handled, "|", <-handled, "|", <-handled)
	cancel()
	fmt.Println("consumer stopped, cancelled:", errors.Is(<-stopped, context.Canceled))
}
