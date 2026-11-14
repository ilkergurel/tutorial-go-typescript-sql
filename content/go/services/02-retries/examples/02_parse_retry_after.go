package main

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// retryAfter reads both forms of Retry-After. A date is measured against the
// response's Date header, so a skewed local clock doesn't matter.
func retryAfter(h http.Header) (time.Duration, bool) {
	v := h.Get("Retry-After")
	if secs, err := strconv.Atoi(v); err == nil && secs >= 0 {
		return time.Duration(secs) * time.Second, true
	}
	at, err := http.ParseTime(v)
	if err != nil {
		return 0, false
	}
	now, err := http.ParseTime(h.Get("Date"))
	if err != nil {
		now = time.Now()
	}
	return max(at.Sub(now), 0), true
}

func main() {
	for _, v := range []string{
		"120",                            // delay-seconds
		"Fri, 13 Nov 2026 10:00:30 GMT",  // HTTP-date, the preferred format
		"Friday, 13-Nov-26 10:00:30 GMT", // obsolete RFC 850 format, still allowed
		"Fri, 13 Nov 2026 09:59:00 GMT",  // already past
		"-5",
	} {
		h := http.Header{}
		h.Set("Date", "Fri, 13 Nov 2026 10:00:00 GMT") // the server's clock
		h.Set("Retry-After", v)
		wait, ok := retryAfter(h)
		fmt.Printf("%-32q wait=%-4v hint=%t\n", v, wait, ok)
	}
}
