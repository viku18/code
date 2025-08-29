package telemetry

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelmetric "go.opentelemetry.io/otel/metric"
)

type OtelMetrics struct {
	meter otelmetric.Meter

	requestCounter  otelmetric.Int64Counter
	requestDuration otelmetric.Float64Histogram
	errorCounter    otelmetric.Int64Counter
}

var otelMetrics *OtelMetrics

func InitOtelMetrics() (*OtelMetrics, error) {
	// Get the global meter provider
	meter := otel.GetMeterProvider().Meter("github.com/acai-travel/tech-challenge/server")

	// Create metrics instruments using current API
	requestCounter, err := meter.Int64Counter(
		"http_requests_total",
		otelmetric.WithDescription("Total number of HTTP requests"),
	)
	if err != nil {
		return nil, err
	}

	requestDuration, err := meter.Float64Histogram(
		"http_request_duration_seconds",
		otelmetric.WithDescription("HTTP request duration in seconds"),
		otelmetric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	errorCounter, err := meter.Int64Counter(
		"http_errors_total",
		otelmetric.WithDescription("Total number of HTTP errors"),
	)
	if err != nil {
		return nil, err
	}

	otelMetrics = &OtelMetrics{
		meter:           meter,
		requestCounter:  requestCounter,
		requestDuration: requestDuration,
		errorCounter:    errorCounter,
	}

	return otelMetrics, nil
}

func RecordOtelRequest(ctx context.Context, method, path string, statusCode int, duration time.Duration) {
	if otelMetrics == nil {
		return
	}

	attrs := otelmetric.WithAttributes(
		attribute.String("method", method),
		attribute.String("path", path),
		attribute.Int("status", statusCode),
	)

	otelMetrics.requestCounter.Add(ctx, 1, attrs)
	otelMetrics.requestDuration.Record(ctx, duration.Seconds(), attrs)

	if statusCode >= 400 {
		otelMetrics.errorCounter.Add(ctx, 1, attrs)
	}
}