# gostack

## Repository layout

```
gostack/
├── config/       # Struct-tag based Cobra flag+env config (library package)
├── frontend/     # SPA serving with embedded FS or Vite dev-server proxy (library package)
├── log/          # JSON slog logger and HTTP request logging middleware (library package)
├── metrics/      # OpenTelemetry + Prometheus setup and HTTP metrics middleware (library package)
└── templates/
    └── full/     # Full-stack template: chi + Huma API, pgx + migrations, React/Vite frontend
```

The **root module** (`github.com/obitech/gostack`) is the importable library.
The **template** (`templates/full/`, module `github.com/example/app`) is scaffolded via:

```sh
gonew github.com/obitech/gostack/templates/full github.com/yourorg/yourapp
```

## Development

All commands are run from the relevant module root.

### Library (root)

```sh
go test -race ./...       # run all tests
go vet ./...              # vet
golangci-lint run ./...   # lint
```

### Template (`templates/full/`)

```sh
make run-dev              # start Go server + Vite dev server
make test                 # go test ./...
make test-integration     # integration tests (requires running postgres)
make lint                 # golangci-lint
make generate             # regenerate openapi.yml + TypeScript types
make start                # docker compose up (postgres + api)
make stop                 # docker compose down
```

## Architecture decisions

### config package
Uses reflection + struct tags to register Cobra flags. Priority: CLI flag > env var > default. Supported types: `string`, `int`, `bool`, `time.Duration`. No config file loading by design — use environment variables in production.

### API layer (template)
Routes are registered with [Huma v2](https://github.com/danielgtaylor/huma) on top of a chi router. Huma auto-generates OpenAPI 3.1 from handler types. Run `make generate` after changing handler signatures to update `openapi.yml` and TypeScript types.

### Middleware ordering (router.go)
`RequestID` → `SlogMiddleware` → `Recoverer` → CORS → Timeout → HTTPMetrics

`RequestID` must be first so the request ID is in the context before `SlogMiddleware`'s defer reads it.

### Health probes
- `/api/v1/healthz` — liveness (process alive, no dependencies)
- `/api/v1/readyz` — readiness (pings database)
- `/api/v1/health` — alias for readyz (backward compat)

### Database
pgxpool with embedded SQL migrations (`golang-migrate`). Migrations run automatically on startup. Use raw SQL via `db.Pool()`. OpenTelemetry tracing is wired but uses a noop tracer provider by default.

### Frontend
In development, requests to `/*` are proxied to the Vite dev server (`VITE_DEV_SERVER`). In production, the compiled `web/dist/` is embedded and served with SPA fallback (unknown paths → `index.html`).

## Conventions

- Follow Uber's Go Style Guide
- Wrap all errors: `fmt.Errorf("context: %w", err)`
- Use `errors.Is`/`errors.As` for error checks
- Use `any` instead of `interface{}`
- Use testify (`assert`, `require`) for all new tests
- Exported functions and types must have doc comments
- Package-level docs required for all non-generated packages
