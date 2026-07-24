# effectiveNext

A high-performance build accelerator for Next.js, written in Go.

**effectiveNext is not a replacement for Next.js.** It acts as a preprocessing, caching,
dependency analysis, and orchestration layer that reduces the work performed by
`next build` and `next dev`.

## Goals

- 2–10× faster cold builds
- 10–30× faster incremental builds
- 50–80% reduction in cache size
- Lower RAM consumption during builds
- Near 100% CPU utilization
- Deterministic and reproducible builds
- Zero required application source code changes
- Cross-platform support (Linux, macOS, Windows)

## Commands

```bash
effectiveNext build        # Optimized production build
effectiveNext dev          # Development mode with watcher
effectiveNext clean        # Remove effectiveNext caches and build state
effectiveNext cache        # Inspect and manage cache
effectiveNext analyze      # Dependency analysis
effectiveNext doctor       # Environment validation
effectiveNext graph        # Export dependency graph
effectiveNext benchmark    # Run performance benchmarks
effectiveNext watch        # Watch files and invalidate caches
```

## Quick Start

```bash
# Verify your environment
effectiveNext doctor

# Build a Next.js project
effectiveNext build

# Start development mode
effectiveNext dev
```

## Build Flags

```bash
effectiveNext build --dry-run          # Preview without building
effectiveNext build --skip-next        # Preprocess only
effectiveNext build --workers 8        # Use 8 worker goroutines
effectiveNext build --no-cache         # Disable caching
effectiveNext dev -p 8080              # Start on port 8080
effectiveNext dev -H 0.0.0.0           # Listen on all interfaces
```

## Package Overview

| Package | Description |
|---------|-------------|
| `pkg/cli` | CLI commands and flags |
| `pkg/config` | Configuration loading and validation |
| `pkg/scanner` | Concurrent filesystem scanner |
| `pkg/graph` | Dependency graph engine |
| `pkg/parser` | SWC WASM parser + regex fallback |
| `pkg/cache` | Binary cache with Zstd compression |
| `pkg/db` | SQLite build database |
| `pkg/routes` | Next.js route detection |
| `pkg/css` | CSS dependency tracking |
| `pkg/assets` | Asset optimization |
| `pkg/scheduler` | DAG-based task scheduler |
| `pkg/watch` | Filesystem watcher |
| `pkg/manifest` | Manifest generation |
| `pkg/engine` | Incremental build engine |
| `pkg/observability` | Metrics and build tracking |
| `pkg/plugins` | Plugin lifecycle system |
| `pkg/fs` | Cross-platform filesystem helpers |
| `pkg/hash` | Content hashing (xxHash) |

## Development

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test ./... -v

# Run benchmarks
go test ./... -bench=. -benchtime=100ms

# Build the binary
go build -o bin/effectiveNext ./cmd/effectiveNext

# Lint the code
go vet ./...
```

## Documentation

| Document | Description |
|----------|-------------|
| [Installation](./INSTALL.md) | Prerequisites, build from source, configuration, troubleshooting |
| [Usage Guide](./USAGE.md) | Complete walkthrough of every command and feature |
| [Architecture](./ARCHITECTURE.md) | System design, data flow, concurrency model |
| [Benchmarks](./BENCHMARKS.md) | Real benchmark results with analysis and scaling data |
| [Developer Guide](./DEVELOPER.md) | Build, test, lint, package deep dive, architecture decisions |
| [Plugin Guide](./PLUGIN.md) | Creating, registering, and testing plugins |

## License

MIT
