package prices

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/laiambryant/tui-cardman/internal/db"
	"github.com/laiambryant/tui-cardman/internal/model"
)

// TCGPlayerPriceService defines the interface for TCGPlayer price-related operations
type TCGPlayerPriceService interface {
	DeletePrices(ctx context.Context, tx *sql.Tx, cardID int64) error
	InsertPrice(ctx context.Context, tx *sql.Tx, cardID int64, priceType string, low, mid, high, market, directLow float64, url, updatedAt string) error
	GetLatestPricesForCard(cardID int64) ([]model.TCGPlayerPriceRow, error)
}

// TCGPlayerPriceServiceImpl implements the TCGPlayerPriceService interface
type TCGPlayerPriceServiceImpl struct {
	db *sql.DB
}

// NewTCGPlayerPriceService creates a new instance of TCGPlayerPriceServiceImpl
func NewTCGPlayerPriceService(db *sql.DB) TCGPlayerPriceService {
	return &TCGPlayerPriceServiceImpl{db: db}
}

const (
	deletePricesTCGQuery = `DELETE FROM prices_tcgplayer WHERE card_id = ?`

	insertPricesTCGQuery = `INSERT INTO prices_tcgplayer (card_id, price_type, low, mid, high, market,
							 direct_low, tcgplayer_url, tcgplayer_updated_at, snapshot_at)
	    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	selectLatestTCGPricesQuery = `
		SELECT price_type, COALESCE(low, 0), COALESCE(mid, 0), COALESCE(high, 0),
		       COALESCE(market, 0), COALESCE(direct_low, 0), COALESCE(tcgplayer_url, ''), snapshot_at
		FROM prices_tcgplayer
		WHERE card_id = ?
		ORDER BY snapshot_at DESC
		LIMIT 10
	`
)

// DeletePrices deletes all TCGPlayer prices for a specific card within a transaction
func (s *TCGPlayerPriceServiceImpl) DeletePrices(ctx context.Context, tx *sql.Tx, cardID int64) error {
	if _, err := db.ExecContextTx(ctx, tx, deletePricesTCGQuery, cardID); err != nil {
		slog.Error("failed to delete TCGPlayer prices", "card_id", cardID, "error", err)
		return fmt.Errorf("failed to delete TCGPlayer prices for card %d: %w", cardID, err)
	}
	return nil
}

// InsertPrice inserts a TCGPlayer price record within a transaction
func (s *TCGPlayerPriceServiceImpl) InsertPrice(ctx context.Context, tx *sql.Tx, cardID int64, priceType string, low, mid, high, market, directLow float64, url, updatedAt string) error {
	if _, err := db.ExecContextTx(ctx, tx, insertPricesTCGQuery, cardID, priceType,
		nullFloat64(low), nullFloat64(mid), nullFloat64(high),
		nullFloat64(market), nullFloat64(directLow),
		url, updatedAt, time.Now()); err != nil {
		slog.Error("failed to insert TCGPlayer price", "card_id", cardID, "price_type", priceType, "error", err)
		return fmt.Errorf("failed to insert TCGPlayer price for card %d: %w", cardID, err)
	}
	return nil
}

func (s *TCGPlayerPriceServiceImpl) GetLatestPricesForCard(cardID int64) ([]model.TCGPlayerPriceRow, error) {
	rows, err := db.Query(s.db, selectLatestTCGPricesQuery, cardID)
	if err != nil {
		return nil, fmt.Errorf("failed to query TCGPlayer prices for card %d: %w", cardID, err)
	}
	defer rows.Close()
	var prices []model.TCGPlayerPriceRow
	for rows.Next() {
		var p model.TCGPlayerPriceRow
		if err := rows.Scan(&p.PriceType, &p.Low, &p.Mid, &p.High, &p.Market, &p.DirectLow, &p.URL, &p.SnapshotAt); err != nil {
			slog.Error("failed to scan TCGPlayer price row", "error", err)
			continue
		}
		prices = append(prices, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to query TCGPlayer prices for card %d: %w", cardID, err)
	}
	return prices, nil
}
