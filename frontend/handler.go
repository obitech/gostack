// Package frontend provides HTTP handlers for serving embedded SPA frontends
// with support for development server proxying.
package frontend

import (
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

// Config holds configuration for the frontend handler.
type Config struct {
	// DevServerURL is the URL of the Vite dev server (e.g., "http://localhost:5173").
	// If set, requests will be proxied to this URL instead of serving embedded assets.
	DevServerURL string
	// Assets is the embedded filesystem containing the frontend build.
	Assets fs.FS
	// Subdir is the subdirectory within Assets containing the build output.
	// Defaults to "dist" if empty.
	Subdir string
}

// NewHandler creates a handler that serves the frontend.
// If config.DevServerURL is set, it proxies to the dev server.
// Otherwise, it serves files from the embedded config.Assets filesystem.
func NewHandler(cfg Config) (http.Handler, error) {
	if cfg.DevServerURL != "" {
		return newDevProxy(cfg.DevServerURL)
	}
	return newStaticHandler(cfg.Assets, cfg.Subdir)
}

func newDevProxy(devServerURL string) (http.Handler, error) {
	target, err := url.Parse(devServerURL)
	if err != nil {
		return nil, fmt.Errorf("parsing dev server URL: %w", err)
	}

	proxy := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(target)
			r.Out.Host = target.Host
		},
	}

	return proxy, nil
}

func newStaticHandler(assets fs.FS, subdir string) (http.Handler, error) {
	if subdir == "" {
		subdir = "dist"
	}

	// Sub into the build directory since embed.FS includes the directory prefix
	distFS, err := fs.Sub(assets, subdir)
	if err != nil {
		return nil, fmt.Errorf("creating sub filesystem for %s: %w", subdir, err)
	}

	fileServer := http.FileServer(http.FS(distFS))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}

		// Try to open the file to check if it exists
		f, err := distFS.Open(path)
		if err != nil {
			// File not found, serve index.html for SPA routing
			serveIndexHTML(w, distFS)
			return
		}
		_ = f.Close()

		fileServer.ServeHTTP(w, r)
	}), nil
}

func serveIndexHTML(w http.ResponseWriter, assets fs.FS) {
	f, err := assets.Open("index.html")
	if err != nil {
		http.Error(w, "index.html not found", http.StatusNotFound)
		return
	}
	defer func() { _ = f.Close() }()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.Copy(w, f)
}
