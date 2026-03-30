package metrics_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/obitech/gostack/metrics"
)

func newTestHTTPMetrics(t *testing.T) *metrics.HTTPMetrics {
	t.Helper()
	m, err := metrics.NewHTTPMetrics(metrics.HTTPMetricsConfig{
		MeterName:    "test/http",
		ExcludePaths: []string{"/internal/"},
	})
	require.NoError(t, err)
	return m
}

func TestNewHTTPMetrics(t *testing.T) {
	m := newTestHTTPMetrics(t)
	assert.NotNil(t, m)
}

func TestHTTPMetrics_Middleware_RequestPassthrough(t *testing.T) {
	m := newTestHTTPMetrics(t)

	r := chi.NewRouter()
	r.Use(m.Middleware())
	r.Get("/hello", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/hello", nil))

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHTTPMetrics_Middleware_ExcludePaths(t *testing.T) {
	m := newTestHTTPMetrics(t)

	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	handler := m.Middleware()(inner)

	// Excluded path — next should still be called, just no metrics recorded.
	req := httptest.NewRequest(http.MethodGet, "/internal/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.True(t, called, "next handler must be called even for excluded paths")
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHTTPMetrics_Middleware_NilRouteContext(t *testing.T) {
	m := newTestHTTPMetrics(t)

	// Use a plain http.Handler (no chi router) — chi.RouteContext returns nil.
	// This must not panic after the nil guard fix.
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := m.Middleware()(inner)

	req := httptest.NewRequest(http.MethodGet, "/some/path", nil)
	rec := httptest.NewRecorder()

	assert.NotPanics(t, func() {
		handler.ServeHTTP(rec, req)
	})
	assert.Equal(t, http.StatusOK, rec.Code)
}
