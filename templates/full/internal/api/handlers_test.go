package api_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/example/app/internal/api"
)

func TestHandler_Liveness(t *testing.T) {
	h := api.NewHandler(slog.Default(), nil)

	output, err := h.Liveness(context.Background(), nil)

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, "ok", output.Body.Status)
}
