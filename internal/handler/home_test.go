package handler

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/kumbuka-me/mailbridge/web"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHome(t *testing.T) {
	t.Parallel()

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)

	Home(HomeConfig{
		Version:        "v1.2.3",
		Commit:         "abc123",
		Assets:         web.Assets,
		BodyFormat:     "html",
		SMTPTLS:        "starttls",
		SMTPRetryCount: 4,
		RateLimit:      25,
		AccessLog:      true,
	})(response, request)

	require.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, "text/html; charset=utf-8", response.Header().Get("Content-Type"))
	assert.Equal(t, "no-store", response.Header().Get("Cache-Control"))
	assert.Contains(t, response.Header().Get("Content-Security-Policy"), "default-src 'none'")
	assert.Equal(t, "DENY", response.Header().Get("X-Frame-Options"))
	assert.Contains(t, response.Body.String(), "mailbridge")
	assert.Contains(t, response.Body.String(), "v1.2.3")
	assert.Contains(t, response.Body.String(), "abc123")
	assert.Contains(t, response.Body.String(), "html")
	assert.Contains(t, response.Body.String(), "starttls")
	assert.Contains(t, response.Body.String(), "25 requests/s")
	assert.Contains(t, response.Body.String(), "enabled")
	assert.Contains(t, response.Body.String(), "POST /mail")
	assert.Contains(t, response.Body.String(), "Authorization: Bearer &lt;token&gt;")
	assert.Contains(t, response.Body.String(), "Effective configuration")
	assert.Contains(t, response.Body.String(), "Configuration reference")
}

func TestHomeConfigurationReference(t *testing.T) {
	t.Parallel()

	response := httptest.NewRecorder()
	Home(HomeConfig{Assets: web.Assets})(response, httptest.NewRequest(http.MethodGet, "/", nil))

	require.Equal(t, http.StatusOK, response.Code)

	body := response.Body.String()
	for _, value := range []string{
		"--listen-address",
		"--api-token",
		"--rate-limit",
		"--smtp-address",
		"--smtp-from",
		"--smtp-username",
		"--smtp-password",
		"--smtp-tls",
		"--smtp-insecure-skip-verify",
		"--smtp-timeout",
		"--smtp-retry-count",
		"--smtp-retry-backoff",
		"--smtp-retry-max-backoff",
		"--smtp-retry-jitter",
		"--body-format",
		"--log-format",
		"--debug",
		"--access-log",
		"MAILBRIDGE__SMTP_RETRY_COUNT",
		"MAILBRIDGE__BODY_FORMAT",
	} {
		assert.Contains(t, body, value)
	}
}

func TestHomeDisabledSettings(t *testing.T) {
	t.Parallel()

	response := httptest.NewRecorder()
	Home(HomeConfig{
		Assets:    web.Assets,
		RateLimit: 0,
		AccessLog: false,
	})(response, httptest.NewRequest(http.MethodGet, "/", nil))

	require.Equal(t, http.StatusOK, response.Code)
	assert.Contains(t, response.Body.String(), "disabled")
}

func TestHomeRenderFailure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		assets fs.FS
	}{
		{
			name:   "missing assets",
			assets: nil,
		},
		{
			name:   "missing template",
			assets: fstest.MapFS{},
		},
		{
			name: "invalid template",
			assets: fstest.MapFS{
				homeTemplatePath: &fstest.MapFile{Data: []byte(`{{`)},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			response := httptest.NewRecorder()
			Home(HomeConfig{Assets: test.assets})(
				response,
				httptest.NewRequest(http.MethodGet, "/", nil),
			)

			assert.Equal(t, http.StatusInternalServerError, response.Code)
			assert.Equal(t, "Render home page failed.\n", response.Body.String())
		})
	}
}

func TestRateLimitStatus(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "disabled", rateLimitStatus(0))
	assert.Equal(t, "disabled", rateLimitStatus(-1))
	assert.Equal(t, "10 requests/s", rateLimitStatus(10))
}
