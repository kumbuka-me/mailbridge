package middleware

import "net/http"

// statusWriter records the first HTTP status written by a handler.
type statusWriter struct {
	http.ResponseWriter
	status int
}

// WriteHeader records and forwards the first response status.
func (w *statusWriter) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

// Write records an implicit 200 response before forwarding the body.
func (w *statusWriter) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(body)
}
