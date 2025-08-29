package telemetry

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	requestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests",
	}, []string{"method", "path", "status"})

	requestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request duration in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path", "status"})

	errorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_errors_total",
		Help: "Total number of HTTP errors",
	}, []string{"method", "path", "status"})
)

func RecordPrometheusRequest(method, path string, statusCode int, duration float64) {
	labels := prometheus.Labels{
		"method": method,
		"path":   path,
		"status": http.StatusText(statusCode),
	}

	requestsTotal.With(labels).Inc()
	requestDuration.With(labels).Observe(duration)

	if statusCode >= 400 {
		errorsTotal.With(labels).Inc()
	}
}

func GetPrometheusHandler() http.Handler {
	return promhttp.Handler()
}