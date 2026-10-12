package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// Logger logs each request with its method, path, status code, and
// duration once the downstream handler returns.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{
			ResponseWriter: w,
		}

		next.ServeHTTP(rw, r)

		slog.InfoContext(r.Context(), "request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", rw.Status()),
			slog.Duration("duration", time.Since(start)),
		)
	})
}
