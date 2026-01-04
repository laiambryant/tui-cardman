# tui-cardman

A TUI to manage your trading card inventory

## Build

This project uses [go-sqlite3](https://github.com/mattn/go-sqlite3), which is a CGO package. Building requires:

1. A C compiler (GCC or similar)
2. The `CGO_ENABLED=1` environment variable
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

### Pokemon TCG API Key

Get your free API key from [Pokemon TCG Developer Portal](https://dev.pokemontcg.io/):
- **Without API key**: 1,000 requests/day, 30/minute
- **With API key**: 20,000 requests/day (default)

## Database Setup

Run migrations to set up the database schema:

```bash
./cardman migrate
```

## Importing Pokemon TCG Data

The application provides two commands for importing card data from the Pokemon TCG API:

### Full Import

Import all Pokemon TCG sets and cards (initial setup):

```bash
./cardman import-full
```

This will:
- Fetch all sets from the Pokemon TCG API
- Import all cards from each set with complete metadata
- Store card images, prices, and other related data
- Track import progress in the database

**Note**: This can take considerable time and API quota. Recommended for initial setup only.

### Incremental Import

Import only new sets that don't exist in your database:

```bash
./cardman import-updates
```

This will:
- Fetch all sets from the API
- Compare with your local database
- Import only net-new sets
- Skip all existing sets (saves time and API quota)

**Recommended**: Run this daily or weekly via cron to catch new set releases.

### Import Data Stored

The import process stores:
- **Sets**: All Pokemon TCG sets with metadata
- **Cards**: Core card data (name, number, supertype, rarity, HP, etc.)
- **Images**: Small and large image URLs
- **Prices**: TCGPlayer and CardMarket price snapshots
- **Import Runs**: History and status of all import operations

## Available Commands

```bash
./cardman migrate          # Run database migrations
./cardman import-full      # Import all Pokemon TCG data
./cardman import-updates   # Import only new sets
./cardman serve            # Start the TUI server
./cardman serve-ssh        # Start the SSH server
```