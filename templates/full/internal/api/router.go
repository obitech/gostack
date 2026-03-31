// Package api provides the HTTP API layer using the Huma framework.
package api //nolint:revive // Standard convention for HTTP handler packages

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"gopkg.in/yaml.v3"

	"github.com/obitech/gostack/log"
	"github.com/obitech/gostack/metrics"
)

// NewRouter creates and configures the HTTP router with all routes registered.
// CorsOrigins is a list of allowed CORS origins; if empty, CORS middleware is not applied.
// RequestTimeout is the per-request deadline; if zero, no timeout is applied.
func NewRouter(
	logger *slog.Logger,
	h *Handler,
	frontendHandler http.Handler,
	httpMetrics *metrics.HTTPMetrics,
	corsOrigins []string,
	requestTimeout time.Duration,
) *chi.Mux {
	r := chi.NewRouter()

	// RequestID must come first so all subsequent middleware can read it.
	r.Use(middleware.RequestID)
	r.Use(log.SlogMiddleware(logger))
	r.Use(middleware.Recoverer)
	if len(corsOrigins) > 0 {
		r.Use(cors.New(cors.Options{
			AllowedOrigins:   corsOrigins,
			AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
			AllowCredentials: true,
			MaxAge:           300,
		}).Handler)
	}
	if requestTimeout > 0 {
		r.Use(middleware.Timeout(requestTimeout))
	}
	if httpMetrics != nil {
		r.Use(httpMetrics.Middleware())
	}

	// Huma API
	config := newAPIConfig()
	api := humachi.New(r, config)

	// Register routes (pass nil handler for OpenAPI generation)
	RegisterRoutes(api, h)

	// Frontend handler (catch-all)
	if frontendHandler != nil {
		r.Handle("/*", frontendHandler)
	}

	return r
}

func newAPIConfig() huma.Config {
	return huma.DefaultConfig("App API", "1.0.0")
}

// GenerateOpenAPIYAML generates the OpenAPI specification as YAML.
func GenerateOpenAPIYAML() ([]byte, error) {
	r := chi.NewRouter()
	config := newAPIConfig()
	api := humachi.New(r, config)

	// Register routes with nil handler to extract type info only.
	RegisterRoutes(api, nil)

	data, err := yaml.Marshal(api.OpenAPI())
	if err != nil {
		return nil, fmt.Errorf("marshaling OpenAPI spec to YAML: %w", err)
	}
	return data, nil
}

// RegisterRoutes registers all API routes.
func RegisterRoutes(api huma.API, h *Handler) {
	// Liveness probe — always ok as long as the process is alive.
	huma.Register(api, huma.Operation{
		OperationID: "liveness",
		Method:      http.MethodGet,
		Path:        "/api/v1/healthz",
		Summary:     "Liveness probe",
		Tags:        []string{"Health"},
	}, func(ctx context.Context, input *struct{}) (*HealthOutput, error) {
		if h == nil {
			return &HealthOutput{Body: HealthResponse{Status: "ok"}}, nil
		}
		return h.Liveness(ctx, input)
	})

	// Readiness probe — checks that the service can handle traffic (database reachable).
	huma.Register(api, huma.Operation{
		OperationID: "readiness",
		Method:      http.MethodGet,
		Path:        "/api/v1/readyz",
		Summary:     "Readiness probe",
		Tags:        []string{"Health"},
	}, func(ctx context.Context, input *struct{}) (*HealthOutput, error) {
		if h == nil {
			return &HealthOutput{Body: HealthResponse{Status: "ok"}}, nil
		}
		return h.Health(ctx, input)
	})

	// Health check — alias for readiness, kept for backwards compatibility.
	huma.Register(api, huma.Operation{
		OperationID: "health",
		Method:      http.MethodGet,
		Path:        "/api/v1/health",
		Summary:     "Health check",
		Tags:        []string{"Health"},
	}, func(ctx context.Context, input *struct{}) (*HealthOutput, error) {
		if h == nil {
			return &HealthOutput{Body: HealthResponse{Status: "ok"}}, nil
		}
		return h.Health(ctx, input)
	})

	// Add your routes here. Example:
	// if h != nil {
	//     huma.Register(api, huma.Operation{
	//         OperationID: "listItems",
	//         Method:      http.MethodGet,
	//         Path:        "/api/v1/items",
	//         Summary:     "List all items",
	//         Tags:        []string{"Items"},
	//     }, h.ListItems)
	// }
}
