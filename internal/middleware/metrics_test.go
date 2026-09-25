package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// mailMetricsStub records one observed mail request.
type mailMetricsStub struct {
	calls    int
	status   int
	duration time.Duration
}

// ObserveMailRequest records one completed request observation.
func (s *mailMetricsStub) ObserveMailRequest(status int, duration time.Duration) {
	s.calls++
	s.status = status
	s.duration = duration
}

func TestInstrumentMail(t *testing.T) {
	t.Parallel()

	metrics := &mailMetricsStub{}
	handler := InstrumentMail(metrics)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/mail", nil))

	assert.Equal(t, http.StatusNoContent, response.Code)
	assert.Equal(t, 1, metrics.calls)
	assert.Equal(t, http.StatusNoContent, metrics.status)
	assert.GreaterOrEqual(t, metrics.duration, time.Duration(0))
}

func TestInstrumentMailRecordsImplicitOK(t *testing.T) {
	t.Parallel()

	metrics := &mailMetricsStub{}
	handler := InstrumentMail(metrics)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/mail", nil))

	assert.Equal(t, 1, metrics.calls)
	assert.Equal(t, http.StatusOK, metrics.status)
}
