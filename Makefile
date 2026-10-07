# Common tasks. "make check" runs what CI runs.

GOLANGCI_LINT_VERSION := v2.14.0
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X github.com/DharmaBytesX/pricewatch-cli/internal/version.Version=$(VERSION)

.PHONY: build test lint lint-docker docs check-docs check clean

build: ## Build ./pricewatch
	go build -ldflags "$(LDFLAGS)" -o pricewatch ./cmd/pricewatch

test: ## Run the tests with the race detector
	go test -race -count=1 ./...

lint: ## Run golangci-lint (install: go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION))
	golangci-lint run ./...

lint-docker: ## Run golangci-lint in Docker, without installing it
	docker run --rm -v "$(CURDIR):/app" -w /app golangci/golangci-lint:$(GOLANGCI_LINT_VERSION) golangci-lint run ./...

docs: ## Write the command reference in docs/commands
	go run ./tools/gendocs docs/commands

check-docs: docs ## Fail when docs/commands does not match the commands
	git diff --exit-code -- docs/commands
	@test -z "$$(git status --porcelain -- docs/commands)" || (echo "docs/commands has new files: commit them"; exit 1)

check: lint test check-docs ## Everything CI checks

clean:
	rm -rf pricewatch dist coverage.out
