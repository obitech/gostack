APP_NAME := app
SERVICES ?= api postgres

# Run targets
run-server:
	go run ./cmd/cli run-server

run-dev:
	VITE_DEV_SERVER=http://localhost:5173 go run ./cmd/cli run-server

# Docker targets
start: stop
	docker compose up -d $(SERVICES)

start-build: stop
	docker compose up -d $(SERVICES) --build

start-db: stop
	docker compose up -d postgres

stop:
	docker compose down || true

logs:
	docker compose logs -f

# Build targets
build: build-web
	go build -o bin/$(APP_NAME) ./cmd/cli

build-web:
	cd web && npm run build

build-docker:
	docker build -t $(APP_NAME) .

# Test targets
test:
	go test -v ./...

test-web:
	cd web && npm test

test-integration:
	go test -v -tags=integration ./...

test-local:
	@trap '$(MAKE) stop' EXIT; \
	$(MAKE) lint && \
	$(MAKE) vet && \
	$(MAKE) test && \
	$(MAKE) test-web && \
	$(MAKE) start-db && sleep 5 && \
	$(MAKE) test-integration

# Lint targets
lint:
	golangci-lint run ./...

lint-fix:
	golangci-lint run --fix ./...

vet:
	go vet ./...

# Code generation
generate:
	go run ./cmd/cli generate-openapi > openapi.yml
	cd web && npm run generate
