// Package metrics provides OpenTelemetry instrumentation with Prometheus export.
package metrics

import (
	"context"
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	prometheusExporter "go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/trace/noop"
)

// Setup initializes OpenTelemetry with Prometheus metrics exporter
// and a noop tracer provider (metrics only, no tracing).
// The namespace prefixes all metric names (e.g., "myapp" produces "myapp_http_request_duration_seconds").
// Returns a shutdown function that should be called on application exit.
func Setup(namespace string) (shutdown func(context.Context) error, err error) {
	// Disable tracing - use noop tracer provider
	otel.SetTracerProvider(noop.NewTracerProvider())

	// Create Prometheus exporter for metrics
	exporter, err := prometheusExporter.New(
		prometheusExporter.WithNamespace(namespace),
	)
	if err != nil {
		return nil, fmt.Errorf("creating prometheus exporter: %w", err)
	}

	// Create and set MeterProvider
	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(exporter),
	)
	otel.SetMeterProvider(meterProvider)

	return meterProvider.Shutdown, nil
}

// Handler returns the Prometheus HTTP handler for the /metrics endpoint.
func Handler() http.Handler {
	return promhttp.Handler()
}
