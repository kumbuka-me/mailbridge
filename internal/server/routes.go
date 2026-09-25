package server

import (
	"net/http"

	"github.com/containeroo/mailbridge/internal/handler"
	"github.com/containeroo/mailbridge/internal/middleware"
)

// addRoutes registers mailbridge's complete HTTP surface.
func addRoutes(mux *http.ServeMux, config Config) {
	mux.Handle("GET /metrics", config.Metrics.Metrics())
	mux.Handle("GET /healthz", handler.Health())
	mux.Handle("POST /healthz", handler.Health())
	mux.Handle("GET /readyz", handler.Readyz())
	mux.Handle("POST /readyz", handler.Readyz())
	mux.Handle("GET /version", handler.Version(config.Version))

	mail := middleware.Chain(
		handler.Mail(config.Forwarder, config.Logger),
		middleware.InstrumentMail(config.Metrics),
		middleware.BearerToken(config.APIToken),
		middleware.RateLimit(config.RateLimit),
	)
	mux.Handle("POST /mail", mail)
}
