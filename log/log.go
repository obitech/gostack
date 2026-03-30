// Package log provides convenience functions for creating structured JSON loggers.
package log

import (
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// NewJSON creates a JSON slog.Logger writing to stdout at the specified level.
func NewJSON(level slog.Level) *slog.Logger {
	return NewJSONWriter(os.Stdout, level)
}

// NewJSONWriter creates a JSON slog.Logger writing to w at the specified level.
func NewJSONWriter(w io.Writer, level slog.Level) *slog.Logger {
	return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level: level,
	}))
}

// SlogMiddleware returns an HTTP middleware that logs each request using the
// provided slog.Logger. It logs the method, path, status code, duration, and
// request ID (if set by chi's RequestID middleware).
func SlogMiddleware(logger *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			defer func() {
				logger.Info("request",
					"method", r.Method,
					"path", r.URL.Path,
					"status", ww.Status(),
					"duration_ms", time.Since(start).Milliseconds(),
					"request_id", middleware.GetReqID(r.Context()),
				)
			}()

			next.ServeHTTP(ww, r)
		})
	}
}
