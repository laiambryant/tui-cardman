# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run

Requires `CGO_ENABLED=1` (set in Makefile) due to `mattn/go-sqlite3`. A C compiler (gcc) must be available.

```bash
make build          # Build binary (./cardman)
make run            # Build + serve TUI
make test           # Run all tests
make test-coverage  # Tests with coverage
make lint           # golangci-lint (install: make lint-install)
make fmt            # Format with gofumpt (install: make gofumpt-install)
make check          # fmt + vet + test
```

Run a single test: `go test -v -run TestName ./internal/services/cards/`

Application commands: `./cardman serve`, `./cardman migrate`, `./cardman import-full`, `./cardman serve-ssh`

## Architecture

Go 1.25 TUI application using Charmbracelet (BubbleTea/Lipgloss/Bubbles) with SQLite, Cobra CLI.

**Module**: `github.com/laiambryant/tui-cardman`

### Layers

- **`cmd/command/`** — Cobra CLI commands (serve, migrate, import-full, import-updates, import-sets, list-sets, serve-ssh)
- **`internal/tui/`** — BubbleTea TUI. Main `Model` in `tui.go` owns all screens and services. Each screen is a separate view file (e.g., `card_game_tabs_view.go`, `lists_view.go`, `main_view.go`). Navigation via `Screen` enum, dispatched in `Update()`/`View()` switch statements.
- **`internal/services/`** — Business logic. Each domain has its own package with an interface + impl pattern (e.g., `CardService` interface, `CardServiceImpl` struct, `NewCardService()` constructor). SQL queries defined as package-level `const` blocks.
- **`internal/model/`** — Data structs (`models.go`): Card, CardGame, Set, UserCollection, UserList, UserListCard, ButtonConfiguration
- **`internal/db/`** — SQLite helpers (`helpers.go`: Query, ExecContext, WithTransaction) and migration system. Migrations in `db/migrations/` numbered 001-011.
- **`internal/runtimecfg/`** — Runtime config manager with observable pattern (Subscribe). Manages themes and keybindings. Theme files in `/themes/` (JSON).
- **`internal/auth/`** — Authentication with bcrypt
- **`internal/pokemontcg/`** — TCGDex API client wrapper for importing card data

### TUI Patterns

- **Screen flow**: `ScreenSplash → ScreenMain → ScreenCardGameMenu → ScreenCardGameTabs | ScreenLists`. Settings (F1) accessible globally.
- **Split layouts**: `lipgloss.JoinHorizontal()` with `RenderPanel()` for bordered panels. `RenderFramedWithModal()` for header/body/footer frames with modal overlay support.
- **Tables**: `NewStyledTable()` wrapper around Bubbles table component.
- **Quantity editing**: Temp changes stored in `tempQuantityChanges map[int64]int`, batch-saved via service with modal confirmation. Pattern used in both collection and lists views.
- **Styling**: All styles go through `StyleManager`. Use `sm.applyBGFG()` when creating custom `lipgloss.Style` to inherit theme background/foreground. Never use raw `lipgloss.NewStyle().Foreground(...)` without applying theme BG/FG — it causes black backgrounds in themed terminals.
- **Errors**: Each package has typed error structs in `errors.go` with `Error()` and `Unwrap()` methods.

### Service Pattern

```go
type FooService interface { ... }
type FooServiceImpl struct { db *sql.DB }
func NewFooService(db *sql.DB) FooService { return &FooServiceImpl{db: db} }
```

Services are created in `initServices()` in `tui.go` and injected into the Model.

## Code Style

- No empty lines or comments inside function bodies
- Imports: stdlib, third-party, internal — separated by blank lines. Local prefix: `github.com/laiambryant/tui-cardman`
- Formatting: `gofumpt` with extra rules enabled
- Linting: golangci-lint with sqlclosecheck, rowserrcheck, staticcheck, gofumpt, goimports enabled
- Logging: `slog` with `logging.SanitizeQuery()` for SQL. DB helpers in `internal/db/helpers.go` handle logging automatically.
