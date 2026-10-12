// Package middleware provides HTTP middleware for request logging,
// panic recovery, and request ID propagation.
package middleware

import (
	"crypto/rand"
	"net/http"

	"github.com/aarock1234/go-template/pkg/log"
)

const (
	headerRequestID = "X-Request-ID"

	// maxRequestIDLength bounds the length of an accepted request ID.
	maxRequestIDLength = 128
)

// RequestID propagates the request's X-Request-ID header, or generates an
// ID when the header is missing or unsafe to log. The ID is stored in the
// request context for logging and echoed on the response.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(headerRequestID)
		if !validRequestID(id) {
			id = rand.Text()
		}

		w.Header().Set(headerRequestID, id)

		next.ServeHTTP(w, r.WithContext(log.WithRequestID(r.Context(), id)))
	})
}

// validRequestID reports whether id is a bounded token of letters, digits,
// and the punctuation used by common trace ID formats, so client input
// cannot forge log content.
func validRequestID(id string) bool {
	if id == "" || len(id) > maxRequestIDLength {
		return false
	}

	for i := range len(id) {
		if !requestIDByte(id[i]) {
			return false
		}
	}

	return true
}

func requestIDByte(c byte) bool {
	switch {
	case 'a' <= c && c <= 'z', 'A' <= c && c <= 'Z', '0' <= c && c <= '9':
		return true
	case c == '-', c == '_', c == '.', c == ':':
		return true
	default:
		return false
	}
}
