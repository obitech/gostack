.PHONY: test test-race vet lint lint-fix check

# Test targets
test:
	go test ./...

test-race:
	go test -race ./...

# Vet
vet:
	go vet ./...

# Lint targets
lint:
	golangci-lint run ./...

lint-fix:
	golangci-lint run --fix ./...

# Run all checks (vet + lint + tests with race detector)
check: vet lint test-race
