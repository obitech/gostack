package api_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/example/app/internal/api"
)

// newTestRouter creates a minimal router for testing without a database or frontend.
// The Handler is created with a nil pool, which is valid for liveness checks.
func newTestRouter(t *testing.T) http.Handler {
	t.Helper()
	h := api.NewHandler(slog.Default(), nil)
	return api.NewRouter(slog.Default(), h, nil, nil, nil, 0)
}

func TestRouter_LivenessProbe(t *testing.T) {
	router := newTestRouter(t)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/healthz", nil))

	require.Equal(t, http.StatusOK, rec.Code)

	body, err := io.ReadAll(rec.Body)
	require.NoError(t, err)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(body, &resp))
	assert.Equal(t, "ok", resp["status"])
}

func TestRouter_ReadyzEndpointRegistered(t *testing.T) {
	router := newTestRouter(t)

	// /readyz is registered; with nil pool it returns 500 (DB unavailable),
	// but it must not return 404 (route must exist).
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/readyz", nil))

	assert.NotEqual(t, http.StatusNotFound, rec.Code, "readyz route must be registered")
}

func TestRouter_CORS_AllowedOrigin(t *testing.T) {
	h := api.NewHandler(slog.Default(), nil)
	router := api.NewRouter(slog.Default(), h, nil, nil, []string{"http://localhost:5173"}, 0)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/healthz", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, "http://localhost:5173", rec.Header().Get("Access-Control-Allow-Origin"))
}

func TestRouter_CORS_BlockedOrigin(t *testing.T) {
	h := api.NewHandler(slog.Default(), nil)
	router := api.NewRouter(slog.Default(), h, nil, nil, []string{"http://localhost:5173"}, 0)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/healthz", nil)
	req.Header.Set("Origin", "http://evil.example.com")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"), "untrusted origin must not receive CORS header")
}

func TestRouter_CORS_DisabledWhenNoOrigins(t *testing.T) {
	// Router with no CORS origins configured.
	router := newTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/healthz", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	// No CORS headers should be present.
	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"))
}
