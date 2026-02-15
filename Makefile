# Termblog Makefile

# Version info
VERSION ?= 0.5.0
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build info
BINARY := termblog
MODULE := github.com/ajmeese7/termblog
LDFLAGS := -X $(MODULE)/internal/version.Version=$(VERSION) \
           -X $(MODULE)/internal/version.Commit=$(COMMIT) \
           -X $(MODULE)/internal/version.Date=$(DATE)

# Go commands
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOMOD := $(GOCMD) mod

.PHONY: all build clean test test-v test-e2e tidy version release tag help

## Build the binary
build:
	$(GOBUILD) -tags fts5 -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/termblog

## Build for production (stripped binary)
build-prod:
	$(GOBUILD) -tags fts5 -ldflags "$(LDFLAGS) -s -w" -o $(BINARY) ./cmd/termblog

## Run unit tests (use `make test-v` for verbose output)
test:
	$(GOTEST) -tags fts5 ./...

## Run unit tests with verbose output
test-v:
	$(GOTEST) -tags fts5 -v ./...

## Run end-to-end browser tests (requires running server: make build && ./termblog serve)
test-e2e:
	cd tests/e2e && npx playwright test

## Tidy dependencies
tidy:
	$(GOMOD) tidy

## Clean build artifacts
clean:
	rm -f $(BINARY)

## Show current version
version:
	@echo "Version: $(VERSION)"
	@echo "Commit:  $(COMMIT)"
	@echo "Date:    $(DATE)"

## Create a new release tag (usage: make tag VERSION=0.2.0)
tag:
	@if [ -z "$(VERSION)" ]; then echo "VERSION is required"; exit 1; fi
	@echo "Creating tag v$(VERSION)..."
	git tag -a v$(VERSION) -m "Release v$(VERSION)"
	@echo "Tag v$(VERSION) created. Push with: git push origin v$(VERSION)"

## Create and push a release tag
release: tag
	git push origin v$(VERSION)
	@echo "Released v$(VERSION)"

## Show help
help:
	@echo "Termblog Makefile"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  build       Build the binary"
	@echo "  build-prod  Build production binary (stripped)"
	@echo "  test        Run unit tests"
	@echo "  test-e2e    Run browser e2e tests (server must be running)"
	@echo "  tidy        Tidy dependencies"
	@echo "  clean       Remove build artifacts"
	@echo "  version     Show version info"
	@echo "  tag         Create a git tag (VERSION=x.y.z)"
	@echo "  release     Create and push a release tag"
	@echo "  help        Show this help"
	@echo ""
	@echo "Examples:"
	@echo "  make build"
	@echo "  make build VERSION=0.2.0"
	@echo "  make release VERSION=0.2.0"

# Default target
all: build
