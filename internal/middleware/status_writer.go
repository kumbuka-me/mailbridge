package middleware

import "net/http"

// statusWriter records the final HTTP status written by a handler.
type statusWriter struct {
	// ResponseWriter receives the original headers and response body.
	http.ResponseWriter
	// status is the final response code, or zero before the response is committed.
	status int
}

// WriteHeader forwards informational responses and records the first final status.
func (w *statusWriter) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	if status >= 100 && status < 200 && status != http.StatusSwitchingProtocols {
		w.ResponseWriter.WriteHeader(status)
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

// Unwrap exposes transport capabilities through http.ResponseController.
func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
