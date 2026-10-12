package middleware

import "net/http"

// responseWriter wraps an http.ResponseWriter to record whether the
// response has started and with which status code.
type responseWriter struct {
	http.ResponseWriter
	status int
	wrote  bool
}

// WriteHeader records the first status code before delegating.
func (w *responseWriter) WriteHeader(code int) {
	if !w.wrote {
		w.status = code
		w.wrote = true
	}

	w.ResponseWriter.WriteHeader(code)
}

// Write records an implicit 200 status on the first write.
func (w *responseWriter) Write(b []byte) (int, error) {
	if !w.wrote {
		w.status = http.StatusOK
		w.wrote = true
	}

	return w.ResponseWriter.Write(b)
}

// Status returns the recorded status code, or 200 if the handler never
// wrote a response, which is what net/http sends in that case.
func (w *responseWriter) Status() int {
	if !w.wrote {
		return http.StatusOK
	}

	return w.status
}

// Unwrap returns the underlying ResponseWriter, allowing
// http.ResponseController to reach it.
func (w *responseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
