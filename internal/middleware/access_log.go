package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// AccessLog records one structured completion event per HTTP request.
func AccessLog(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			response := &statusWriter{ResponseWriter: w}

			next.ServeHTTP(response, r)

			status := response.status
			if status == 0 {
				status = http.StatusOK
			}
			logger.InfoContext(
				r.Context(),
				"request",
				"event", "request_complete",
				"method", r.Method,
				"path", r.URL.Path,
				"status", status,
				"duration", time.Since(started).String(),
			)
		})
	}
}
