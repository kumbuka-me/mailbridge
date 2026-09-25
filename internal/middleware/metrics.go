package middleware

import (
	"net/http"
	"time"
)

// MailMetrics records completed mail endpoint requests.
type MailMetrics interface {
	ObserveMailRequest(status int, duration time.Duration)
}

// InstrumentMail records status and duration for requests reaching the mail endpoint.
func InstrumentMail(metrics MailMetrics) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			response := &statusWriter{ResponseWriter: w}

			next.ServeHTTP(response, r)

			status := response.status
			if status == 0 {
				status = http.StatusOK
			}
			metrics.ObserveMailRequest(status, time.Since(started))
		})
	}
}
