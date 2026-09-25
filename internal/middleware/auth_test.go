package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBearerToken(t *testing.T) {
	t.Parallel()

	handler := BearerToken("secret")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	tests := []struct {
		name          string
		authorization string
		status        int
	}{
		{name: "valid", authorization: "Bearer secret", status: http.StatusNoContent},
		{name: "case insensitive scheme", authorization: "bearer secret", status: http.StatusNoContent},
		{name: "missing", status: http.StatusUnauthorized},
		{name: "wrong token", authorization: "Bearer other", status: http.StatusUnauthorized},
		{name: "wrong scheme", authorization: "Basic secret", status: http.StatusUnauthorized},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			request := httptest.NewRequest(http.MethodPost, "/", nil)
			request.Header.Set("Authorization", test.authorization)
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			assert.Equal(t, test.status, response.Code)
		})
	}
}
