package server

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/containeroo/mailbridge/internal/application"
	appmetrics "github.com/containeroo/mailbridge/internal/metrics"
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

	t.Run("health", func(t *testing.T) {
		t.Parallel()
		handler, _, _ := newTestServer()
		response := httptest.NewRecorder()

		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/healthz", nil))

		assert.Equal(t, http.StatusOK, response.Code)
		assert.JSONEq(t, `{"status":"ok"}`, response.Body.String())
	})

	t.Run("metrics", func(t *testing.T) {
		t.Parallel()
		handler, _, _ := newTestServer()
		response := httptest.NewRecorder()

		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/metrics", nil))

		assert.Equal(t, http.StatusOK, response.Code)
		assert.Contains(t, response.Body.String(), `mailbridge_build_info{commit="abc123",version="test"} 1`)
		assert.NotContains(t, response.Body.String(), "mailbridge_mail_requests_total")
	})

	t.Run("health is not instrumented as mail", func(t *testing.T) {
		t.Parallel()
		handler, _, _ := newTestServer()

		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/healthz", nil))
		metricsResponse := httptest.NewRecorder()
		handler.ServeHTTP(metricsResponse, httptest.NewRequest(http.MethodGet, "/metrics", nil))

		assert.NotContains(t, metricsResponse.Body.String(), "mailbridge_mail_requests_total")
	})

	t.Run("mail requires bearer token and records status", func(t *testing.T) {
		t.Parallel()
		handler, _, _ := newTestServer()
		response := httptest.NewRecorder()

		handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/mail", bytes.NewBufferString(`{}`)))

		assert.Equal(t, http.StatusUnauthorized, response.Code)
		metricsResponse := httptest.NewRecorder()
		handler.ServeHTTP(metricsResponse, httptest.NewRequest(http.MethodGet, "/metrics", nil))
		assert.Contains(t, metricsResponse.Body.String(), `mailbridge_mail_requests_total{code="401"} 1`)
	})

	t.Run("mail forwards request and records status", func(t *testing.T) {
		t.Parallel()
		handler, forwarder, _ := newTestServer()
		body := []byte(`{"recipients":{"to":[{"email":"alice@example.com"}]},"message":{"subject":"Updated","body":"Changed"}}`)
		request := httptest.NewRequest(http.MethodPost, "/mail", bytes.NewReader(body))
		request.Header.Set("Authorization", "Bearer secret")
		response := httptest.NewRecorder()

		handler.ServeHTTP(response, request)

		require.Equal(t, http.StatusNoContent, response.Code)
		require.Len(t, forwarder.request.Recipients.To, 1)
		assert.Equal(t, "alice@example.com", forwarder.request.Recipients.To[0].Email)

		metricsResponse := httptest.NewRecorder()
		handler.ServeHTTP(metricsResponse, httptest.NewRequest(http.MethodGet, "/metrics", nil))
		assert.Contains(t, metricsResponse.Body.String(), `mailbridge_mail_requests_total{code="204"} 1`)
	})
}

// newTestServer constructs an isolated routed application for one server test.
func newTestServer() (http.Handler, *forwarderStub, *appmetrics.Registry) {
	forwarder := &forwarderStub{}
	metrics := appmetrics.NewRegistry("test", "abc123")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := New(Config{
		Forwarder: forwarder,
		APIToken:  "secret",
		Logger:    logger,
		Metrics:   metrics,
	})
	return handler, forwarder, metrics
}
