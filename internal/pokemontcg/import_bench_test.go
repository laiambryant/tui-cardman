package pokemontcg

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"testing"

	card "github.com/laiambryant/tui-cardman/internal/services/cards"
	"github.com/laiambryant/tui-cardman/internal/services/importruns"
	"github.com/laiambryant/tui-cardman/internal/services/prices"
	"github.com/laiambryant/tui-cardman/internal/services/sets"
	_ "github.com/mattn/go-sqlite3"
)

// setupBenchmarkDB creates an in-memory SQLite database with required schema
func setupBenchmarkDB(b *testing.B) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		b.Fatalf("failed to open database: %v", err)
	}

	// Create required tables
	schema := `
	CREATE TABLE card_games (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE sets (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		api_id TEXT NOT NULL UNIQUE,
		code TEXT,
		name TEXT NOT NULL,
		printed_total INTEGER,
		total INTEGER,
		symbol_url TEXT,
		logo_url TEXT,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE cards (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		api_id TEXT NOT NULL UNIQUE,
		set_id INTEGER NOT NULL,
		number TEXT NOT NULL,
		name TEXT NOT NULL,
		rarity TEXT,
		artist TEXT,
		card_game_id INTEGER NOT NULL,
		is_placeholder INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (set_id) REFERENCES sets(id),
		FOREIGN KEY (card_game_id) REFERENCES card_games(id)
	);

	CREATE TABLE prices_tcgplayer (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		card_id INTEGER NOT NULL REFERENCES cards(id) ON DELETE CASCADE,
		price_type TEXT NOT NULL,
		low REAL,
		mid REAL,
		high REAL,
		market REAL,
		direct_low REAL,
		tcgplayer_url TEXT,
		tcgplayer_updated_at TEXT,
		snapshot_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE prices_cardmarket (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		card_id INTEGER NOT NULL REFERENCES cards(id) ON DELETE CASCADE,
		avg_price REAL,
		trend_price REAL,
		url TEXT,
		snapshot_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE import_runs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		import_type TEXT NOT NULL,
		status TEXT NOT NULL,
		sets_processed INTEGER NOT NULL DEFAULT 0,
		cards_imported INTEGER NOT NULL DEFAULT 0,
		errors_count INTEGER NOT NULL DEFAULT 0,
		started_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		completed_at DATETIME,
		notes TEXT
	);

	INSERT INTO card_games (name) VALUES ('Pokemon');
	`

	if _, err := db.Exec(schema); err != nil {
		b.Fatalf("failed to create schema: %v", err)
	}

	return db
}

// generateMockCards creates a set of mock cards for testing
func generateMockCards(count int, setID string) []Card {
	cards := make([]Card, count)
	for i := range count {
		cards[i] = Card{
			ID:     fmt.Sprintf("%s-%d", setID, i+1),
			Name:   fmt.Sprintf("Test Card %d", i+1),
			Number: fmt.Sprintf("%d", i+1),
			Artist: "Test Artist",
			Rarity: "Common",
			TCGPlayer: &TCGPlayerPrices{
				URL:       "https://example.com",
				UpdatedAt: "2024-01-01",
				Prices: map[string]TCGPlayerPrice{
					"normal": {
						Low:    1.0,
						Mid:    2.0,
						High:   3.0,
						Market: 2.5,
					},
				},
			},
			CardMarket: &CardMarketPrices{
				URL:       "https://example.com",
				UpdatedAt: "2024-01-01",
				Prices: map[string]CardMarketPrice{
					"averageSellPrice": {
						Avg:   2.0,
						Trend: 2.1,
					},
				},
			},
		}
	}
	return cards
}

// importCardsPerCardTransaction simulates the old approach (one transaction per card)
func importCardsPerCardTransaction(ctx context.Context, s *ImportService, cards []Card, setID int64) error {
	for _, card := range cards {
		if err := s.UpsertCard(ctx, card, setID); err != nil {
			return err
		}
	}
	return nil
}

// importCardsPerSetTransaction simulates the new approach (one transaction per set)
func importCardsPerSetTransaction(ctx context.Context, s *ImportService, cards []Card, setID int64) (err error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		err = tx.Rollback()
	}()
	for _, card := range cards {
		if err := s.upsertCardTx(ctx, tx, card, setID); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return err
}

// BenchmarkImportPerCardTransaction benchmarks the old approach
func BenchmarkImportPerCardTransaction(b *testing.B) {
	cardCounts := []int{10, 50, 100, 200}

	for _, count := range cardCounts {
		b.Run(fmt.Sprintf("Cards_%d", count), func(b *testing.B) {
			db := setupBenchmarkDB(b)
			defer db.Close()

			logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

			setService := sets.NewSetService(db)
			cardService := card.NewCardService(db)
			tcgPlayerService := prices.NewTCGPlayerPriceService(db)
			cardMarketService := prices.NewCardMarketPriceService(db)
			importRunService := importruns.NewImportRunService(db)

			service := &ImportService{
				db:                     db,
				logger:                 logger,
				setService:             setService,
				cardService:            cardService,
				tcgPlayerPriceService:  tcgPlayerService,
				cardMarketPriceService: cardMarketService,
				importRunService:       importRunService,
				pokemonGameID:          1,
			}

			ctx := context.Background()
			mockCards := generateMockCards(count, "test-set")

			b.ResetTimer()
			for i := 0; b.Loop(); i++ {
				b.StopTimer()
				// Create a new set for each iteration
				setID, err := setService.UpsertSet(ctx, fmt.Sprintf("test-set-%d", i), "TST", fmt.Sprintf("Test Set %d", i), count, count)
				if err != nil {
					b.Fatalf("failed to create set: %v", err)
				}
				b.StartTimer()

				if err := importCardsPerCardTransaction(ctx, service, mockCards, setID); err != nil {
					b.Fatalf("failed to import cards: %v", err)
				}
			}
		})
	}
}

// BenchmarkImportPerSetTransaction benchmarks the new approach
func BenchmarkImportPerSetTransaction(b *testing.B) {
	cardCounts := []int{10, 50, 100, 200}

	for _, count := range cardCounts {
		b.Run(fmt.Sprintf("Cards_%d", count), func(b *testing.B) {
			db := setupBenchmarkDB(b)
			defer db.Close()

			logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

			setService := sets.NewSetService(db)
			cardService := card.NewCardService(db)
			tcgPlayerService := prices.NewTCGPlayerPriceService(db)
			cardMarketService := prices.NewCardMarketPriceService(db)
			importRunService := importruns.NewImportRunService(db)

			service := &ImportService{
				db:                     db,
				logger:                 logger,
				setService:             setService,
				cardService:            cardService,
				tcgPlayerPriceService:  tcgPlayerService,
				cardMarketPriceService: cardMarketService,
				importRunService:       importRunService,
				pokemonGameID:          1,
			}

			ctx := context.Background()
			mockCards := generateMockCards(count, "test-set")

			b.ResetTimer()
			for i := 0; b.Loop(); i++ {
				b.StopTimer()
				// Create a new set for each iteration
				setID, err := setService.UpsertSet(ctx, fmt.Sprintf("test-set-%d", i), "TST", fmt.Sprintf("Test Set %d", i), count, count)
				if err != nil {
					b.Fatalf("failed to create set: %v", err)
				}
				b.StartTimer()

				if err := importCardsPerSetTransaction(ctx, service, mockCards, setID); err != nil {
					b.Fatalf("failed to import cards: %v", err)
				}
			}
		})
	}
}
