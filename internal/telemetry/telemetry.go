package telemetry

import (
	"context"
	"net/http"
	"time"

	//"golang.org/x/exp/slog"
)

func Init() {
	//slog.Info("Metrics initialized with Prometheus")
}

func RecordRequest(ctx context.Context, method, path string, statusCode int, duration time.Duration) {
	// Use Prometheus for metrics recording
	RecordPrometheusRequest(method, path, statusCode, duration.Seconds())
}

func GetMetricsHandler() http.Handler {
	return GetPrometheusHandler()
}