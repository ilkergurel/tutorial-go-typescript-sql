package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

func main() {
	// signal.NotifyContext in TraceLite; here a goroutine below plays the signal.
	ctx, sigterm := context.WithCancel(context.Background())

	// Start in dependency order: pool, migrations, jobs, listener, HTTP.
	db, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	check(err)
	names, _ := fs.Glob(migrationFiles, "migrations/*.sql")
	for _, name := range names { // goose.Up in TraceLite: only what isn't applied yet
		sql, _ := migrationFiles.ReadFile(name)
		_, err := db.Exec(ctx, string(sql))
		check(err) // exit rather than serve a schema the code doesn't expect
	}
	fmt.Printf("migrations: %d applied, from inside the binary\n", len(names))

	jobsCtx, stopJobs := context.WithCancel(context.WithoutCancel(ctx))
	jobsDone := make(chan struct{})
	go func() { <-jobsCtx.Done(); close(jobsDone) }() // step 11's Pool.Run(jobsCtx), reduced to its lifetime
	fmt.Println("jobs: 4 workers claiming")

	started := make(chan struct{})
	srv := &http.Server{ReadHeaderTimeout: 5 * time.Second, Handler: http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			close(started)
			time.Sleep(300 * time.Millisecond) // still running when the signal arrives
			w.Write([]byte(`{"items": [], "nextCursor": null}`))
		})}
	ln, err := net.Listen("tcp", "127.0.0.1:0") // TRACELITE_ADDR
	check(err)
	go srv.Serve(ln)
	fmt.Println("http: serving")

	answer := make(chan string)
	go func() {
		resp, err := http.Get("http://" + ln.Addr().String() + "/api/projects")
		check(err)
		resp.Body.Close()
		answer <- resp.Status
	}()
	go func() { <-started; sigterm() }()

	<-ctx.Done()
	fmt.Println("signal: shutting down in reverse order")
	stopCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	check(srv.Shutdown(stopCtx)) // no new connections; requests in flight finish first
	fmt.Println("http: stopped; the request in flight got", <-answer)
	stopJobs() // after HTTP: a last request may still have queued a job
	<-jobsDone
	fmt.Println("jobs: drained")
	fmt.Println("listener (step 9): closed")
	db.Close()
	fmt.Println("database pool: closed")
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
