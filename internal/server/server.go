// Package server constructs mailbridge's HTTP surface.
package server

import (
	"log/slog"
	"net/http"

	"github.com/containeroo/mailbridge/internal/handler"
	appmetrics "github.com/containeroo/mailbridge/internal/metrics"
	"github.com/containeroo/mailbridge/internal/middleware"
)

// Config groups the capabilities used to construct HTTP routes.
type Config struct {
	// Version is the running build version.
	Version string
	// Forwarder handles decoded mail requests.
	Forwarder handler.Forwarder
	// APIToken authenticates incoming mail requests.
	APIToken string
	// Logger records request and delivery failures.
	Logger *slog.Logger
	// AccessLog enables HTTP request logging.
	AccessLog bool
	// RateLimit limits authenticated mail requests per second; zero disables limiting.
	RateLimit int
	// Metrics exposes Prometheus metrics and records mail endpoint requests.
	Metrics *appmetrics.Registry
}

// New constructs the fully routed HTTP application.
func New(config Config) http.Handler {
	logger := config.Logger

	mux := http.NewServeMux()
	addRoutes(mux, config)

	middlewares := []middleware.Middleware{middleware.RecoverPanics(logger)}
	if config.AccessLog {
		middlewares = append(middlewares, middleware.AccessLog(logger))
	}

	return middleware.Chain(mux, middlewares...)
}
