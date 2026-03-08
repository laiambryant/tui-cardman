# Contributing

Thank you for your interest in contributing to tui-cardman.

## Prerequisites

- Go 1.24 or later
- GCC or Clang (required for CGO / SQLite)
- `make` (optional but convenient)

## Getting Started

```bash
git clone https://github.com/laiambryant/tui-cardman.git
cd tui-cardman
cp .env.example .env
go mod download
```

## Building

```bash
# Build binary
CGO_ENABLED=1 go build -o cardman ./cmd/main.go

# Using Make
make build
```

## Running Tests

```bash
go test ./...

# With coverage
go test -cover ./...
make test-coverage
```

Tests use an in-memory SQLite database and apply all migrations automatically. No external services are required.

## Code Style

- Run `go fmt ./...` before committing
- Run `go vet ./...` to catch common issues
- Interfaces for all service layers; dependency injection via constructors

## Project Structure

``` txt
cmd/            CLI entry points (Cobra commands)
internal/
  auth/         Authentication and session management
  config/       Environment-based configuration loading
  db/           Database helpers and migrations (SQL files in db/migrations/)
  export/       CSV, text, and PTCGO export/import
  logging/      Logging utilities (query sanitization)
  model/        Shared data models and structs
  pokemontcg/   TCGDex API client
  runtimecfg/   Per-user runtime keybinding configuration
  services/     Business logic layer (one package per domain)
  testutil/     Shared test helpers (SetupTestDB, ApplyTestMigrations)
  tui/          BubbleTea terminal UI components
```

## Database Migrations

Migrations live in `internal/db/migrations/` as numbered `*.up.sql` / `*.down.sql` pairs. Add new migrations as the next numbered file. Tests apply all migrations automatically via `testutil.ApplyTestMigrations`.

## Submitting a Pull Request

1. Fork the repository and create a feature branch from `main`
2. Write tests for any new service-layer code
3. Ensure `go test ./...`, `go fmt ./...`, and `go vet ./...` all pass
4. Open a PR with a clear description of what changed and why

## Reporting Bugs

Open a GitHub issue with:

- Go version (`go version`)
- OS and architecture
- Steps to reproduce
- Expected vs actual behaviour
