package prices

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"github.com/laiambryant/tui-cardman/internal/db"
	"github.com/laiambryant/tui-cardman/internal/model"
)

// CardMarketPriceService defines the interface for CardMarket price-related operations
type CardMarketPriceService interface {
	DeletePrices(ctx context.Context, tx *sql.Tx, cardID int64) error
	InsertPrice(ctx context.Context, tx *sql.Tx, cardID int64, avgPrice, trendPrice float64, url string) error
	GetLatestPriceForCard(cardID int64) (*model.CardMarketPriceRow, error)
}

// CardMarketPriceServiceImpl implements the CardMarketPriceService interface
type CardMarketPriceServiceImpl struct {
	db *sql.DB
}

// NewCardMarketPriceService creates a new instance of CardMarketPriceServiceImpl
func NewCardMarketPriceService(db *sql.DB) CardMarketPriceService {
	return &CardMarketPriceServiceImpl{db: db}
}

const (
	deletePricesCardMarketQuery = `DELETE FROM prices_cardmarket WHERE card_id = ?`

	insertPricesCardMarketQuery = `INSERT INTO prices_cardmarket (card_id, avg_price, trend_price, url, snapshot_at)
	    VALUES (?, ?, ?, ?, ?)`

	selectLatestCardMarketPriceQuery = `
		SELECT COALESCE(avg_price, 0), COALESCE(trend_price, 0), COALESCE(url, ''), snapshot_at
		FROM prices_cardmarket
		WHERE card_id = ?
		ORDER BY snapshot_at DESC
		LIMIT 1
	`
)

// DeletePrices deletes all CardMarket prices for a specific card within a transaction
func (s *CardMarketPriceServiceImpl) DeletePrices(ctx context.Context, tx *sql.Tx, cardID int64) error {
	if _, err := db.ExecContextTx(ctx, tx, deletePricesCardMarketQuery, cardID); err != nil {
		slog.Error("failed to delete CardMarket prices", "card_id", cardID, "error", err)
		return &FailedToDeleteCardMarketPricesError{Err: err}
	}
	return nil
}

// InsertPrice inserts a CardMarket price record within a transaction
func (s *CardMarketPriceServiceImpl) InsertPrice(ctx context.Context, tx *sql.Tx, cardID int64, avgPrice, trendPrice float64, url string) error {
	if _, err := db.ExecContextTx(ctx, tx, insertPricesCardMarketQuery, cardID,
		nullFloat64(avgPrice), nullFloat64(trendPrice), url, time.Now()); err != nil {
		slog.Error("failed to insert CardMarket price", "card_id", cardID, "error", err)
		return &FailedToInsertCardMarketPriceError{Err: err}
	}
	return nil
}

func (s *CardMarketPriceServiceImpl) GetLatestPriceForCard(cardID int64) (*model.CardMarketPriceRow, error) {
	var p model.CardMarketPriceRow
	err := db.QueryRow(s.db, selectLatestCardMarketPriceQuery, cardID).Scan(&p.AvgPrice, &p.TrendPrice, &p.URL, &p.SnapshotAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, &FailedToQueryCardMarketPricesError{Err: err}
	}
	return &p, nil
}
