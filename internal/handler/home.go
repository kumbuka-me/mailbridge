package handler

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"strings"
)

const homeTemplatePath = "index.gohtml"

// HomeConfig contains the non-sensitive runtime settings shown on the home page.
type HomeConfig struct {
	// Version is the running build version.
	Version string
	// Commit is the running build commit.
	Commit string
	// Assets contains the embedded web application assets served by HTTP endpoints.
	Assets fs.FS
	// BodyFormat is the default email body format.
	BodyFormat string
	// SMTPTLS is the configured SMTP TLS mode.
	SMTPTLS string
	// SMTPRetryCount is the number of retries after the initial SMTP attempt.
	SMTPRetryCount int
	// RateLimit is the maximum number of mail requests accepted per second.
	RateLimit int
	// AccessLog reports whether HTTP access logging is enabled.
	AccessLog bool
}

// homeView contains presentation-ready values for the embedded home page.
type homeView struct {
	HomeConfig
	RateLimitStatus string
	AccessLogStatus string
}

// Home returns mailbridge's small human-readable API and runtime information page.
func Home(config HomeConfig) http.HandlerFunc {
	body, err := renderHome(config)
	if err != nil {
		return homeRenderError()
	}

	cspDirectives := []string{
		"default-src 'none'",
		"img-src 'self'",
		"script-src 'unsafe-inline'",
		"style-src 'unsafe-inline'",
		"base-uri 'none'",
		"form-action 'none'",
		"frame-ancestors 'none'",
	}
	cspValue := strings.Join(cspDirectives, "; ")

	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Security-Policy", cspValue)
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
}

// renderHome renders the embedded home page once while the HTTP surface is constructed.
func renderHome(config HomeConfig) ([]byte, error) {
	if config.Assets == nil {
		return nil, errors.New("web assets are not configured")
	}

	source, err := fs.ReadFile(config.Assets, homeTemplatePath)
	if err != nil {
		return nil, fmt.Errorf("read home template: %w", err)
	}

	homeTemplate, err := template.New("index").Parse(string(source))
	if err != nil {
		return nil, fmt.Errorf("parse home template: %w", err)
	}

	view := homeView{
		HomeConfig:      config,
		RateLimitStatus: rateLimitStatus(config.RateLimit),
		AccessLogStatus: enabledStatus(config.AccessLog),
	}

	var rendered bytes.Buffer
	if err := homeTemplate.Execute(&rendered, view); err != nil {
		return nil, fmt.Errorf("render home template: %w", err)
	}

	return rendered.Bytes(), nil
}

// homeRenderError returns a stable internal-server-error handler without exposing template details.
func homeRenderError() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "Render home page failed.", http.StatusInternalServerError)
	}
}

// rateLimitStatus returns a human-readable rate-limit setting.
func rateLimitStatus(rateLimit int) string {
	if rateLimit <= 0 {
		return "disabled"
	}
	if rateLimit == 1 {
		return "1 request/s"
	}
	return fmt.Sprintf("%d requests/s", rateLimit)
}

// enabledStatus returns enabled or disabled for a boolean setting.
func enabledStatus(enabled bool) string {
	if enabled {
		return "enabled"
	}
	return "disabled"
}
