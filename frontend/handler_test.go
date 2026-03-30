package frontend_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/obitech/gostack/frontend"
)

// testFS builds a minimal FS that mimics an embedded frontend build.
func testFS() fstest.MapFS {
	return fstest.MapFS{
		"dist/index.html": {Data: []byte(`<html>app</html>`)},
		"dist/app.js":     {Data: []byte(`console.log("app")`)},
	}
}

func TestNewHandler_ServesStaticFile(t *testing.T) {
	h, err := frontend.NewHandler(frontend.Config{
		Assets: testFS(),
		Subdir: "dist",
	})
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))

	assert.Equal(t, http.StatusOK, rec.Code)
	body, _ := io.ReadAll(rec.Body)
	assert.Contains(t, string(body), `console.log("app")`)
}

func TestNewHandler_ServesIndexHTML(t *testing.T) {
	h, err := frontend.NewHandler(frontend.Config{
		Assets: testFS(),
		Subdir: "dist",
	})
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.Equal(t, http.StatusOK, rec.Code)
	body, _ := io.ReadAll(rec.Body)
	assert.Contains(t, string(body), "<html>app</html>")
}

func TestNewHandler_SPAFallback(t *testing.T) {
	h, err := frontend.NewHandler(frontend.Config{
		Assets: testFS(),
		Subdir: "dist",
	})
	require.NoError(t, err)

	// A path that doesn't exist in the FS should fall back to index.html.
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/some/deep/route", nil))

	assert.Equal(t, http.StatusOK, rec.Code)
	body, _ := io.ReadAll(rec.Body)
	assert.Contains(t, string(body), "<html>app</html>", "unknown paths must serve index.html for SPA routing")
}

func TestNewHandler_DefaultsSubdirToDist(t *testing.T) {
	h, err := frontend.NewHandler(frontend.Config{
		Assets: testFS(),
		// Subdir intentionally omitted — should default to "dist".
	})
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestNewHandler_DevServerProxy(t *testing.T) {
	// Spin up a test backend that represents the Vite dev server.
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("vite response"))
	}))
	defer backend.Close()

	h, err := frontend.NewHandler(frontend.Config{
		DevServerURL: backend.URL,
	})
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.Equal(t, http.StatusOK, rec.Code)
	body, _ := io.ReadAll(rec.Body)
	assert.Equal(t, "vite response", string(body))
}

func TestNewHandler_InvalidDevServerURL(t *testing.T) {
	_, err := frontend.NewHandler(frontend.Config{
		DevServerURL: "://invalid-url",
	})
	assert.Error(t, err)
}
