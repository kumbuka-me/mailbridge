package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/containeroo/mailbridge/internal/response"
)

// BearerToken requires the configured bearer token for a request.
func BearerToken(expected string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			provided, ok := bearerToken(r.Header.Get("Authorization"))
			if !ok || !sameSecret(provided, expected) {
				response.Problem(w, http.StatusUnauthorized, "Unauthorized.")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// bearerToken parses a case-insensitive Bearer authorization scheme.
func bearerToken(value string) (string, bool) {
	fields := strings.Fields(value)
	if len(fields) != 2 || !strings.EqualFold(fields[0], "Bearer") {
		return "", false
	}

	token := strings.TrimSpace(fields[1])
	return token, token != ""
}

// sameSecret compares two non-empty secrets without data-dependent byte comparison.
func sameSecret(provided, expected string) bool {
	if provided == "" || expected == "" || len(provided) != len(expected) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
}
