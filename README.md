# tui-cardman

[![Build](https://github.com/laiambryant/tui-cardman/actions/workflows/build.yml/badge.svg)](https://github.com/laiambryant/tui-cardman/actions/workflows/build.yml)
[![Release](https://github.com/laiambryant/tui-cardman/actions/workflows/release.yml/badge.svg)](https://github.com/laiambryant/tui-cardman/releases/latest)
[![Go Version](https://img.shields.io/github/go-mod/go-version/laiambryant/tui-cardman)](go.mod)
[![License](https://img.shields.io/github/license/laiambryant/tui-cardman)](LICENSE)

A terminal UI for managing your trading card collection. Track cards, build decks, maintain wishlists, and import card data — all from the command line.

> **Status**: v0.2.0 — core features are functional. Expect rough edges.

## Features

- **Collection management** — track cards you own with quantities
- **Deck builder** — build decks for each supported game
- **Lists / wishlists** — maintain custom card lists with color labels
- **Card import** — import all Pokemon TCG sets and cards from the TCGDex API
- **Price tracking** — TCGPlayer and CardMarket price snapshots
- **Export** — export collections to CSV, text, and PTCGO formats
- **Multi-user SSH mode** — run as a shared SSH server for multiple users
- **SQLite backend** — single-file database, no external dependencies

## Installation

### Download a binary (recommended)

Grab the latest release from the [releases page](https://github.com/laiambryant/tui-cardman/releases). Linux (amd64/arm64) and macOS (amd64/arm64) binaries are provided.

```bash
# Linux amd64 example
VERSION=v0.2.0
curl -L "https://github.com/laiambryant/tui-cardman/releases/download/${VERSION}/cardman_${VERSION}_linux-amd64.tar.gz" | tar xz
sudo mv cardman /usr/local/bin/
```

### Build from source

Requires Go 1.25+ and GCC (for CGO/SQLite).

```bash
git clone https://github.com/laiambryant/tui-cardman.git
cd tui-cardman
CGO_ENABLED=1 go build -o cardman ./cmd/main.go
```

### Docker

```bash
# Run interactively (TUI mode)
docker compose run --rm cardman serve

# Or with Docker directly
docker build -t cardman .
docker run -it -v cardman_data:/app/data cardman serve
```

## Quick Start

```bash
# 1. Configure
cp .env.example .env
# Edit .env — set DB_DSN and optionally API_KEY

# 2. Run migrations
./cardman migrate

# 3. Import Pokemon TCG data (initial setup — takes a few minutes)
./cardman import-full

# 4. Launch the TUI
./cardman serve
```

## Configuration

Copy `.env.example` to `.env` and edit as needed:

```bash
cp .env.example .env
```

| Variable        | Description                                      | Default                         |
|-----------------|--------------------------------------------------|---------------------------------|
| `DB_DSN`        | SQLite DSN                                       | `file:cardman.db?_fk=1`         |
| `LOG_LEVEL`     | Logging verbosity (`DEBUG`, `INFO`, `WARN`, `ERROR`) | `INFO`                     |
| `SSH_PORT`      | SSH server port                                  | `2222`                          |
| `SSH_HOST_KEY`  | Path to SSH host key file                        | `~/.ssh/cardman_host_key`       |
| `API_KEY`       | Pokemon TCG API key (optional, increases limits) | —                               |

## SSH Server Mode

Run cardman as a shared SSH server so multiple users can access their collections remotely:

```bash
# Set in .env:
# SSH_PORT=2222
# SSH_HOST_KEY=./host_key

./cardman serve-ssh
```

If `SSH_HOST_KEY` does not exist, `serve-ssh` will generate an ED25519 host key automatically.

Users connect with: `ssh -p 2222 username@your-server`

## Build

This project requires CGO because it uses [go-sqlite3](https://github.com/mattn/go-sqlite3). A C compiler (GCC or Clang) must be available.

```bash
# Build
CGO_ENABLED=1 go build -o cardman ./cmd/main.go

# Using Make
make build

# Run tests
go test ./...
```

## Importing Card Data

### Full import (initial setup)

```bash
./cardman import-full
# or: make import-full
```

Fetches all sets and cards from the TCGDex Pokemon TCG API. This can take several minutes on first run.

### Incremental import (keep up to date)

```bash
./cardman import-updates
# or: make import-updates
```

Only imports sets not already in your database. Run weekly via cron to pick up new releases.

### Import specific sets

```bash
./cardman import-sets base1
./cardman import-sets base1 jungle fossil
```

## Available Commands

Primary commands:

```
cardman migrate          Run database migrations
cardman import-full      Import all Pokemon TCG data
cardman import-updates   Import only new sets
cardman import-sets      Import one or more specific sets by ID
cardman list-sets        List available sets from the API
cardman serve            Launch the TUI (local mode)
cardman serve-ssh        Start SSH server (multi-user mode)
```

Grouped CLI commands:

```text
cardman cards list <game>
cardman cards info <game> <set-code> <number>
cardman collection list <game>
cardman collection add <game> <set-code> <number> <quantity>
cardman collection remove <game> <set-code> <number> <quantity>
cardman collection import <game> <csv-file>
cardman collection export <game>
cardman lists list <game>
cardman lists create <game> <name>
cardman lists show <list-id>
cardman lists add-card <list-id> <set-code> <number> <quantity>
cardman decks list <game>
cardman decks create <game> <name>
cardman decks show <deck-id>
cardman decks add-card <deck-id> <set-code> <number> <quantity>
cardman import list-sets <game>
cardman import sets <game> <set-id...>
cardman import all <game>
cardman import new <game>
```

```
make help                Show all Make targets
make build               Build the binary
make test                Run all tests
make test-coverage       Run tests with coverage report
make fmt                 Format code
make lint                Run golangci-lint
make migrate             Run database migrations
make import-full         Import all Pokemon TCG data
make import-updates      Import only new sets
make clean               Remove build artifacts
```

## License

[MIT](LICENSE)
