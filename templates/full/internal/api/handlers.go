package api //nolint:revive // Standard convention for HTTP handler packages

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Handler contains all HTTP handlers for the API.
type Handler struct {
	logger *slog.Logger
	pool   *pgxpool.Pool
}

// NewHandler creates a new Handler with the given dependencies.
func NewHandler(logger *slog.Logger, pool *pgxpool.Pool) *Handler {
	return &Handler{logger: logger, pool: pool}
}

// Health performs a deep health check by pinging the database.
// It is used for the /health and /readyz endpoints.
func (h *Handler) Health(ctx context.Context, _ *struct{}) (*HealthOutput, error) {
	if err := h.pool.Ping(ctx); err != nil {
		return nil, huma.NewError(http.StatusServiceUnavailable, "database unavailable", err)
	}
	return &HealthOutput{Body: HealthResponse{Status: "ok"}}, nil
}

// Liveness is a lightweight liveness probe that always returns ok
// as long as the process is running.
func (h *Handler) Liveness(_ context.Context, _ *struct{}) (*HealthOutput, error) {
	return &HealthOutput{Body: HealthResponse{Status: "ok"}}, nil
}
