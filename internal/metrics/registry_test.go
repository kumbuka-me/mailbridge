package metrics

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistryExposesBuildAndRuntimeMetrics(t *testing.T) {
	t.Parallel()

	registry := NewRegistry("1.2.3", "abc123")
	output := scrapeMetrics(t, registry)

	assert.Contains(t, output, `mailbridge_build_info{commit="abc123",version="1.2.3"} 1`)
	assert.Contains(t, output, "go_goroutines")
	assert.Contains(t, output, "process_cpu_seconds_total")
}

func TestRegistryObservesMailRequests(t *testing.T) {
	t.Parallel()

	registry := NewRegistry("test", "test")
	registry.ObserveMailRequest(http.StatusNoContent, 25*time.Millisecond)
	registry.ObserveMailRequest(http.StatusUnauthorized, 10*time.Millisecond)

	output := scrapeMetrics(t, registry)
	assert.Contains(t, output, `mailbridge_mail_requests_total{code="204"} 1`)
	assert.Contains(t, output, `mailbridge_mail_requests_total{code="401"} 1`)
	assert.Contains(t, output, "mailbridge_mail_request_duration_seconds_count 2")
}

func TestRegistryObservesEmailDeliveries(t *testing.T) {
	t.Parallel()

	registry := NewRegistry("test", "test")
	registry.ObserveEmailDelivery(25*time.Millisecond, nil)
	registry.ObserveEmailDelivery(10*time.Millisecond, errors.New("smtp unavailable"))

	output := scrapeMetrics(t, registry)
	assert.Contains(t, output, "mailbridge_email_deliveries_total 2")
	assert.Contains(t, output, "mailbridge_email_delivery_errors_total 1")
	assert.Contains(t, output, "mailbridge_email_delivery_duration_seconds_count 2")
}

// scrapeMetrics returns the text exposition produced by the registry handler.
func scrapeMetrics(t *testing.T, registry *Registry) string {
	t.Helper()

	request := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	response := httptest.NewRecorder()
	registry.Metrics().ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code)
	body, err := io.ReadAll(response.Result().Body)
	require.NoError(t, err)
	require.NoError(t, response.Result().Body.Close())
	return strings.TrimSpace(string(body))
}
