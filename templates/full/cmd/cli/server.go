// Package main provides the CLI entry point for the application.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/example/app/internal/api"
	"github.com/example/app/internal/database"
	"github.com/example/app/web"
	"github.com/obitech/gostack/config"
	"github.com/obitech/gostack/frontend"
	golog "github.com/obitech/gostack/log"
	"github.com/obitech/gostack/metrics"
)

// ServerConfig defines all configuration options for the server command.
type ServerConfig struct {
	DatabaseURL     string        `flag:"database-url,d" env:"DATABASE_URL" default:"postgres://app:app@localhost:5432/app?sslmode=disable" desc:"database connection URL"`
	Addr            string        `flag:"addr,a" env:"ADDR" default:":8080" desc:"server listen address"`
	ViteDevServer   string        `flag:"vite-dev-server" env:"VITE_DEV_SERVER" desc:"Vite dev server URL for frontend proxying"`
	ReadTimeout     time.Duration `flag:"server-read-timeout" env:"SERVER_READ_TIMEOUT" default:"15s" desc:"HTTP server read timeout"`
	WriteTimeout    time.Duration `flag:"server-write-timeout" env:"SERVER_WRITE_TIMEOUT" default:"15s" desc:"HTTP server write timeout"`
	IdleTimeout     time.Duration `flag:"server-idle-timeout" env:"SERVER_IDLE_TIMEOUT" default:"60s" desc:"HTTP server idle timeout"`
	ShutdownTimeout time.Duration `flag:"server-shutdown-timeout" env:"SERVER_SHUTDOWN_TIMEOUT" default:"30s" desc:"graceful shutdown timeout"`
	RequestTimeout  time.Duration `flag:"request-timeout" env:"REQUEST_TIMEOUT" default:"30s" desc:"per-request handler timeout"`
	CORSOrigins     string        `flag:"cors-origins" env:"CORS_ORIGINS" default:"http://localhost:5173" desc:"allowed CORS origins, comma-separated (empty to disable)"`
	LogLevel        string        `flag:"log-level,l" env:"LOG_LEVEL" default:"info" desc:"log level (debug, info, warn, error)"`
}

var serverCfg ServerConfig

func init() {
	if err := config.RegisterFlags(serverCmd, &serverCfg); err != nil {
		panic(fmt.Sprintf("registering server flags: %v", err))
	}
}

var serverCmd = &cobra.Command{
	Use:   "run-server",
	Short: "Start the API server",
	Long:  "Starts the HTTP API server with metrics, database connection, and frontend serving.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runServer(cmd)
	},
}

// parseLogLevel converts a string log level to slog.Level.
func parseLogLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// parseCORSOrigins splits a comma-separated origins string into a slice,
// trimming whitespace and skipping empty entries.
func parseCORSOrigins(s string) []string {
	if s == "" {
		return nil
	}
	var origins []string
	for _, part := range strings.Split(s, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			origins = append(origins, trimmed)
		}
	}
	return origins
}

func runServer(cmd *cobra.Command) error {
	if err := config.Load(cmd, &serverCfg); err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Logger
	logger := golog.NewJSON(parseLogLevel(serverCfg.LogLevel))
	slog.SetDefault(logger)

	// Metrics
	shutdown, err := metrics.Setup("app")
	if err != nil {
		return fmt.Errorf("setting up metrics: %w", err)
	}
	defer func() {
		if err := shutdown(context.Background()); err != nil {
			slog.Error("shutting down metrics", "error", err)
		}
	}()

	httpMetrics, err := metrics.NewHTTPMetrics(metrics.HTTPMetricsConfig{
		MeterName:    "app/http",
		ExcludePaths: []string{"/internal/"},
	})
	if err != nil {
		return fmt.Errorf("creating HTTP metrics: %w", err)
	}

	// Database
	db, err := database.New(ctx, serverCfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}
	defer db.Close()

	// Handlers
	h := api.NewHandler(logger, db.Pool())

	// Frontend
	frontendHandler, err := frontend.NewHandler(frontend.Config{
		DevServerURL: serverCfg.ViteDevServer,
		Assets:       web.Dist,
		Subdir:       "dist",
	})
	if err != nil {
		return fmt.Errorf("creating frontend handler: %w", err)
	}

	// Router
	router := api.NewRouter(
		logger,
		h,
		frontendHandler,
		httpMetrics,
		parseCORSOrigins(serverCfg.CORSOrigins),
		serverCfg.RequestTimeout,
	)

	// Metrics endpoint
	router.Handle("/internal/metrics", metrics.Handler())

	// Server
	srv := &http.Server{
		Addr:         serverCfg.Addr,
		Handler:      router,
		ReadTimeout:  serverCfg.ReadTimeout,
		WriteTimeout: serverCfg.WriteTimeout,
		IdleTimeout:  serverCfg.IdleTimeout,
	}

	// Start server
	go func() {
		slog.Info("starting server", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", "error", err)
		}
	}()

	// Wait for shutdown signal
	<-ctx.Done()
	slog.Info("shutting down server")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), serverCfg.ShutdownTimeout)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutting down server: %w", err)
	}

	slog.Info("server stopped")
	return nil
}
