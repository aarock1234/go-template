// Package respond provides JSON response helpers for HTTP handlers.
package respond

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// internalError is written when a response body cannot be encoded.
var internalError = []byte(`{"error":"internal error","status":500}` + "\n")

// errorResponse is the payload shape used by Error.
type errorResponse struct {
	Error  string `json:"error"`
	Status int    `json:"status"`
}

// JSON writes data as a JSON response with the given status code. The body
// is encoded before anything is written, so an encoding failure produces a
// 500 error instead of a truncated response.
func JSON[T any](w http.ResponseWriter, status int, data T) {
	body, err := json.Marshal(data)
	if err != nil {
		slog.Error("encode response", slog.Any("error", err))

		status = http.StatusInternalServerError
		body = internalError
	} else {
		body = append(body, '\n')
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_, _ = w.Write(body)
}

// Error writes a JSON error response with the given status code and message.
func Error(w http.ResponseWriter, status int, message string) {
	JSON(w, status, errorResponse{
		Error:  message,
		Status: status,
	})
}

// NoContent writes a 204 No Content response with no body.
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}
