package main

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"runtime/debug"
)

// Set at release: go build -ldflags "-X main.version=1.0.0" ./cmd/tracelite
var version = "dev"

// Version adds the commit that go build records when it builds a Git checkout.
func Version() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, s := range info.Settings {
			if s.Key == "vcs.revision" {
				return version + " (" + s.Value[:12] + ")"
			}
		}
	}
	return version + " (built outside a Git checkout)"
}

type Config struct {
	Addr        string // TRACELITE_ADDR, default 127.0.0.1:8080
	DatabaseURL string // DATABASE_URL, required
	Owner       string // TRACELITE_OWNER, required: the audit actor of a single-user install (step 13)
}

// Load reads the environment through getenv (os.Getenv in main) and reports
// every problem at once, so a customer fixes the service's settings in one go.
func Load(getenv func(string) string) (Config, error) {
	c := Config{Addr: "127.0.0.1:8080", DatabaseURL: getenv("DATABASE_URL"), Owner: getenv("TRACELITE_OWNER")}
	if a := getenv("TRACELITE_ADDR"); a != "" {
		c.Addr = a
	}
	var errs []error
	host, _, err := net.SplitHostPort(c.Addr)
	ip := net.ParseIP(host)
	if err != nil {
		errs = append(errs, fmt.Errorf("TRACELITE_ADDR %q: %w", c.Addr, err))
	} else if host != "localhost" && (ip == nil || !ip.IsLoopback()) && getenv("TRACELITE_ALLOW_REMOTE") != "yes" {
		errs = append(errs, fmt.Errorf("TRACELITE_ADDR %q is reachable from other machines, and TraceLite has no login; "+
			"set TRACELITE_ALLOW_REMOTE=yes behind a proxy that authenticates", c.Addr))
	}
	if u, err := url.Parse(c.DatabaseURL); err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") {
		errs = append(errs, errors.New("DATABASE_URL is required, as postgres://user:password@host:port/database"))
	}
	if c.Owner == "" {
		errs = append(errs, errors.New("TRACELITE_OWNER must name the person using this installation"))
	}
	return c, errors.Join(errs...)
}

func main() {
	fmt.Println("tracelite", Version())
	for _, env := range []map[string]string{
		{"DATABASE_URL": "postgres://tracelite:Gizli-2026@127.0.0.1:5432/tracelite", "TRACELITE_OWNER": "Ayşe Yılmaz"},
		{"TRACELITE_ADDR": ":8080"},
	} {
		c, err := Load(func(k string) string { return env[k] })
		if err != nil {
			fmt.Printf("refusing to start:\n%v\n", err)
			continue
		}
		db, _ := url.Parse(c.DatabaseURL) // Load has checked that it parses
		fmt.Printf("starting: addr=%s db=%s owner=%s\n", c.Addr, db.Redacted(), c.Owner)
	}
}
