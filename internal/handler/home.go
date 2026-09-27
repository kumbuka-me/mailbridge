package handler

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
	"net/http"
)

var (
	//go:embed index.gohtml
	homeTemplateSource string

	homeTemplate = template.Must(template.New("index").Parse(homeTemplateSource))
)

// HomeConfig contains the non-sensitive runtime settings shown on the home page.
type HomeConfig struct {
	// Version is the running build version.
	Version string
	// Commit is the running build commit.
	Commit string
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
	view := homeView{
		HomeConfig:      config,
		RateLimitStatus: rateLimitStatus(config.RateLimit),
		AccessLogStatus: enabledStatus(config.AccessLog),
	}

	var rendered bytes.Buffer
	renderErr := homeTemplate.Execute(&rendered, view)
	body := rendered.Bytes()

	return func(w http.ResponseWriter, _ *http.Request) {
		if renderErr != nil {
			http.Error(w, "Render home page failed.", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
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
