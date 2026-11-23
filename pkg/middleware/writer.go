package middleware

import "net/http"

// responseWriter wraps an http.ResponseWriter to record whether the
// response has started and with which status code.
type responseWriter struct {
	http.ResponseWriter
	status int
	wrote  bool
}

// WriteHeader delegates, then records the first final status code. An
// informational 1xx code does not start the response, and a code that
// net/http rejects by panicking is never recorded, so panic recovery can
// still answer.
func (w *responseWriter) WriteHeader(code int) {
	w.ResponseWriter.WriteHeader(code)

	if !w.wrote && code >= http.StatusOK {
		w.status = code
		w.wrote = true
	}
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
