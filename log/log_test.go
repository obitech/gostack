package log_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	golog "github.com/obitech/gostack/log"
)

func TestNewJSON(t *testing.T) {
	logger := golog.NewJSON(slog.LevelInfo)
	assert.NotNil(t, logger)
}

func TestNewJSONWriter_OutputFormat(t *testing.T) {
	var buf bytes.Buffer
	logger := golog.NewJSONWriter(&buf, slog.LevelInfo)

	logger.Info("test message", "key", "value")

	var entry map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &entry))
	assert.Equal(t, "test message", entry["msg"])
	assert.Equal(t, "INFO", entry["level"])
	assert.Equal(t, "value", entry["key"])
}

func TestNewJSONWriter_LevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	logger := golog.NewJSONWriter(&buf, slog.LevelWarn)

	logger.Debug("should be filtered")
	logger.Info("should also be filtered")
	logger.Warn("should appear")

	lines := bytes.Split(bytes.TrimRight(buf.Bytes(), "\n"), []byte("\n"))
	assert.Len(t, lines, 1, "only the Warn entry should be written")

	var entry map[string]any
	require.NoError(t, json.Unmarshal(lines[0], &entry))
	assert.Equal(t, "should appear", entry["msg"])
}

func TestSlogMiddleware_LogsExpectedFields(t *testing.T) {
	var buf bytes.Buffer
	logger := golog.NewJSONWriter(&buf, slog.LevelInfo)

	r := chi.NewRouter()
	// RequestID must precede SlogMiddleware so the request_id is available.
	r.Use(middleware.RequestID)
	r.Use(golog.SlogMiddleware(logger))
	r.Get("/test", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var entry map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &entry))
	assert.Equal(t, "GET", entry["method"])
	assert.Equal(t, "/test", entry["path"])
	assert.Equal(t, float64(http.StatusOK), entry["status"])
	assert.NotEmpty(t, entry["request_id"], "request_id must be non-empty when RequestID middleware precedes SlogMiddleware")
	assert.Contains(t, entry, "duration_ms")
}

func TestSlogMiddleware_StatusCodeReflected(t *testing.T) {
	var buf bytes.Buffer
	logger := golog.NewJSONWriter(&buf, slog.LevelInfo)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(golog.SlogMiddleware(logger))
	r.Get("/not-found", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/not-found", nil))

	var entry map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &entry))
	assert.Equal(t, float64(http.StatusNotFound), entry["status"])
}
