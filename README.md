# gostack

A collection of reusable Go packages for building web applications with:
- Structured JSON logging (slog)
- OpenTelemetry metrics with Prometheus export
- Embedded SPA frontend serving with dev server proxy support

## Installation

```bash
go get github.com/obitech/gostack
```

## Packages

### `log` - JSON Logger

Simple slog JSON logger factory.

```go
import (
    "log/slog"
    golog "github.com/obitech/gostack/log"
)

// Create JSON logger to stdout
logger := golog.NewJSON(slog.LevelInfo)
slog.SetDefault(logger)

// Or with custom writer
logger := golog.NewJSONWriter(os.Stderr, slog.LevelDebug)
```

### `metrics` - OpenTelemetry + Prometheus

Sets up OpenTelemetry with Prometheus exporter and provides HTTP request metrics middleware.

```go
import "github.com/obitech/gostack/metrics"

// Initialize metrics (call once at startup)
shutdown, err := metrics.Setup("myapp")  // namespace prefixes all metrics
if err != nil {
    return err
}
defer shutdown(context.Background())

// Create HTTP metrics middleware
httpMetrics, err := metrics.NewHTTPMetrics(metrics.HTTPMetricsConfig{
    MeterName:    "myapp/http",
    ExcludePaths: []string{"/internal/", "/health"},
})
if err != nil {
    return err
}

// Use with chi router
r := chi.NewRouter()
r.Use(httpMetrics.Middleware())

// Expose Prometheus endpoint
r.Handle("/internal/metrics", metrics.Handler())
```

Metrics exported:
- `myapp_http_request_duration_seconds` - Request latency histogram
- `myapp_http_request_size_bytes` - Request body size histogram
- `myapp_http_response_size_bytes` - Response body size histogram

All metrics include labels: `route`, `method`, `status`.

### `config` - Struct-Based CLI Config

Declarative configuration for Cobra commands using struct tags. Handles flag registration and value loading with priority: flag > env > default.

```go
import (
    "github.com/obitech/gostack/config"
    "github.com/spf13/cobra"
)

type ServerConfig struct {
    Addr    string        `flag:"addr,a" env:"ADDR" default:":8080" desc:"listen address"`
    Timeout time.Duration `flag:"timeout" env:"TIMEOUT" default:"30s" desc:"request timeout"`
    Debug   bool          `flag:"debug,d" env:"DEBUG" default:"false" desc:"enable debug mode"`
    Workers int           `flag:"workers,w" env:"WORKERS" default:"4" desc:"worker count"`
}

var cfg ServerConfig

func init() {
    // Registers --addr/-a, --timeout, --debug/-d, --workers/-w flags
    if err := config.RegisterFlags(cmd, &cfg); err != nil {
        panic(err)
    }
}

func run(cmd *cobra.Command, args []string) error {
    // Loads values: flag if set, else env if set, else default
    if err := config.Load(cmd, &cfg); err != nil {
        return err
    }
    // Use cfg.Addr, cfg.Timeout, etc.
}
```

Supported types: `string`, `int`, `bool`, `time.Duration`

Struct tags:
- `flag:"name"` or `flag:"name,x"` - flag name and optional single-char shorthand
- `env:"VAR_NAME"` - environment variable to check
- `default:"value"` - default value (parsed per field type)
- `desc:"text"` - description shown in `--help`

### `frontend` - SPA Serving

Serves embedded SPA frontend with dev server proxy support.

```go
import (
    "github.com/obitech/gostack/frontend"
    "myapp/web"  // Your embed.FS
)

handler, err := frontend.NewHandler(frontend.Config{
    DevServerURL: os.Getenv("VITE_DEV_SERVER"),  // e.g., "http://localhost:5173"
    Assets:       web.Dist,                       // embed.FS
    Subdir:       "dist",                         // subdirectory in Assets
})
if err != nil {
    return err
}

// Mount as catch-all route
r.Handle("/*", handler)
```

Features:
- Development: Proxies to Vite/webpack dev server for hot reload
- Production: Serves embedded static files
- SPA routing: Returns `index.html` for unknown paths

## Project Templates

Use `gonew` to create new projects from templates:

```bash
# Full-stack (Go API + Database + React frontend)
gonew github.com/obitech/gostack/templates/full github.com/you/myapp
```

## License

MIT
