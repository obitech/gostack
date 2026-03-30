package metrics

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// HTTPMetrics holds the OpenTelemetry instruments for HTTP request metrics.
type HTTPMetrics struct {
	requestDuration metric.Float64Histogram
	requestSize     metric.Int64Histogram
	responseSize    metric.Int64Histogram
	excludePaths    []string
}

// HTTPMetricsConfig configures the HTTP metrics middleware.
type HTTPMetricsConfig struct {
	// MeterName is the name used to create the meter (e.g., "myapp/http").
	MeterName string
	// ExcludePaths is a list of path prefixes to exclude from metrics collection
	// (e.g., []string{"/internal/", "/health"}).
	ExcludePaths []string
}

// NewHTTPMetrics creates and registers HTTP metrics instruments.
// Must be called after metrics.Setup() has initialized the global MeterProvider.
func NewHTTPMetrics(cfg HTTPMetricsConfig) (*HTTPMetrics, error) {
	meter := otel.Meter(cfg.MeterName)

	requestDuration, err := meter.Float64Histogram(
		"http_request_duration_seconds",
		metric.WithDescription("Duration of HTTP requests in seconds"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10),
	)
	if err != nil {
		return nil, fmt.Errorf("creating request duration histogram: %w", err)
	}

	requestSize, err := meter.Int64Histogram(
		"http_request_size_bytes",
		metric.WithDescription("Size of HTTP request bodies in bytes"),
		metric.WithUnit("By"),
		metric.WithExplicitBucketBoundaries(100, 1000, 10000, 100000, 1000000, 10000000),
	)
	if err != nil {
		return nil, fmt.Errorf("creating request size histogram: %w", err)
	}

	responseSize, err := meter.Int64Histogram(
		"http_response_size_bytes",
		metric.WithDescription("Size of HTTP response bodies in bytes"),
		metric.WithUnit("By"),
		metric.WithExplicitBucketBoundaries(100, 1000, 10000, 100000, 1000000, 10000000),
	)
	if err != nil {
		return nil, fmt.Errorf("creating response size histogram: %w", err)
	}

	return &HTTPMetrics{
		requestDuration: requestDuration,
		requestSize:     requestSize,
		responseSize:    responseSize,
		excludePaths:    cfg.ExcludePaths,
	}, nil
}

// Middleware returns an HTTP middleware that records request metrics.
// Requests to paths matching ExcludePaths prefixes are excluded from metrics collection.
// Note: This middleware uses chi.RouteContext for route patterns - it works with chi router.
func (m *HTTPMetrics) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip metrics for excluded paths
			for _, prefix := range m.excludePaths {
				if strings.HasPrefix(r.URL.Path, prefix) {
					next.ServeHTTP(w, r)
					return
				}
			}

			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			// Get request size from Content-Length header
			// For chunked encoding this will be -1, treat as 0
			requestBytes := max(r.ContentLength, 0)

			defer func() {
				duration := time.Since(start).Seconds()

				// Get route pattern (populated by chi after routing).
				// Guard against nil in case the middleware is used outside of chi.
				routePattern := "unknown"
				if rc := chi.RouteContext(r.Context()); rc != nil {
					if rp := rc.RoutePattern(); rp != "" {
						routePattern = rp
					}
				}

				attrs := metric.WithAttributes(
					attribute.String("route", routePattern),
					attribute.String("method", r.Method),
					attribute.String("status", strconv.Itoa(ww.Status())),
				)

				m.requestDuration.Record(r.Context(), duration, attrs)
				m.requestSize.Record(r.Context(), requestBytes, attrs)
				m.responseSize.Record(r.Context(), int64(ww.BytesWritten()), attrs)
			}()

			next.ServeHTTP(ww, r)
		})
	}
}
