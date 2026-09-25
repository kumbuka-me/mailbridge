package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRateLimit(t *testing.T) {
	t.Parallel()

	t.Run("disabled", func(t *testing.T) {
		t.Parallel()

		calls := 0
		handler := RateLimit(0)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			calls++
			w.WriteHeader(http.StatusNoContent)
		}))

		for range 3 {
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/mail", nil))
			assert.Equal(t, http.StatusNoContent, response.Code)
		}
		assert.Equal(t, 3, calls)
	})

	t.Run("rejects requests above configured rate", func(t *testing.T) {
		t.Parallel()

		calls := 0
		handler := RateLimit(1)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			calls++
			w.WriteHeader(http.StatusNoContent)
		}))

		first := httptest.NewRecorder()
		handler.ServeHTTP(first, httptest.NewRequest(http.MethodPost, "/mail", nil))
		assert.Equal(t, http.StatusNoContent, first.Code)

		second := httptest.NewRecorder()
		handler.ServeHTTP(second, httptest.NewRequest(http.MethodPost, "/mail", nil))
		assert.Equal(t, http.StatusTooManyRequests, second.Code)
		assert.Equal(t, "1", second.Header().Get("Retry-After"))
		assert.JSONEq(t, `{"error":"Too many requests."}`, second.Body.String())
		assert.Equal(t, 1, calls)
	})
}
