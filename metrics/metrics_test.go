package metrics_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/obitech/gostack/metrics"
)

func TestSetup(t *testing.T) {
	shutdown, err := metrics.Setup("gostack_test_setup")
	require.NoError(t, err)
	require.NotNil(t, shutdown)

	t.Cleanup(func() {
		require.NoError(t, shutdown(context.Background()))
	})
}

func TestSetup_ShutdownIdempotent(t *testing.T) {
	shutdown, err := metrics.Setup("gostack_test_shutdown")
	require.NoError(t, err)

	// Shutdown should succeed.
	require.NoError(t, shutdown(context.Background()))
}

func TestHandler_NotNil(t *testing.T) {
	h := metrics.Handler()
	assert.NotNil(t, h)
}

func TestHandler_Serves(t *testing.T) {
	h := metrics.Handler()

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}
