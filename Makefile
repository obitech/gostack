.PHONY: test test-race vet lint lint-fix check release release-dry-run release-check

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

# Release targets
release:
ifndef VERSION
	$(error VERSION is required. Usage: make release VERSION=v0.2.0)
endif
	@echo "Creating and pushing tag $(VERSION)..."
	git tag -a $(VERSION) -m "Release $(VERSION)"
	git push origin $(VERSION)

release-dry-run:
	goreleaser release --snapshot --clean

release-check:
	goreleaser check
