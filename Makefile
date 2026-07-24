# effectiveNext Makefile

.PHONY: all build test test-race bench lint fmt vet clean doctor help validate-tag release-snapshot release

BINARY_NAME := effective-next
MAIN_PKG := ./cmd/effective-next
GO := go
GOFLAGS :=

all: fmt vet test build

build:
	$(GO) build $(GOFLAGS) -o bin/$(BINARY_NAME) $(MAIN_PKG)

test:
	$(GO) test ./... -count=1

test-race:
	$(GO) test ./... -race -count=1

bench:
	$(GO) test ./... -run=^$$ -bench=. -benchmem

lint:
	golangci-lint run ./...

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

clean:
	rm -rf bin/ .effective-next/

doctor: build
	./bin/$(BINARY_NAME) doctor

validate-tag:
	@TAG=$$(git describe --tags --exact-match 2>/dev/null); \
	if [ -z "$$TAG" ]; then echo "No tag found on current commit"; exit 1; fi; \
	echo "Found tag: $$TAG"; \
	echo "$$TAG" | grep -qE '^v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?$$' || \
		{ echo "Invalid tag format: $$TAG (expected vX.Y.Z)"; exit 1; }; \
	echo "Tag format is valid"

release-snapshot:
	goreleaser release --snapshot --clean

release:
	goreleaser release --clean

help:
	@echo "Available targets:"
	@echo "  build             Build the effective-next binary"
	@echo "  test              Run unit tests"
	@echo "  test-race         Run unit tests with race detector"
	@echo "  bench             Run benchmarks"
	@echo "  lint              Run linter"
	@echo "  fmt               Format Go code"
	@echo "  vet               Run go vet"
	@echo "  clean             Remove build artifacts and cache"
	@echo "  doctor            Build and run effective-next doctor"
	@echo "  validate-tag      Validate current git tag format (semver)"
	@echo "  release-snapshot  Build release snapshot locally (no publish)"
	@echo "  release           Build and publish release via GoReleaser"
