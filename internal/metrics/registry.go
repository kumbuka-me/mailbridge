// Package metrics exposes mailbridge's built-in Prometheus instrumentation.
package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Registry owns the private Prometheus registry and application metrics.
type Registry struct {
	// registry contains only collectors explicitly registered by mailbridge.
	registry *prometheus.Registry
	// mailRequests counts completed POST /mail requests by response code.
	mailRequests *prometheus.CounterVec
	// mailRequestDuration observes complete POST /mail request duration.
	mailRequestDuration prometheus.Histogram
	// emailDeliveries counts completed SMTP delivery operations.
	emailDeliveries prometheus.Counter
	// emailDeliveryErrors counts failed SMTP delivery operations.
	emailDeliveryErrors prometheus.Counter
	// emailDeliveryDuration observes complete SMTP delivery operation duration.
	emailDeliveryDuration prometheus.Histogram
}

// NewRegistry constructs mailbridge's private Prometheus registry.
func NewRegistry(version, commit string) *Registry {
	registry := prometheus.NewRegistry()

	mailRequests := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mailbridge_mail_requests_total",
			Help: "Total number of completed POST /mail requests by status code.",
		},
		[]string{"code"},
	)
	mailRequestDuration := prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "mailbridge_mail_request_duration_seconds",
		Help:    "POST /mail request duration in seconds.",
		Buckets: prometheus.DefBuckets,
	})
	emailDeliveries := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "mailbridge_email_deliveries_total",
		Help: "Total number of completed email delivery operations.",
	})
	emailDeliveryErrors := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "mailbridge_email_delivery_errors_total",
		Help: "Total number of failed email delivery operations.",
	})
	emailDeliveryDuration := prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "mailbridge_email_delivery_duration_seconds",
		Help:    "Email delivery operation duration in seconds.",
		Buckets: prometheus.DefBuckets,
	})
	buildInfo := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "mailbridge_build_info",
		Help: "mailbridge build information.",
		ConstLabels: prometheus.Labels{
			"version": version,
			"commit":  commit,
		},
	})
	buildInfo.Set(1)

	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		mailRequests,
		mailRequestDuration,
		emailDeliveries,
		emailDeliveryErrors,
		emailDeliveryDuration,
		buildInfo,
	)

	return &Registry{
		registry:              registry,
		mailRequests:          mailRequests,
		mailRequestDuration:   mailRequestDuration,
		emailDeliveries:       emailDeliveries,
		emailDeliveryErrors:   emailDeliveryErrors,
		emailDeliveryDuration: emailDeliveryDuration,
	}
}

// Metrics returns the Prometheus exposition handler for this registry.
func (r *Registry) Metrics() http.Handler {
	return promhttp.HandlerFor(r.registry, promhttp.HandlerOpts{})
}

// ObserveMailRequest records one completed POST /mail request.
func (r *Registry) ObserveMailRequest(status int, duration time.Duration) {
	r.mailRequests.WithLabelValues(strconv.Itoa(status)).Inc()
	r.mailRequestDuration.Observe(duration.Seconds())
}

// ObserveEmailDelivery records one completed email delivery operation.
func (r *Registry) ObserveEmailDelivery(duration time.Duration, err error) {
	r.emailDeliveries.Inc()
	r.emailDeliveryDuration.Observe(duration.Seconds())
	if err != nil {
		r.emailDeliveryErrors.Inc()
	}
}
