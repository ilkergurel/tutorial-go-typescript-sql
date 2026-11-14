package main

import (
	"log"
	"log/slog"
	"os"
)

func dropTime(groups []string, a slog.Attr) slog.Attr {
	if a.Key == slog.TimeKey && len(groups) == 0 {
		return slog.Attr{}
	}
	return a
}

func main() {
	// Machines get every record from Info up, as JSON on stdout (the log
	// collector). People get warnings and errors as text on stderr.
	machines := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{ReplaceAttr: dropTime})
	people := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelWarn, ReplaceAttr: dropTime,
	})
	slog.SetDefault(slog.New(slog.NewMultiHandler(machines, people))) // Go 1.26

	slog.Info("cache warmed", "entries", 1024)
	slog.Warn("disk almost full", "free_pct", 4)

	// After SetDefault, the old log package writes through the same handler.
	log.Printf("legacy code still calls log.Printf: %d retries", 3)
}
