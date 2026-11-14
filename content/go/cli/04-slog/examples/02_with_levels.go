package main

import (
	"log/slog"
	"os"
)

type Token string

// The built-in handlers call LogValue, so no slog call can print the secret.
func (Token) LogValue() slog.Value { return slog.StringValue("REDACTED") }

type User struct {
	ID    int
	Email string
}

// LogValue chooses what a User looks like in logs: the ID, not the e-mail.
func (u User) LogValue() slog.Value { return slog.GroupValue(slog.Int("id", u.ID)) }

func dropTime(groups []string, a slog.Attr) slog.Attr {
	if a.Key == slog.TimeKey && len(groups) == 0 {
		return slog.Attr{}
	}
	return a
}

func main() {
	level := new(slog.LevelVar) // starts at Info; safe to change concurrently
	h := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level, ReplaceAttr: dropTime})
	logger := slog.New(h)

	// With returns a logger that adds these attributes to every record.
	reqLog := logger.With("request_id", "r-42", "method", "POST")
	reqLog.Info("login", "user", User{ID: 7, Email: "ayse@example.com"}, "token", Token("s3cr3t"))
	reqLog.Debug("cache lookup", "hit", false) // below Info: dropped

	level.Set(slog.LevelDebug) // e.g. from an admin endpoint, no restart
	reqLog.Debug("cache lookup", "hit", false)

	reqLog.WithGroup("payment").Info("charged", "amount", 120, "currency", "TRY")

	// A missing value compiles; go vet reports "missing a final value".
	reqLog.Info("odd arguments", "user")
}
