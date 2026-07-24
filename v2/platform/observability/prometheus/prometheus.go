package prometheus

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics records HTTP request observations as Prometheus metrics. It
// implements observability.MetricsRecorder.
type Metrics struct {
	requestDuration *prometheus.HistogramVec
	requestsTotal   *prometheus.CounterVec
	requestSize     *prometheus.SummaryVec

	// labels are appended to every metric to allow callers to
	// distinguish between different servers or services.
	labels prometheus.Labels
}

// NewMetrics creates a Metrics instance with standard OpenMetrics-compatible
// metric names. The provided labels (e.g. {"service": "api"}) are attached to
// every observation.
func NewMetrics(labels prometheus.Labels) *Metrics {
	if labels == nil {
		labels = prometheus.Labels{}
	}

	return &Metrics{
		requestDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:        "http_request_duration_seconds",
			Help:        "Duration of HTTP requests in seconds.",
			ConstLabels: labels,
		}, []string{"method", "path", "code"}),

		requestsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name:        "http_requests_total",
			Help:        "Total number of HTTP requests.",
			ConstLabels: labels,
		}, []string{"method", "path", "code"}),

		requestSize: prometheus.NewSummaryVec(prometheus.SummaryOpts{
			Name:        "http_response_size_bytes",
			Help:        "Size of HTTP response bodies in bytes.",
			ConstLabels: labels,
		}, []string{"method", "path"}),
	}
}

// ObserveHTTPRequest implements observability.MetricsRecorder.
func (m *Metrics) ObserveHTTPRequest(method, path string, code int, duration time.Duration, size int64) {
	if m == nil {
		return
	}

	codeStr := strconv.Itoa(code)

	if m.requestDuration != nil {
		m.requestDuration.WithLabelValues(method, path, codeStr).Observe(duration.Seconds())
	}
	if m.requestsTotal != nil {
		m.requestsTotal.WithLabelValues(method, path, codeStr).Inc()
	}
	if m.requestSize != nil {
		m.requestSize.WithLabelValues(method, path).Observe(float64(size))
	}
}

// Register registers all metrics with reg. Call this once during
// application startup.
func (m *Metrics) Register(reg *prometheus.Registry) {
	if m == nil || reg == nil {
		return
	}
	reg.MustRegister(m.requestDuration)
	reg.MustRegister(m.requestsTotal)
	reg.MustRegister(m.requestSize)
}

// Handler returns an http.Handler that exposes the registered metrics in
// Prometheus text format. Use alongside your router.
func Handler(reg *prometheus.Registry) http.Handler {
	return promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
}
