# tui-cardman

[![Build](https://github.com/laiambryant/tui-cardman/actions/workflows/build.yml/badge.svg)](https://github.com/laiambryant/tui-cardman/actions/workflows/build.yml)
[![Release](https://github.com/laiambryant/tui-cardman/actions/workflows/release.yml/badge.svg)](https://github.com/laiambryant/tui-cardman/releases/latest)
[![Go Version](https://img.shields.io/github/go-mod/go-version/laiambryant/tui-cardman)](go.mod)
[![License](https://img.shields.io/github/license/laiambryant/tui-cardman)](LICENSE)

A terminal UI for managing your trading card collection — Pokémon TCG, Yu-Gi-Oh!, Magic: The Gathering, and One Piece. Track cards, build decks, maintain wishlists, and import card data, all from the command line.

> **Status**: v0.2.0 — core features are functional. Expect rough edges.

## Supported games

| Game                 | CLI alias  | Data source    |
|----------------------|------------|----------------|
| Pokémon TCG          | `pokemon`  | TCGDex API     |
| Yu-Gi-Oh!            | `yugioh`   | YGOPRODeck API |
| Magic: The Gathering | `mtg`      | MTG API        |
| One Piece            | `onepiece` | OPTCG API      |

Each game has its own importer and a dedicated card-detail renderer (HP/attacks for Pokémon, ATK/DEF and link markers for Yu-Gi-Oh!, mana cost and P/T for MTG, cost/power/counter/life for One Piece).

## Features

- **Collection management** — track cards you own with quantities and rarity-colored set-completion grids
- **Deck builder** — build decks with per-game validation (see below)
- **Lists / wishlists** — maintain custom card lists with color labels
- **Card import** — import sets and cards for all four games, from the TUI import queue or the CLI
- **Fuzzy search** — typo-tolerant search across name, number, rarity, and artist
- **Price tracking** — TCGPlayer and Cardmarket price snapshots with collection value history
- **Import / export** — export collections, decks, and lists to CSV, text, and PTCGO; import collections from CSV
- **Multi-user SSH mode** — run as a shared SSH server; each user gets an isolated collection and settings
- **SQLite backend** — single-file database, pure Go ([modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite)), no external dependencies

### Deck validation rules

| Game                 | Deck size                    | Copy limit                    |
|----------------------|------------------------------|-------------------------------|
| Pokémon TCG          | exactly 60                   | max 4 (basic energy exempt)   |
| Yu-Gi-Oh!            | 40–60 main deck              | max 3                         |
| Magic: The Gathering | at least 60                  | max 4 (basic lands exempt)    |
| One Piece            | exactly 51 (1 Leader + 50)   | max 4                         |

## Installation

### Download a binary (recommended)

Grab the latest release from the [releases page](https://github.com/laiambryant/tui-cardman/releases). Binaries are provided for Linux (amd64/arm64), macOS (amd64/arm64), and Windows (amd64).

```bash
# Linux amd64 example
VERSION=v0.2.0
curl -L "https://github.com/laiambryant/tui-cardman/releases/download/${VERSION}/cardman_${VERSION}_linux-amd64.tar.gz" | tar xz
sudo mv cardman /usr/local/bin/
```

### Build from source

Requires Go 1.25+. The build is pure Go (SQLite via [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite)), so no C compiler is needed and cross-compiling is a plain `GOOS`/`GOARCH` matter.

```bash
git clone https://github.com/laiambryant/tui-cardman.git
cd tui-cardman
make build
# or: go build -o cardman ./cmd/main.go
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
# 1. Configure (optional — every variable has a working default)
cp .env.example .env

# 2. Run migrations
./cardman migrate

# 3. Import card data (initial setup — takes a few minutes per game)
./cardman import all pokemon

# 4. Launch the TUI
./cardman serve
```

Sets can also be imported from inside the TUI via the **Import Sets** mode, which queues sets per game and shows live progress.

## Configuration

Configuration is read from the environment, with `.env` loaded automatically if present. Copy `.env.example` to `.env` and edit as needed:

```bash
cp .env.example .env
```

| Variable       | Description                                           | Default                                   |
|----------------|-------------------------------------------------------|-------------------------------------------|
| `DB_DSN`       | SQLite DSN                                            | `file:cardman.db?_pragma=foreign_keys(1)` |
| `LOG_LEVEL`    | Logging verbosity (`DEBUG`, `INFO`, `WARN`, `ERROR`)  | `INFO`                                    |
| `SSH_PORT`     | SSH server port                                       | `2222`                                    |
| `SSH_HOST_KEY` | Path to SSH host key file                             | `~/.ssh/cardman_host_key`                 |
| `API_KEY`      | [TCGDex](https://tcgdex.dev/) API key (optional, increases rate limits) | —                       |

## SSH Server Mode

Run cardman as a shared SSH server so multiple users can access their collections remotely:

```bash
# Generate a host key (first time only)
ssh-keygen -t ed25519 -f ~/.ssh/cardman_host_key -N ""

./cardman serve-ssh
```

Users connect with: `ssh -p 2222 username@your-server`

Each user registers an account on first connection and gets an isolated collection, decks, lists, and per-user settings stored in the database.

## Importing Card Data

The `import` command group works for every supported game. `<game>` is one of `pokemon`, `yugioh`, `mtg`, `onepiece`.

```bash
# List available sets from the API
./cardman import list-sets pokemon

# Import everything (initial setup)
./cardman import all pokemon

# Import only sets not already in your database — run via cron to stay current
./cardman import new pokemon

# Import specific sets by ID
./cardman import sets pokemon base1 jungle fossil
```

The legacy Pokémon-only commands (`import-full`, `import-updates`, `import-sets`, `list-sets`) still work but are superseded by the `import` group.

## Available Commands

```
cardman migrate                                Run database migrations (--driver, --dsn, --dir)
cardman serve                                  Launch the TUI (local mode)
cardman serve-ssh                              Start SSH server (multi-user mode)

cardman import list-sets <game>                List available sets from the API
cardman import sets <game> <set-id...>         Import specific sets
cardman import all <game>                      Import all sets and cards
cardman import new <game>                      Import only new sets

cardman cards list <game>                      List cards (--search, --json)
cardman cards info <game> <set-code> <number>  Show details for one card (--json)

cardman collection list <game>                 List your collection
cardman collection add <game> <set> <n> <qty>  Add cards to your collection
cardman collection remove <game> <set> <n> <qty>  Remove cards
cardman collection import <game> <csv-file>    Import a collection from CSV
cardman collection export <game>               Export (--format csv|text|ptcgo, --output)

cardman decks list|create|delete|show          Manage decks (create: --format)
cardman decks add-card|remove-card             Edit deck contents
cardman decks validate <deck-id>               Validate a deck against game rules
cardman decks export <deck-id>                 Export (--format csv|text|ptcgo, --output)

cardman lists list|create|delete|show          Manage lists (create: --description, --color)
cardman lists add-card|remove-card             Edit list contents
cardman lists export <list-id>                 Export (--format csv|text, --output)
```

Run any command with `--help` for the full flag reference.

> **Note**: the `cards`, `collection`, `decks`, and `lists` commands operate on the first local user — run `cardman serve` once first to create it.

```
make help                Show all Make targets
make build               Build the binary
make test                Run all tests
make test-coverage       Run tests with coverage report
make fmt                 Format code
make lint                Run golangci-lint
make migrate             Run database migrations
make clean               Remove build artifacts
```

## License

[MIT](LICENSE)
