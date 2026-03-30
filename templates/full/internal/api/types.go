// Package api provides the HTTP API layer using the Huma framework.
package api //nolint:revive // Standard convention for HTTP handler packages

// HealthResponse is the response body for health check.
type HealthResponse struct {
	Status string `json:"status" example:"ok"`
}

// HealthOutput wraps the health response.
type HealthOutput struct {
	Body HealthResponse
}
