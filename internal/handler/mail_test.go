package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/containeroo/mailbridge/internal/application"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// forwarderStub records requests delivered by handler tests.
type forwarderStub struct {
	request application.Request
	err     error
}

// Forward records the supplied request and returns the configured error.
func (s *forwarderStub) Forward(_ context.Context, request application.Request) error {
	s.request = request
	return s.err
}

func TestMail(t *testing.T) {
	t.Parallel()

	var logs strings.Builder
	forwarder := &forwarderStub{}
	body := []byte(`{"recipients":{"to":[{"email":"alice@example.com","display_name":"Alice"}],"cc":[{"email":"charlie@example.com","display_name":"Charlie"}],"bcc":[{"email":"audit@example.com"}]},"message":{"subject":"Updated","body":"<p>Changed</p>","body_format":"html"}}`)
	request := httptest.NewRequest(http.MethodPost, "/mail", bytes.NewReader(body))
	response := httptest.NewRecorder()

	Mail(forwarder, slog.New(slog.NewTextHandler(&logs, nil)))(response, request)

	require.Equal(t, http.StatusNoContent, response.Code)
	require.Len(t, forwarder.request.Recipients.To, 1)
	assert.Equal(t, "alice@example.com", forwarder.request.Recipients.To[0].Email)
	assert.Equal(t, application.BodyFormatHTML, forwarder.request.Message.BodyFormat)
	assert.Contains(t, logs.String(), "email_delivery_succeeded")
	assert.Contains(t, logs.String(), "to_count=1")
	assert.Contains(t, logs.String(), "cc_count=1")
	assert.Contains(t, logs.String(), "bcc_count=1")
	assert.NotContains(t, logs.String(), "alice@example.com")
}

// TestMailLogsHumanReadableDuration verifies delivery duration is encoded as a string.
func TestMailLogsHumanReadableDuration(t *testing.T) {
	t.Parallel()

	var logs bytes.Buffer
	body := []byte(`{"recipients":{"to":[{"email":"alice@example.com"}]},"message":{"subject":"Updated","body":"Changed"}}`)
	request := httptest.NewRequest(http.MethodPost, "/mail", bytes.NewReader(body))
	response := httptest.NewRecorder()

	Mail(&forwarderStub{}, slog.New(slog.NewJSONHandler(&logs, nil)))(response, request)

	require.Equal(t, http.StatusNoContent, response.Code)
	var entry map[string]any
	require.NoError(t, json.Unmarshal(logs.Bytes(), &entry))
	assert.IsType(t, "", entry["duration"])
	assert.NotEmpty(t, entry["duration"])
}

func TestMailRejectsTrailingJSON(t *testing.T) {
	t.Parallel()

	body := `{"recipients":{}} {"recipients":{}}`
	request := httptest.NewRequest(http.MethodPost, "/mail", bytes.NewBufferString(body))
	response := httptest.NewRecorder()

	Mail(&forwarderStub{}, nil)(response, request)

	assert.Equal(t, http.StatusBadRequest, response.Code)
}

func TestMailRejectsUnknownFields(t *testing.T) {
	t.Parallel()

	body := `{"recipients":{},"message":{},"unexpected":true}`
	request := httptest.NewRequest(http.MethodPost, "/mail", bytes.NewBufferString(body))
	response := httptest.NewRecorder()

	Mail(&forwarderStub{}, nil)(response, request)

	assert.Equal(t, http.StatusBadRequest, response.Code)
}

func TestMailReportsValidationFailure(t *testing.T) {
	t.Parallel()

	forwarder := &forwarderStub{err: application.ValidateRequest(application.Request{})}
	request := httptest.NewRequest(http.MethodPost, "/mail", bytes.NewBufferString(`{"recipients":{},"message":{}}`))
	response := httptest.NewRecorder()

	Mail(forwarder, nil)(response, request)

	assert.Equal(t, http.StatusBadRequest, response.Code)
}

func TestMailReportsDeliveryFailure(t *testing.T) {
	t.Parallel()

	var logs strings.Builder
	forwarder := &forwarderStub{err: errors.New("smtp unavailable")}
	request := httptest.NewRequest(http.MethodPost, "/mail", bytes.NewBufferString(`{"recipients":{"to":[{"email":"alice@example.com"}]},"message":{"subject":"Updated","body":"Changed"}}`))
	response := httptest.NewRecorder()

	Mail(forwarder, slog.New(slog.NewTextHandler(&logs, nil)))(response, request)

	assert.Equal(t, http.StatusBadGateway, response.Code)
	assert.Contains(t, logs.String(), "email_delivery_failed")
	assert.Contains(t, logs.String(), "to_count=1")
	assert.NotContains(t, logs.String(), "alice@example.com")
}
