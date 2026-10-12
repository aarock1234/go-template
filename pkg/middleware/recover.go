package middleware

import (
	"errors"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/aarock1234/go-template/pkg/respond"
)

// Recover catches panics in downstream handlers, logs them with a stack
// trace, and responds with a 500 JSON error unless the response has
// already started. http.ErrAbortHandler is re-panicked so the server
// aborts the response as the handler intended.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := &responseWriter{
			ResponseWriter: w,
		}

		defer func() {
			v := recover()
			if v == nil {
				return
			}

			if err, ok := v.(error); ok && errors.Is(err, http.ErrAbortHandler) {
				panic(v)
			}

			slog.ErrorContext(r.Context(), "panic recovered",
				slog.Any("panic", v),
				slog.String("stack", string(debug.Stack())),
			)

			if !rw.wrote {
				respond.Error(rw, http.StatusInternalServerError, "internal error")
			}
		}()

		next.ServeHTTP(rw, r)
	})
}
