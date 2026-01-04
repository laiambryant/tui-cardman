package pokemontcg

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

const (
	createImportRunQuery = `INSERT INTO import_runs (import_type, status, started_at) 
	    VALUES (?, ?, ?)`

	updateImportRunQuery = `UPDATE import_runs 
	    SET status = ?, sets_processed = ?, cards_imported = ?, errors_count = ?, 
		   completed_at = ?, notes = ?
	    WHERE id = ?`

	selectSetIDQuery = `SELECT id FROM sets WHERE api_id = ?`

	insertSetQuery = `INSERT INTO sets (api_id, code, name, series, printed_total, total, 
					  release_date, symbol_url, logo_url, updated_at)
	    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	updateSetQuery = `UPDATE sets 
	    SET code = ?, name = ?, series = ?, printed_total = ?, total = ?, 
		   release_date = ?, symbol_url = ?, logo_url = ?, updated_at = ?
	    WHERE id = ?`

	selectCardIDQuery = `SELECT id FROM cards WHERE api_id = ?`

	insertCardQuery = `INSERT INTO cards (api_id, set_id, number, name, rarity, artist, updated_at)
		    VALUES (?, ?, ?, ?, ?, ?, ?)`

	updateCardQuery = `UPDATE cards 
		 SET set_id = ?, number = ?, name = ?, rarity = ?, 
			 artist = ?, updated_at = ?
		 WHERE id = ?`

	deleteCardImagesQuery       = `DELETE FROM card_images WHERE card_id = ?`
	deletePricesTCGQuery        = `DELETE FROM prices_tcgplayer WHERE card_id = ?`
	deletePricesCardMarketQuery = `DELETE FROM prices_cardmarket WHERE card_id = ?`

	insertCardImagesQuery = `INSERT INTO card_images (card_id, small_url, large_url, updated_at)
	    VALUES (?, ?, ?, ?)`

	insertPricesTCGQuery = `INSERT INTO prices_tcgplayer (card_id, price_type, low, mid, high, market, 
							 direct_low, tcgplayer_url, tcgplayer_updated_at, snapshot_at)
	    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	insertPricesCardMarketQuery = `INSERT INTO prices_cardmarket (card_id, avg_price, trend_price, url, snapshot_at)
	    VALUES (?, ?, ?, ?, ?)`

	selectAllSetAPIIDsQuery = `SELECT api_id FROM sets`
)

type ImportService struct {
	db     *sql.DB
	client *Client
	logger *slog.Logger
}

func NewImportService(db *sql.DB, client *Client, logger *slog.Logger) *ImportService {
	return &ImportService{
		db:     db,
		client: client,
		logger: logger,
	}
}

type ImportRun struct {
	ID            int64
	ImportType    string
	Status        string
	SetsProcessed int
	CardsImported int
	ErrorsCount   int
	StartedAt     time.Time
	CompletedAt   *time.Time
	Notes         string
}

func (s *ImportService) CreateImportRun(ctx context.Context, importType string) (int64, error) {
	result, err := s.db.ExecContext(ctx, createImportRunQuery, importType, "running", time.Now())
	if err != nil {
		return 0, fmt.Errorf("failed to create import run: %w", err)
	}
	return result.LastInsertId()
}

func (s *ImportService) UpdateImportRun(ctx context.Context, runID int64, status string, setsProcessed, cardsImported, errorsCount int, notes string) error {
	_, err := s.db.ExecContext(ctx, updateImportRunQuery, status, setsProcessed, cardsImported, errorsCount, time.Now(), notes, runID)
	if err != nil {
		return fmt.Errorf("failed to update import run: %w", err)
	}
	return nil
}

func (s *ImportService) UpsertSet(ctx context.Context, set Set) (int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	var setID int64
	err = tx.QueryRowContext(ctx, selectSetIDQuery, set.ID).Scan(&setID)
	if err == sql.ErrNoRows {
		result, err := tx.ExecContext(ctx, insertSetQuery,
			set.ID, set.PtcgoCode, set.Name, set.Series, set.PrintedTotal, set.Total,
			set.ReleaseDate, set.Images.Symbol, set.Images.Logo, time.Now())
		if err != nil {
			return 0, fmt.Errorf("failed to insert set: %w", err)
		}
		setID, err = result.LastInsertId()
		if err != nil {
			return 0, fmt.Errorf("failed to get last insert ID: %w", err)
		}
	} else if err != nil {
		return 0, fmt.Errorf("failed to query set: %w", err)
	} else {
		_, err = tx.ExecContext(ctx, updateSetQuery,
			set.PtcgoCode, set.Name, set.Series, set.PrintedTotal, set.Total,
			set.ReleaseDate, set.Images.Symbol, set.Images.Logo, time.Now(), setID)
		if err != nil {
			return 0, fmt.Errorf("failed to update set: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}
	return setID, nil
}

func (s *ImportService) UpsertCard(ctx context.Context, card Card, setID int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	cardID, err := s.upsertCardCore(ctx, tx, card, setID)
	if err != nil {
		return err
	}
	if err := s.replaceCardChildren(ctx, tx, cardID, card); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit card transaction: %w", err)
	}
	return nil
}

func (s *ImportService) upsertCardCore(ctx context.Context, tx *sql.Tx, card Card, setID int64) (int64, error) {
	var cardID int64
	err := tx.QueryRowContext(ctx, selectCardIDQuery, card.ID).Scan(&cardID)
	if err == sql.ErrNoRows {
		result, err := tx.ExecContext(ctx, insertCardQuery, card.ID, setID, card.Number, card.Name, card.Rarity, card.Artist, time.Now())
		if err != nil {
			return 0, fmt.Errorf("failed to insert card: %w", err)
		}
		cardID, err = result.LastInsertId()
		if err != nil {
			return 0, fmt.Errorf("failed to get card ID: %w", err)
		}
		return cardID, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to query card: %w", err)
	}

	if _, err := tx.ExecContext(ctx, updateCardQuery, setID, card.Number, card.Name, card.Rarity, card.Artist, time.Now(), cardID); err != nil {
		return 0, fmt.Errorf("failed to update card: %w", err)
	}
	return cardID, nil
}

func (s *ImportService) replaceCardChildren(ctx context.Context, tx *sql.Tx, cardID int64, card Card) error {
	if _, err := tx.ExecContext(ctx, deleteCardImagesQuery, cardID); err != nil {
		return fmt.Errorf("failed to delete old card images: %w", err)
	}
	if _, err := tx.ExecContext(ctx, deletePricesTCGQuery, cardID); err != nil {
		return fmt.Errorf("failed to delete old TCGPlayer prices: %w", err)
	}
	if _, err := tx.ExecContext(ctx, deletePricesCardMarketQuery, cardID); err != nil {
		return fmt.Errorf("failed to delete old CardMarket prices: %w", err)
	}

	if err := s.insertCardImagesTx(ctx, tx, cardID, card); err != nil {
		return err
	}
	if err := s.insertTCGPricesTx(ctx, tx, cardID, card); err != nil {
		return err
	}
	if err := s.insertCardMarketPricesTx(ctx, tx, cardID, card); err != nil {
		return err
	}
	return nil
}

func (s *ImportService) insertCardImagesTx(ctx context.Context, tx *sql.Tx, cardID int64, card Card) error {
	if card.Images.Small == "" && card.Images.Large == "" {
		return nil
	}
	if _, err := tx.ExecContext(ctx, insertCardImagesQuery, cardID, card.Images.Small, card.Images.Large, time.Now()); err != nil {
		return fmt.Errorf("failed to insert card images: %w", err)
	}
	return nil
}

func (s *ImportService) insertTCGPricesTx(ctx context.Context, tx *sql.Tx, cardID int64, card Card) error {
	if card.TCGPlayer == nil || card.TCGPlayer.Prices == nil {
		return nil
	}
	for priceType, price := range card.TCGPlayer.Prices {
		if _, err := tx.ExecContext(ctx, insertPricesTCGQuery, cardID, priceType, nullFloat64(price.Low), nullFloat64(price.Mid), nullFloat64(price.High), nullFloat64(price.Market), nullFloat64(price.DirectLow), card.TCGPlayer.URL, card.TCGPlayer.UpdatedAt, time.Now()); err != nil {
			return fmt.Errorf("failed to insert TCGPlayer price: %w", err)
		}
	}
	return nil
}

func (s *ImportService) insertCardMarketPricesTx(ctx context.Context, tx *sql.Tx, cardID int64, card Card) error {
	if card.CardMarket == nil || card.CardMarket.Prices == nil {
		return nil
	}
	for _, price := range card.CardMarket.Prices {
		if _, err := tx.ExecContext(ctx, insertPricesCardMarketQuery, cardID, nullFloat64(price.Avg), nullFloat64(price.Trend), card.CardMarket.URL, time.Now()); err != nil {
			return fmt.Errorf("failed to insert CardMarket price: %w", err)
		}
	}
	return nil
}

func (s *ImportService) GetExistingSetAPIIDs(ctx context.Context) (map[string]bool, error) {
	rows, err := s.db.QueryContext(ctx, selectAllSetAPIIDsQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query existing sets: %w", err)
	}
	defer rows.Close()
	existingSets := make(map[string]bool)
	for rows.Next() {
		var apiID string
		if err := rows.Scan(&apiID); err != nil {
			return nil, fmt.Errorf("failed to scan set api_id: %w", err)
		}
		existingSets[apiID] = true
	}
	return existingSets, rows.Err()
}

func (s *ImportService) ImportSet(ctx context.Context, set Set) (int, error) {
	s.logger.Info("Importing set", "set_id", set.ID, "name", set.Name)
	setID, err := s.UpsertSet(ctx, set)
	if err != nil {
		return 0, fmt.Errorf("failed to upsert set: %w", err)
	}
	cardsImported := 0
	page := 1
	for {
		s.logger.Debug("Fetching cards page", "set_id", set.ID, "page", page)
		paginatedResp, cards, err := s.client.GetCardsForSet(ctx, set.ID, page)
		if err != nil {
			return cardsImported, fmt.Errorf("failed to fetch cards for set %s page %d: %w", set.ID, page, err)
		}
		for _, card := range cards {
			if err := s.UpsertCard(ctx, card, setID); err != nil {
				s.logger.Error("Failed to upsert card", "card_id", card.ID, "error", err)
				continue
			}
			cardsImported++
		}
		s.logger.Info("Imported cards page", "set_id", set.ID, "page", page, "count", len(cards), "total_so_far", cardsImported)
		if page*paginatedResp.PageSize >= paginatedResp.TotalCount {
			break
		}
		page++
	}
	s.logger.Info("Completed set import", "set_id", set.ID, "total_cards", cardsImported)
	return cardsImported, nil
}

// ImportAllSets imports all sets (full import)
func (s *ImportService) ImportAllSets(ctx context.Context) error {
	runID, err := s.CreateImportRun(ctx, "import-full")
	if err != nil {
		return fmt.Errorf("failed to create import run: %w", err)
	}
	sets, err := s.client.GetSets(ctx)
	if err != nil {
		_ = s.UpdateImportRun(ctx, runID, "failed", 0, 0, 1, fmt.Sprintf("Failed to fetch sets: %v", err))
		return fmt.Errorf("failed to fetch sets: %w", err)
	}
	s.logger.Info("Starting full import", "total_sets", len(sets))
	totalCardsImported := 0
	setsProcessed := 0
	errorCount := 0
	var errorMessages []string
	for _, set := range sets {
		cardsImported, err := s.ImportSet(ctx, set)
		if err != nil {
			s.logger.Error("Failed to import set", "set_id", set.ID, "error", err)
			errorCount++
			errorMessages = append(errorMessages, fmt.Sprintf("Set %s: %v", set.ID, err))
			continue
		}
		totalCardsImported += cardsImported
		setsProcessed++
	}
	status := "completed"
	notes := fmt.Sprintf("Imported %d sets with %d total cards", setsProcessed, totalCardsImported)
	if errorCount > 0 {
		status = "completed_with_errors"
		notes += fmt.Sprintf(". Errors: %s", strings.Join(errorMessages, "; "))
	}
	if err := s.UpdateImportRun(ctx, runID, status, setsProcessed, totalCardsImported, errorCount, notes); err != nil {
		return fmt.Errorf("failed to update import run: %w", err)
	}
	s.logger.Info("Full import completed", "sets_processed", setsProcessed, "cards_imported", totalCardsImported, "errors", errorCount)
	return nil
}

// ImportNewSets imports only sets that don't exist in the database
func (s *ImportService) ImportNewSets(ctx context.Context) error {
	runID, err := s.CreateImportRun(ctx, "import-updates")
	if err != nil {
		return fmt.Errorf("failed to create import run: %w", err)
	}
	// Fetch all sets from API
	sets, err := s.client.GetSets(ctx)
	if err != nil {
		_ = s.UpdateImportRun(ctx, runID, "failed", 0, 0, 1, fmt.Sprintf("Failed to fetch sets: %v", err))
		return fmt.Errorf("failed to fetch sets: %w", err)
	}
	// Get existing set API IDs from database
	existingSets, err := s.GetExistingSetAPIIDs(ctx)
	if err != nil {
		_ = s.UpdateImportRun(ctx, runID, "failed", 0, 0, 1, fmt.Sprintf("Failed to query existing sets: %v", err))
		return fmt.Errorf("failed to query existing sets: %w", err)
	}
	// Filter to only new sets
	var newSets []Set
	for _, set := range sets {
		if !existingSets[set.ID] {
			newSets = append(newSets, set)
		}
	}
	if len(newSets) == 0 {
		s.logger.Info("No new sets to import")
		_ = s.UpdateImportRun(ctx, runID, "completed", 0, 0, 0, "No new sets found")
		return nil
	}
	s.logger.Info("Starting incremental import", "new_sets", len(newSets), "total_sets", len(sets))
	totalCardsImported := 0
	setsProcessed := 0
	errorCount := 0
	var errorMessages []string
	for _, set := range newSets {
		cardsImported, err := s.ImportSet(ctx, set)
		if err != nil {
			s.logger.Error("Failed to import set", "set_id", set.ID, "error", err)
			errorCount++
			errorMessages = append(errorMessages, fmt.Sprintf("Set %s: %v", set.ID, err))
			continue
		}
		totalCardsImported += cardsImported
		setsProcessed++
	}
	status := "completed"
	notes := fmt.Sprintf("Imported %d new sets with %d total cards", setsProcessed, totalCardsImported)
	if errorCount > 0 {
		status = "completed_with_errors"
		notes += fmt.Sprintf(". Errors: %s", strings.Join(errorMessages, "; "))
	}
	if err := s.UpdateImportRun(ctx, runID, status, setsProcessed, totalCardsImported, errorCount, notes); err != nil {
		return fmt.Errorf("failed to update import run: %w", err)
	}
	s.logger.Info("Incremental import completed", "sets_processed", setsProcessed, "cards_imported", totalCardsImported, "errors", errorCount)
	return nil
}

// Helper function to convert float64 to sql.NullFloat64
func nullFloat64(f float64) sql.NullFloat64 {
	if f == 0 {
		return sql.NullFloat64{Valid: false}
	}
	return sql.NullFloat64{Float64: f, Valid: true}
}
