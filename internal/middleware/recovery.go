package middleware

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/containeroo/mailbridge/internal/response"
)

// RecoverPanics converts handler panics into logged internal server errors.
func RecoverPanics(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					if recovered == http.ErrAbortHandler {
						panic(recovered)
					}
					logger.ErrorContext(
						r.Context(),
						"request panic",
						"event", "request_panic",
						"error", fmt.Sprint(recovered),
					)
					response.Problem(w, http.StatusInternalServerError, "The request could not be processed.")
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
