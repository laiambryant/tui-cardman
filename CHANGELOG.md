# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-03-01

Initial alpha release of tui-cardman.

### Added

- Terminal UI for managing a Pokemon TCG card collection (BubbleTea / Charm)
- **Collection view** — browse your cards with set completion tracking
- **Deck builder** — create and validate 60-card decks; enforces 4-copy rule and basic energy exemption
- **Lists / wishlists** — custom card lists with color labels
- **Card import** — full and incremental import from the TCGDex Pokemon TCG API
- **Import specific sets** by set ID
- **Price tracking** — TCGPlayer and CardMarket price snapshots
- **Export** — CSV, plain text, and PTCGO format export
- **Multi-user SSH server mode** — share a cardman instance over SSH
- **SQLite backend** via `mattn/go-sqlite3` (CGO); single-file database
- **14 database migrations** covering users, card games, sets, cards, collections, lists, decks, prices, import runs, and button configuration
- **Runtime keybinding configuration** — per-user configurable keybindings stored in the database (SSH mode) or locally
- Cobra CLI with subcommands: `migrate`, `serve`, `serve-ssh`, `import-full`, `import-updates`, `import-sets`, `list-sets`
- `.env` based configuration with `.env.example` template
- Docker support: multi-stage `Dockerfile`, `docker-compose.yml`
- GoReleaser config with native CGO builds for Linux (amd64/arm64) and macOS (amd64/arm64)
- Test coverage for: auth, config, db migrations, export/import, runtime config, deck service, list service, card game service, button config service, sets service, user service, user collection service, prices service, import runs service, pokemon TCG integration

### Known Limitations

- TUI has no unit tests (hard to test terminal rendering; tested manually)
- Magic: The Gathering and Yu-Gi-Oh! card games are seeded in the database but not yet fully supported in the import pipeline
- Price data requires manual import runs; no automatic scheduling built in
- SSH host key must be provisioned manually before running `serve-ssh`

[Unreleased]: https://github.com/laiambryant/tui-cardman/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/laiambryant/tui-cardman/releases/tag/v0.1.0
