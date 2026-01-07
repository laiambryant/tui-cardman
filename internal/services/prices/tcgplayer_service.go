package prices

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// TCGPlayerPriceService defines the interface for TCGPlayer price-related operations
type TCGPlayerPriceService interface {
	DeletePrices(ctx context.Context, tx *sql.Tx, cardID int64) error
	InsertPrice(ctx context.Context, tx *sql.Tx, cardID int64, priceType string, low, mid, high, market, directLow float64, url, updatedAt string) error
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
)

// DeletePrices deletes all TCGPlayer prices for a specific card within a transaction
func (s *TCGPlayerPriceServiceImpl) DeletePrices(ctx context.Context, tx *sql.Tx, cardID int64) error {
	if _, err := tx.ExecContext(ctx, deletePricesTCGQuery, cardID); err != nil {
		return fmt.Errorf("failed to delete TCGPlayer prices: %w", err)
	}
	return nil
}

// InsertPrice inserts a TCGPlayer price record within a transaction
func (s *TCGPlayerPriceServiceImpl) InsertPrice(ctx context.Context, tx *sql.Tx, cardID int64, priceType string, low, mid, high, market, directLow float64, url, updatedAt string) error {
	if _, err := tx.ExecContext(ctx, insertPricesTCGQuery, cardID, priceType,
		nullFloat64(low), nullFloat64(mid), nullFloat64(high),
		nullFloat64(market), nullFloat64(directLow),
		url, updatedAt, time.Now()); err != nil {
		return fmt.Errorf("failed to insert TCGPlayer price: %w", err)
	}
	return nil
}
