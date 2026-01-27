# tui-cardman

A TUI to manage your trading card inventory

## Build

This project uses [go-sqlite3](https://github.com/mattn/go-sqlite3), which is a CGO package. Building requires:

1. A C compiler (GCC or similar)
2. The `CGO_ENABLED=1` environment variable

### Using Make (Recommended)

```bash
make build
```

### Manual Build

```bash
CGO_ENABLED=1 go build -o cardman ./cmd/main.go
```

## Configuration

Create a `.env` file in the project root (or copy from `.env.example`):

```bash
cp .env.example .env
```

Edit `.env` to configure:

- `LOG_LEVEL` - Logging verbosity (DEBUG, INFO, WARN, ERROR)
- `DB_DSN` - Database connection string
- `SSH_PORT` - SSH server port
- `SSH_HOST_KEY` - SSH host key path
- `API_KEY` - Pokemon TCG API key (optional, but recommended)

### TCGDex API Key

Get your API key or use the public TCGDex endpoints at <https://tcgdex.dev/>:

- **Without API key**: limited public access depending on TCGDex rate limits
- **With API key**: pass the token via `API_KEY` in your `.env` and the application will send it as `X-Api-Key`.

## Importing Pokemon TCG Data (via TCGDex)

## Database Setup

Run migrations to set up the database schema:

```bash
make migrate
# or manually: ./cardman migrate
```

## Importing Pokemon TCG Data

The application provides commands for importing card data from the TCGDex API (Pokemon TCG data served by TCGDex):

### Full Import

Import all Pokemon TCG sets and cards (initial setup) via TCGDex:

```bash
make import-full
# or manually: ./cardman import-full
```

This will:

- Fetch all sets from the Pokemon TCG API
- Import all cards from each set with complete metadata
- Store card images, prices, and other related data
- Track import progress in the database

**Note**: This can take considerable time and API quota. Recommended for initial setup only.

### Incremental Import

Import only new sets that don't exist in your database (via TCGDex):

```bash
make import-updates
# or manually: ./cardman import-updates
```

This will:

- Fetch all sets from the API
- Compare with your local database
- Import only net-new sets
- Skip all existing sets (saves time and API quota)

**Recommended**: Run this daily or weekly via cron to catch new set releases.

### Import Specific Sets

Import one or more specific sets by their set IDs (useful to import only chosen sets). This uses the TCGDex `sets` endpoint under the hood:

```bash
./cardman import-sets base1
./cardman import-sets base1 jungle fossil
```

This command fetches the specified sets from TCGDex and imports all cards for each set. Use `list-sets` to discover set IDs.

### Import Data Stored

The import process stores:

- **Sets**: All Pokemon TCG sets with metadata
- **Cards**: Core card data (name, number, supertype, rarity, HP, etc.)
- **Images**: Small and large image URLs
- **Prices**: TCGPlayer and CardMarket price snapshots
- **Import Runs**: History and status of all import operations

## Available Commands

### Using Make

```bash
make help              # Show all available commands
make build             # Build the application
make test              # Run all tests
make test-coverage     # Run tests with coverage
make test-bench        # Run benchmarks
make fmt               # Format code
make vet               # Run go vet
make lint              # Run golangci-lint
make lint-install      # Install golangci-lint
make tidy              # Tidy dependencies
make check             # Run fmt, vet, and test
make all               # Full build pipeline (tidy, check, build)
make migrate           # Run database migrations
make serve             # Start the TUI server
make serve-ssh         # Start the SSH server
make import-full       # Import all Pokemon TCG data
make import-updates    # Import only new sets
make list-sets         # List available sets from the API
make clean             # Remove build artifacts
make install           # Install binary to GOPATH/bin
make run               # Build and run (serve)
```

### Manual Commands

```bash
./cardman migrate          # Run database migrations
./cardman import-full      # Import all Pokemon TCG data
./cardman import-updates   # Import only new sets
./cardman import-sets      # Import one or more specific sets by ID
./cardman list-sets        # List available sets from the API
./cardman serve            # Start the TUI server
./cardman serve-ssh        # Start the SSH server
```
