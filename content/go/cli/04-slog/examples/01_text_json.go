package main

import (
	"errors"
	"log/slog"
	"os"
	"time"
)

// dropTime removes the timestamp so this output is reproducible. Real
// programs keep it; the function is the ReplaceAttr example from the docs.
func dropTime(groups []string, a slog.Attr) slog.Attr {
	if a.Key == slog.TimeKey && len(groups) == 0 {
		return slog.Attr{} // a zero Attr is dropped
	}
	return a
}

func main() {
	slog.Info("default logger: stderr, with a timestamp") // not compared

	opts := &slog.HandlerOptions{ReplaceAttr: dropTime}
	for _, h := range []slog.Handler{
		slog.NewTextHandler(os.Stdout, opts), // for people
		slog.NewJSONHandler(os.Stdout, opts), // for machines
	} {
		logger := slog.New(h)
		logger.Debug("not shown: the default minimum level is Info")
		logger.Info("signed in", "user", "Ayşe Yılmaz", "city", "İzmir", "attempt", 2)
		logger.Warn("slow query",
			slog.Duration("took", 1500*time.Millisecond),
			slog.Group("db", slog.String("table", "orders"), slog.Int("rows", 1200)))
		logger.Error("payment failed", "err", errors.New("card declined"), "amount", 99.9)
	}
}
