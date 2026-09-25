// Package flags parses deployment-level configuration for mailbridge.
package flags

import (
	"errors"
	"fmt"
	"net"
	"net/mail"
	"strings"
	"time"

	"github.com/containeroo/mailbridge/internal/application"
	"github.com/containeroo/mailbridge/internal/logging"
	"github.com/containeroo/tinyflags"
)

// SMTPTLSMode is the stable mailbridge CLI representation of SMTP TLS behavior.
type SMTPTLSMode string

const (
	// SMTPTLSStartTLS requires STARTTLS, usually on port 587.
	SMTPTLSStartTLS SMTPTLSMode = "starttls"
	// SMTPTLSTLS uses implicit TLS, usually on port 465.
	SMTPTLSTLS SMTPTLSMode = "tls"
	// SMTPTLSNone disables SMTP transport encryption.
	SMTPTLSNone SMTPTLSMode = "none"
)

// Config contains deployment-level runtime configuration.
type Config struct {
	// ListenAddress is the TCP address used by the HTTP server.
	ListenAddress string
	// APIToken authenticates incoming mail requests.
	APIToken string
	// SMTPAddress is the SMTP server in HOST:PORT form.
	SMTPAddress string
	// SMTPFrom is the envelope and message sender mailbox.
	SMTPFrom string
	// SMTPUsername optionally enables SMTP authentication.
	SMTPUsername string
	// SMTPPassword is the password paired with SMTPUsername.
	SMTPPassword string
	// SMTPTLS selects STARTTLS, implicit TLS, or no TLS.
	SMTPTLS SMTPTLSMode
	// SMTPInsecureSkipVerify disables SMTP certificate verification.
	SMTPInsecureSkipVerify bool
	// SMTPTimeout bounds each SMTP delivery attempt.
	SMTPTimeout time.Duration
	// SMTPRetryCount is the number of retries after the initial SMTP attempt.
	SMTPRetryCount int
	// SMTPRetryBackoff is the delay before the first SMTP retry.
	SMTPRetryBackoff time.Duration
	// SMTPRetryMaxBackoff caps local exponential SMTP retry backoff.
	SMTPRetryMaxBackoff time.Duration
	// SMTPRetryJitter enables full jitter for SMTP retry delays.
	SMTPRetryJitter bool
	// BodyFormat is the default email body format when a request does not override it.
	BodyFormat application.BodyFormat
	// LogFormat selects structured text or JSON logging.
	LogFormat logging.LogFormat
	// Debug enables verbose diagnostic logging.
	Debug bool
	// AccessLog enables HTTP request logging.
	AccessLog bool
	// RateLimit is the maximum number of mail requests accepted per second; zero disables limiting.
	RateLimit int
	// Overrides records values explicitly overridden through flags or environment variables.
	Overrides map[string]any
}

// Parse parses command-line arguments into application configuration.
func Parse(args []string, version string) (Config, error) {
	cfg := Config{}
	tf := tinyflags.NewFlagSet("mailbridge", tinyflags.ContinueOnError)
	tf.EnvPrefix("MAILBRIDGE_")
	tf.Version(version)

	listen := tf.TCPAddr(
		"listen-address",
		&net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 8080},
		"Address on which the HTTP server listens",
	).
		Short("a").
		Placeholder("ADDR").
		Value()

	tf.StringVar(&cfg.APIToken, "api-token", "", "Bearer token required for incoming mail requests").
		Required().
		Placeholder("TOKEN").
		OverriddenValueMaskFn(tinyflags.MaskFirstLast).
		Value()
	tf.StringVar(&cfg.SMTPAddress, "smtp-address", "", "SMTP server address in HOST:PORT form").
		Required().
		Validate(func(s string) error {
			if _, _, err := net.SplitHostPort(s); err != nil {
				return errors.New("smtp address must use HOST:PORT form")
			}

			from := strings.TrimSpace(cfg.SMTPFrom)
			address, err := mail.ParseAddress(from)
			if err != nil || address.Address == "" {
				return errors.New("smtp from address is invalid")
			}
			return nil
		}).
		Finalize(func(s string) string {
			return strings.TrimSpace(s)
		}).
		Placeholder("HOST:PORT").
		Value()
	tf.StringVar(&cfg.SMTPFrom, "smtp-from", "", "Sender email address").
		Required().
		Placeholder("EMAIL").
		Value()
	tf.StringVar(&cfg.SMTPUsername, "smtp-username", "", "SMTP authentication username").Value()
	tf.StringVar(&cfg.SMTPPassword, "smtp-password", "", "SMTP authentication password").
		OverriddenValueMaskFn(tinyflags.MaskFirstLast).
		Requires("smtp-username").
		Value()
	tlsMode := tinyflags.Enum(
		tf,
		"smtp-tls",
		SMTPTLSStartTLS,
		"SMTP TLS mode",
		SMTPTLSStartTLS,
		SMTPTLSTLS,
		SMTPTLSNone,
	).
		Placeholder("MODE").
		Value()
	tf.BoolVar(
		&cfg.SMTPInsecureSkipVerify,
		"smtp-insecure-skip-verify",
		false,
		"Disable SMTP TLS certificate verification",
	).Value()
	tf.DurationVar(&cfg.SMTPTimeout, "smtp-timeout", 30*time.Second, "Timeout for each SMTP delivery attempt").
		Validate(positiveDuration("smtp timeout")).
		Value()
	tf.IntVar(&cfg.SMTPRetryCount, "smtp-retry-count", 3, "Number of SMTP retries after the initial attempt").
		Validate(nonNegativeInt("smtp retry count")).
		Value()
	tf.DurationVar(&cfg.SMTPRetryBackoff, "smtp-retry-backoff", time.Second, "Delay before the first SMTP retry").
		Validate(nonNegativeDuration("smtp retry backoff")).
		Value()
	tf.DurationVar(&cfg.SMTPRetryMaxBackoff, "smtp-retry-max-backoff", 30*time.Second, "Maximum local SMTP retry backoff").
		Validate(nonNegativeDuration("smtp retry max backoff")).
		Value()
	tf.BoolVar(&cfg.SMTPRetryJitter, "smtp-retry-jitter", true, "Apply full jitter to SMTP retry delays").Value()
	bodyFormat := tinyflags.Enum(
		tf,
		"body-format",
		application.BodyFormatText,
		"Default email body format",
		application.BodyFormatText,
		application.BodyFormatHTML,
	).
		Placeholder("FORMAT").
		Value()

	logFormat := tinyflags.Enum(
		tf,
		"log-format",
		logging.LogFormatJSON,
		"Log output format",
		logging.LogFormatText,
		logging.LogFormatJSON,
	).
		Short("l").
		Placeholder("FORMAT").
		Value()
	tf.BoolVar(&cfg.Debug, "debug", false, "Enable verbose diagnostic logging").Short("d").Value()
	tf.BoolVar(&cfg.AccessLog, "access-log", false, "Enable HTTP request access logging").Value()
	tf.IntVar(&cfg.RateLimit, "rate-limit", 0, "Maximum mail requests per second; zero disables rate limiting").
		Validate(nonNegativeInt("rate limit")).
		Value()

	if err := tf.Parse(args); err != nil {
		return Config{}, err
	}

	cfg.ListenAddress = (*listen).String()
	cfg.SMTPTLS = *tlsMode
	cfg.BodyFormat = *bodyFormat
	cfg.LogFormat = *logFormat
	cfg.Overrides = tf.OverriddenValues()

	return cfg, nil
}

// positiveDuration validates a duration that must be greater than zero.
func positiveDuration(name string) func(time.Duration) error {
	return func(value time.Duration) error {
		if value <= 0 {
			return fmt.Errorf("%s must be greater than zero", name)
		}
		return nil
	}
}

// nonNegativeDuration validates a duration that may be zero but not negative.
func nonNegativeDuration(name string) func(time.Duration) error {
	return func(value time.Duration) error {
		if value < 0 {
			return fmt.Errorf("%s must not be negative", name)
		}
		return nil
	}
}

// nonNegativeInt validates an integer that may be zero but not negative.
func nonNegativeInt(name string) func(int) error {
	return func(value int) error {
		if value < 0 {
			return fmt.Errorf("%s must not be negative", name)
		}
		return nil
	}
}
