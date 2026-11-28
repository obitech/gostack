.PHONY: test test-race vet lint lint-fix check release release-bump release-template release-dry-run release-check \
	require-version require-clean require-synced-main

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
#
# Releasing takes two steps, because the template's go.mod can only require a
# library version once its tag exists:
#
#   1. make release VERSION=vX.Y.Z
#      Tags and pushes the library, then opens a PR that bumps the template's
#      gostack requirement to vX.Y.Z (release-bump, which can be re-run).
#   2. Merge that PR, then: make release-template VERSION=vX.Y.Z
#      Tags the merge commit as templates/full/vX.Y.Z, which is what
#      `gonew github.com/obitech/gostack/templates/full@vX.Y.Z` resolves.
TEMPLATE_DIR := templates/full
LIBRARY_MODULE := github.com/obitech/gostack
TEMPLATE_BRANCH = release/template-$(VERSION)

release: require-version require-clean require-synced-main
	@echo "Creating and pushing tag $(VERSION)..."
	git tag -a $(VERSION) -m "Release $(VERSION)"
	git push origin $(VERSION)
	$(MAKE) release-bump VERSION=$(VERSION)

release-bump: require-version require-clean require-synced-main
	git switch -c $(TEMPLATE_BRANCH)
	cd $(TEMPLATE_DIR) && GOWORK=off go get $(LIBRARY_MODULE)@$(VERSION) && GOWORK=off go mod tidy
	git add $(TEMPLATE_DIR)/go.mod $(TEMPLATE_DIR)/go.sum
	git commit -m "chore(template): require gostack $(VERSION)"
	git push -u origin $(TEMPLATE_BRANCH)
	gh pr create --fill --base main --head $(TEMPLATE_BRANCH)
	git switch main
	@echo "Merge the PR, then run: make release-template VERSION=$(VERSION)"

release-template: require-version require-clean
	git switch main
	git pull --ff-only origin main
	@got=$$(cd $(TEMPLATE_DIR) && GOWORK=off go list -m -f '{{.Version}}' $(LIBRARY_MODULE)); \
	test "$$got" = "$(VERSION)" || { \
		echo "$(TEMPLATE_DIR) requires $(LIBRARY_MODULE) $$got, want $(VERSION). Merge the bump PR first." >&2; \
		exit 1; \
	}
	git tag -a $(TEMPLATE_DIR)/$(VERSION) -m "Template $(VERSION)"
	git push origin $(TEMPLATE_DIR)/$(VERSION)

require-version:
	@echo "$(VERSION)" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$$' || { \
		echo "VERSION must look like vX.Y.Z (got '$(VERSION)'). Usage: make $(firstword $(MAKECMDGOALS)) VERSION=v0.2.1" >&2; \
		exit 1; \
	}

require-clean:
	@test -z "$$(git status --porcelain)" || { echo "Working tree is not clean." >&2; exit 1; }

require-synced-main:
	@test "$$(git rev-parse --abbrev-ref HEAD)" = main || { echo "Must be on main." >&2; exit 1; }
	@git fetch -q origin main
	@test "$$(git rev-parse HEAD)" = "$$(git rev-parse origin/main)" || { echo "main is not in sync with origin/main." >&2; exit 1; }

release-dry-run:
	goreleaser release --snapshot --clean

release-check:
	goreleaser check
