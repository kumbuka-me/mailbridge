package flags

import (
	"testing"
	"time"

	"github.com/containeroo/mailbridge/internal/application"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	t.Parallel()

	cfg, err := Parse([]string{
		"--api-token", "secret",
		"--smtp-address", "smtp.example.com:587",
		"--smtp-from", "mailbridge <mailbridge@example.com>",
	}, "test")

	require.NoError(t, err)
	assert.Equal(t, "127.0.0.1:8080", cfg.ListenAddress)
	assert.Equal(t, "secret", cfg.APIToken)
	assert.Equal(t, SMTPTLSStartTLS, cfg.SMTPTLS)
	assert.Equal(t, 30*time.Second, cfg.SMTPTimeout)
	assert.Equal(t, 3, cfg.SMTPRetryCount)
	assert.Equal(t, time.Second, cfg.SMTPRetryBackoff)
	assert.Equal(t, 30*time.Second, cfg.SMTPRetryMaxBackoff)
	assert.True(t, cfg.SMTPRetryJitter)
	assert.Equal(t, application.BodyFormatText, cfg.BodyFormat)
	assert.Zero(t, cfg.RateLimit)
}

func TestRateLimit(t *testing.T) {
	t.Parallel()

	cfg, err := Parse([]string{
		"--api-token", "secret",
		"--smtp-address", "smtp.example.com:587",
		"--smtp-from", "mailbridge@example.com",
		"--rate-limit", "25",
	}, "test")

	require.NoError(t, err)
	assert.Equal(t, 25, cfg.RateLimit)
}

func TestRateLimitMustNotBeNegative(t *testing.T) {
	t.Parallel()

	_, err := Parse([]string{
		"--api-token", "secret",
		"--smtp-address", "smtp.example.com:587",
		"--smtp-from", "mailbridge@example.com",
		"--rate-limit", "-1",
	}, "test")

	require.Error(t, err)
}

func TestEnvironment(t *testing.T) {
	t.Setenv("JSON2MAIL__API_TOKEN", "secret")
	t.Setenv("JSON2MAIL__SMTP_ADDRESS", "smtp.example.com:465")
	t.Setenv("JSON2MAIL__SMTP_FROM", "mailbridge@example.com")
	t.Setenv("JSON2MAIL__SMTP_TLS", "tls")
	t.Setenv("JSON2MAIL__SMTP_RETRY_COUNT", "5")
	t.Setenv("JSON2MAIL__SMTP_RETRY_BACKOFF", "2s")
	t.Setenv("JSON2MAIL__BODY_FORMAT", "html")
	cfg, err := Parse(nil, "test")

	require.NoError(t, err)
	assert.Equal(t, SMTPTLSTLS, cfg.SMTPTLS)
	assert.Equal(t, 5, cfg.SMTPRetryCount)
	assert.Equal(t, 2*time.Second, cfg.SMTPRetryBackoff)
	assert.Equal(t, application.BodyFormatHTML, cfg.BodyFormat)
}

func TestSMTPAuthMustBeComplete(t *testing.T) {
	t.Parallel()

	_, err := Parse([]string{
		"--api-token", "secret",
		"--smtp-address", "smtp.example.com:587",
		"--smtp-from", "mailbridge@example.com",
		"--smtp-username", "user",
	}, "test")

	require.Error(t, err)
}

func TestSMTPFromMustBeValid(t *testing.T) {
	t.Parallel()

	_, err := Parse([]string{
		"--api-token", "secret",
		"--smtp-address", "smtp.example.com:587",
		"--smtp-from", "not-an-email",
	}, "test")

	require.Error(t, err)
}

func TestRetryConfigurationMustNotBeNegative(t *testing.T) {
	t.Parallel()

	_, err := Parse([]string{
		"--api-token", "secret",
		"--smtp-address", "smtp.example.com:587",
		"--smtp-from", "mailbridge@example.com",
		"--smtp-retry-count", "-1",
	}, "test")

	require.Error(t, err)
}

func TestBodyFormatMustBeSupported(t *testing.T) {
	t.Parallel()

	_, err := Parse([]string{
		"--api-token", "secret",
		"--smtp-address", "smtp.example.com:587",
		"--smtp-from", "mailbridge@example.com",
		"--body-format", "markdown",
	}, "test")

	require.Error(t, err)
}
