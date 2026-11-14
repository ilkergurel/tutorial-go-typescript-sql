package httpapi

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"
)

// statusRecorder captures the status code for response logging; WriteHeader may
// never be called (implicit 200), so the default must be 200, not 0.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// Unwrap lets http.ResponseController reach the real connection through this
// wrapper. Without it, the run route's longer write deadline fails with
// ErrNotSupported and the 15s server WriteTimeout cuts slow runs off mid-reply.
func (r *statusRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}

// withLogging is the middleware pattern the http lessons teach: middleware is a
// function from http.Handler to http.Handler, wrapped innermost-last.
func withLogging(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)
			logger.Info("http request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", rec.status),
				slog.Duration("duration", time.Since(start)),
			)
		})
	}
}

// withRecovery turns a panicking handler into a logged 500 problem response.
// Without it net/http recovers the panic itself, logs it and drops the
// connection: the browser reads EOF instead of an error it can show.
//
// It must sit inside withLogging: a panic unwinds through every wrapper
// outside the recover, so the logging middleware only writes its line (with
// status 500) when the panic stops here.
//
// Two cases still end in a dropped connection, on purpose:
//   - http.ErrAbortHandler is how a handler asks net/http to abort quietly, so
//     it is re-panicked untouched.
//   - Once the handler has written anything, the status line may be on the
//     wire and a 500 can no longer replace it; finishing normally would end a
//     half-written body as if it were complete. Aborting tells the client the
//     reply is broken.
func withRecovery(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tw := &writeTracker{ResponseWriter: w}
			defer func() {
				v := recover()
				if v == nil {
					return
				}
				// net/http compares with ==, not errors.Is; match it.
				if v == http.ErrAbortHandler {
					panic(v)
				}
				logger.Error("handler panicked",
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.Any("panic", v),
					// Still the panicking goroutine's stack: deferred calls
					// run before it unwinds.
					slog.String("stack", string(debug.Stack())),
				)
				if tw.wrote {
					panic(http.ErrAbortHandler)
				}
				problem(w, http.StatusInternalServerError, "internal server error")
			}()
			next.ServeHTTP(tw, r)
		})
	}
}

// writeTracker records whether the handler has started its response.
type writeTracker struct {
	http.ResponseWriter
	wrote bool
}

func (t *writeTracker) WriteHeader(code int) {
	t.wrote = true
	t.ResponseWriter.WriteHeader(code)
}

func (t *writeTracker) Write(b []byte) (int, error) {
	t.wrote = true
	return t.ResponseWriter.Write(b)
}

// Unwrap keeps http.ResponseController working through this wrapper, as on
// statusRecorder: the run route extends its write deadline through it.
func (t *writeTracker) Unwrap() http.ResponseWriter {
	return t.ResponseWriter
}

// FlushError counts a flush as a write: it sends the status line. Without it,
// http.ResponseController would flush through Unwrap and bypass the tracker.
func (t *writeTracker) FlushError() error {
	t.wrote = true
	return http.NewResponseController(t.ResponseWriter).Flush()
}

// withProblemFallbacks makes the mux's own 404 and 405 answers problem+json,
// like every other error. ServeMux has no hook for them: it writes them with
// http.Error as text/plain. mux.Handler reports an empty pattern when no route
// will handle the request, so only the mux's own replies (404, 405, or a
// path-cleaning redirect, which passes through) get the rewriting writer; a
// handler's own 404 is never touched.
//
// Rejected alternative: a catch-all "/" route. It matches every method, so a
// request that should get 405 (GET /api/run) would get the catch-all's 404.
func withProblemFallbacks(mux *http.ServeMux) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, pattern := mux.Handler(r); pattern == "" {
			w = &fallbackWriter{ResponseWriter: w, r: r}
		}
		// mux.ServeHTTP, not the handler Handler returned: only ServeHTTP
		// fills in r.PathValue.
		mux.ServeHTTP(w, r)
	})
}

// fallbackWriter swaps the mux's text/plain 404 and 405 for problem(). The
// mux sets Allow before it writes a 405, so the header survives. Other
// statuses (the mux's path-cleaning redirects) pass through.
type fallbackWriter struct {
	http.ResponseWriter
	r        *http.Request
	replaced bool
}

func (f *fallbackWriter) WriteHeader(code int) {
	switch code {
	case http.StatusNotFound:
		f.replaced = true
		problem(f.ResponseWriter, code, "no route for "+f.r.Method+" "+f.r.URL.Path)
	case http.StatusMethodNotAllowed:
		f.replaced = true
		problem(f.ResponseWriter, code, f.r.Method+" is not allowed on "+f.r.URL.Path+
			"; allowed: "+f.Header().Get("Allow"))
	default:
		f.ResponseWriter.WriteHeader(code)
	}
}

// Write drops http.Error's text body once problem() has answered.
func (f *fallbackWriter) Write(b []byte) (int, error) {
	if f.replaced {
		return len(b), nil
	}
	return f.ResponseWriter.Write(b)
}
