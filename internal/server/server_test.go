package server

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/containeroo/mailbridge/internal/application"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// forwarderStub records the request delivered through the routed HTTP application.
type forwarderStub struct {
	request application.Request
}

// Forward records the supplied request.
func (s *forwarderStub) Forward(_ context.Context, request application.Request) error {
	s.request = request
	return nil
}

func TestRoutes(t *testing.T) {
	t.Parallel()

	forwarder := &forwarderStub{}
	handler := New(Config{Forwarder: forwarder, APIToken: "secret"})

	t.Run("health", func(t *testing.T) {
		t.Parallel()
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/healthz", nil))
		assert.Equal(t, http.StatusOK, response.Code)
		assert.JSONEq(t, `{"status":"ok"}`, response.Body.String())
	})

	t.Run("mail requires bearer token", func(t *testing.T) {
		t.Parallel()
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/mail", bytes.NewBufferString(`{}`)))
		assert.Equal(t, http.StatusUnauthorized, response.Code)
	})

	t.Run("mail forwards request", func(t *testing.T) {
		body := []byte(`{"recipients":{"to":[{"email":"alice@example.com"}]},"message":{"subject":"Updated","body":"Changed"}}`)
		request := httptest.NewRequest(http.MethodPost, "/mail", bytes.NewReader(body))
		request.Header.Set("Authorization", "Bearer secret")
		response := httptest.NewRecorder()

		handler.ServeHTTP(response, request)

		require.Equal(t, http.StatusNoContent, response.Code)
		require.Len(t, forwarder.request.Recipients.To, 1)
		assert.Equal(t, "alice@example.com", forwarder.request.Recipients.To[0].Email)
	})
}
