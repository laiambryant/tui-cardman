package pokemontcg

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/laiambryant/tui-cardman/internal/logging"
	"github.com/laiambryant/tui-cardman/internal/model"
	card "github.com/laiambryant/tui-cardman/internal/services/cards"
	"github.com/laiambryant/tui-cardman/internal/services/importruns"
	"github.com/laiambryant/tui-cardman/internal/services/pokemoncard"
	"github.com/laiambryant/tui-cardman/internal/services/prices"
	"github.com/laiambryant/tui-cardman/internal/services/sets"
)

type ImportService struct {
	db                     *sql.DB
	client                 *Client
	logger                 *slog.Logger
	importRunService       importruns.ImportRunService
	setService             sets.SetService
	cardService            card.CardService
	tcgPlayerPriceService  prices.TCGPlayerPriceService
	cardMarketPriceService prices.CardMarketPriceService
	pokemonCardService     pokemoncard.PokemonCardService
	pokemonGameID          int64
}

func NewImportService(
	db *sql.DB,
	client *Client,
	logger *slog.Logger,
	importRunService importruns.ImportRunService,
	setService sets.SetService,
	cardService card.CardService,
	tcgPlayerPriceService prices.TCGPlayerPriceService,
	cardMarketPriceService prices.CardMarketPriceService,
	pokemonCardService pokemoncard.PokemonCardService,
) *ImportService {
	service := &ImportService{
		db:                     db,
		client:                 client,
		logger:                 logger,
		importRunService:       importRunService,
		setService:             setService,
		cardService:            cardService,
		tcgPlayerPriceService:  tcgPlayerPriceService,
		cardMarketPriceService: cardMarketPriceService,
		pokemonCardService:     pokemonCardService,
	}

	// Fetch Pokemon card game ID
	if err := service.initPokemonGameID(context.Background()); err != nil {
		logger.Error("Failed to initialize Pokemon game ID", "error", err)
	}

	return service
}

func (s *ImportService) initPokemonGameID(ctx context.Context) error {
	query := "SELECT id FROM card_games WHERE name = ?"
	slog.Debug("query row", "query", logging.SanitizeQuery(query), "args", []any{"Pokemon"})
	var gameID int64
	err := s.db.QueryRowContext(ctx, query, "Pokemon").Scan(&gameID)
	if err != nil {
		return fmt.Errorf("failed to get pokemon card game id: %w", err)
	}
	s.pokemonGameID = gameID
	s.logger.Debug("Initialized Pokemon card game ID", "id", gameID)
	return nil
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
	return s.importRunService.CreateImportRun(ctx, importType)
}

func (s *ImportService) UpdateImportRun(ctx context.Context, runID int64, status string, setsProcessed, cardsImported, errorsCount int, notes string) error {
	return s.importRunService.UpdateImportRun(ctx, runID, status, setsProcessed, cardsImported, errorsCount, notes)
}

func (s *ImportService) UpsertSet(ctx context.Context, set Set) (int64, error) {
	return s.setService.UpsertSet(ctx, set.ID, set.PtcgoCode, set.Name,
		set.PrintedTotal, set.Total)
}

func (s *ImportService) UpsertCard(ctx context.Context, card Card, setID int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && err == nil {
			err = rollbackErr
		}
	}()
	if err := s.upsertCardTx(ctx, tx, card, setID); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit card transaction: %w", err)
	}
	return nil
}

// upsertCardTx upserts a card using the provided transaction
func (s *ImportService) upsertCardTx(ctx context.Context, tx *sql.Tx, card Card, setID int64) error {
	cardID, err := s.cardService.UpsertCard(ctx, tx, card.ID, setID, card.Number, card.Name, card.Rarity, card.Artist, s.pokemonGameID)
	if err != nil {
		return err
	}
	if err := s.replaceCardChildren(ctx, tx, cardID, card); err != nil {
		return err
	}
	if err := s.upsertPokemonCardTx(ctx, tx, cardID, card); err != nil {
		return err
	}
	return nil
}

func (s *ImportService) upsertPokemonCardTx(ctx context.Context, tx *sql.Tx, cardID int64, card Card) error {
	if s.pokemonCardService == nil {
		return nil
	}
	pc := &model.PokemonCard{
		CardID:         cardID,
		HP:             card.HP,
		Retreat:        card.Retreat,
		Category:       card.Category,
		Stage:          card.Stage,
		EvolveFrom:     card.EvolveFrom,
		Description:    card.Description,
		Level:          card.Level,
		RegulationMark: card.RegulationMark,
		LegalStandard:  card.LegalStandard,
		LegalExpanded:  card.LegalExpanded,
		Types:          card.Types,
	}
	for _, a := range card.Attacks {
		pc.Attacks = append(pc.Attacks, model.PokemonCardAttack{
			Name: a.Name, Cost: a.Cost, Effect: a.Effect, Damage: a.Damage,
		})
	}
	for _, a := range card.Abilities {
		pc.Abilities = append(pc.Abilities, model.PokemonCardAbility{
			Type: a.Type, Name: a.Name, Effect: a.Effect,
		})
	}
	for _, w := range card.Weaknesses {
		pc.Weaknesses = append(pc.Weaknesses, model.PokemonCardWeakRes{
			Type: w.Type, Value: w.Value,
		})
	}
	for _, r := range card.Resistances {
		pc.Resistances = append(pc.Resistances, model.PokemonCardWeakRes{
			Type: r.Type, Value: r.Value,
		})
	}
	return s.pokemonCardService.UpsertPokemonCard(ctx, tx, cardID, pc)
}

func (s *ImportService) replaceCardChildren(ctx context.Context, tx *sql.Tx, cardID int64, card Card) error {
	if err := s.tcgPlayerPriceService.DeletePrices(ctx, tx, cardID); err != nil {
		return err
	}
	if err := s.insertTCGPricesTx(ctx, tx, cardID, card); err != nil {
		return err
	}
	if err := s.cardMarketPriceService.DeletePrices(ctx, tx, cardID); err != nil {
		return err
	}
	if err := s.insertCardMarketPricesTx(ctx, tx, cardID, card); err != nil {
		return err
	}
	return nil
}

func (s *ImportService) insertTCGPricesTx(ctx context.Context, tx *sql.Tx, cardID int64, card Card) error {
	if card.TCGPlayer == nil || card.TCGPlayer.Prices == nil {
		return nil
	}
	for priceType, price := range card.TCGPlayer.Prices {
		if err := s.tcgPlayerPriceService.InsertPrice(ctx, tx, cardID, priceType,
			price.Low, price.Mid, price.High, price.Market, price.DirectLow,
			card.TCGPlayer.URL, card.TCGPlayer.UpdatedAt); err != nil {
			return err
		}
	}
	return nil
}

func (s *ImportService) insertCardMarketPricesTx(ctx context.Context, tx *sql.Tx, cardID int64, card Card) error {
	if card.CardMarket == nil || card.CardMarket.Prices == nil {
		return nil
	}
	for _, price := range card.CardMarket.Prices {
		if err := s.cardMarketPriceService.InsertPrice(ctx, tx, cardID,
			price.Avg, price.Trend, card.CardMarket.URL); err != nil {
			return err
		}
	}
	return nil
}

func (s *ImportService) GetExistingSetAPIIDs(ctx context.Context) (map[string]bool, error) {
	apiIDs, err := s.setService.GetAllSetAPIIDs(ctx)
	if err != nil {
		return nil, err
	}
	existingSets := make(map[string]bool)
	for _, apiID := range apiIDs {
		existingSets[apiID] = true
	}
	return existingSets, nil
}

type importResult struct {
	setsProcessed      int
	totalCardsImported int
	errorCount         int
	errorMessages      []string
}

func (s *ImportService) ImportSet(ctx context.Context, set Set) (int, error) {
	s.logger.Info("Importing set", "set_id", set.ID, "name", set.Name)
	setID, err := s.UpsertSet(ctx, set)
	if err != nil {
		return 0, fmt.Errorf("failed to upsert set %s: %w", set.ID, err)
	}

	// Create a single transaction for all cards in this set
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && err == nil {
			err = rollbackErr
		}
	}()

	cardsImported := 0
	page := 1
	for {
		s.logger.Debug("Fetching cards page", "set_id", set.ID, "page", page)
		paginatedResp, cards, err := s.client.GetCardsForSet(ctx, set.ID, page)
		if err != nil {
			return cardsImported, fmt.Errorf("failed to fetch cards for set %s page %d: %w", set.ID, page, err)
		}
		for _, card := range cards {
			if err := s.upsertCardTx(ctx, tx, card, setID); err != nil {
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

	// Commit the transaction once for all cards in the set
	if err := tx.Commit(); err != nil {
		s.logger.Error("Failed to commit set transaction", "set_id", set.ID, "error", err)
		return cardsImported, fmt.Errorf("failed to commit set transaction: %w", err)
	}

	s.logger.Info("Completed set import", "set_id", set.ID, "total_cards", cardsImported)
	return cardsImported, nil
}

func (s *ImportService) processSets(ctx context.Context, sets []Set) *importResult {
	result := &importResult{}
	for _, set := range sets {
		cardsImported, err := s.ImportSet(ctx, set)
		if err != nil {
			s.logger.Error("Failed to import set", "set_id", set.ID, "error", err)
			result.errorCount++
			result.errorMessages = append(result.errorMessages, fmt.Sprintf("Set %s: %v", set.ID, err))
			continue
		}
		result.totalCardsImported += cardsImported
		result.setsProcessed++
	}
	return result
}

func (s *ImportService) buildImportNotes(setsProcessed, totalCards, errorCount int, extraNotes ...string) (string, string) {
	status := "completed"
	notes := fmt.Sprintf("Imported %d sets with %d total cards", setsProcessed, totalCards)
	for _, extra := range extraNotes {
		if extra != "" {
			notes += ". " + extra
		}
	}
	if errorCount > 0 {
		status = "completed_with_errors"
	}
	return status, notes
}

func (s *ImportService) completeImportRun(ctx context.Context, runID int64, result *importResult, extraNotes ...string) error {
	status, notes := s.buildImportNotes(result.setsProcessed, result.totalCardsImported, result.errorCount, extraNotes...)
	if result.errorCount > 0 {
		notes += fmt.Sprintf(". Errors: %s", strings.Join(result.errorMessages, "; "))
	}
	return s.UpdateImportRun(ctx, runID, status, result.setsProcessed, result.totalCardsImported, result.errorCount, notes)
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
	result := s.processSets(ctx, sets)
	if err := s.completeImportRun(ctx, runID, result); err != nil {
		return fmt.Errorf("failed to update import run: %w", err)
	}

	s.logger.Info("Full import completed", "sets_processed", result.setsProcessed, "cards_imported", result.totalCardsImported, "errors", result.errorCount)
	return nil
}

// ImportNewSets imports only sets that don't exist in the database
func (s *ImportService) ImportNewSets(ctx context.Context) error {
	runID, err := s.CreateImportRun(ctx, "import-updates")
	if err != nil {
		return fmt.Errorf("failed to create import run: %w", err)
	}
	sets, err := s.client.GetSets(ctx)
	if err != nil {
		_ = s.UpdateImportRun(ctx, runID, "failed", 0, 0, 1, fmt.Sprintf("Failed to fetch sets: %v", err))
		return fmt.Errorf("failed to fetch sets: %w", err)
	}
	newSets, err := s.filterNewSets(ctx, sets)
	if err != nil {
		_ = s.UpdateImportRun(ctx, runID, "failed", 0, 0, 1, fmt.Sprintf("Failed to query existing sets: %v", err))
		return fmt.Errorf("failed to query existing sets: %w", err)
	}
	if len(newSets) == 0 {
		s.logger.Info("No new sets to import")
		_ = s.UpdateImportRun(ctx, runID, "completed", 0, 0, 0, "No new sets found")
		return nil
	}
	s.logger.Info("Starting incremental import", "new_sets", len(newSets), "total_sets", len(sets))
	result := s.processSets(ctx, newSets)
	if err := s.completeImportRun(ctx, runID, result); err != nil {
		return fmt.Errorf("failed to update import run: %w", err)
	}
	s.logger.Info("Incremental import completed", "sets_processed", result.setsProcessed, "cards_imported", result.totalCardsImported, "errors", result.errorCount)
	return nil
}

func (s *ImportService) filterNewSets(ctx context.Context, sets []Set) ([]Set, error) {
	existingSets, err := s.GetExistingSetAPIIDs(ctx)
	if err != nil {
		return nil, err
	}
	var newSets []Set
	for _, set := range sets {
		if !existingSets[set.ID] {
			newSets = append(newSets, set)
		}
	}
	return newSets, nil
}

// ImportSpecificSets imports only the specified sets by their IDs
func (s *ImportService) ImportSpecificSets(ctx context.Context, setIDs []string) error {
	runID, err := s.CreateImportRun(ctx, "import-specific")
	if err != nil {
		return fmt.Errorf("failed to create import run: %w", err)
	}
	allSets, err := s.client.GetSets(ctx)
	if err != nil {
		_ = s.UpdateImportRun(ctx, runID, "failed", 0, 0, 1, fmt.Sprintf("Failed to fetch sets: %v", err))
		return fmt.Errorf("failed to fetch sets: %w", err)
	}
	setsToImport, notFound := s.findRequestedSets(allSets, setIDs)
	if len(notFound) > 0 {
		s.logger.Warn("Some sets were not found", "not_found", strings.Join(notFound, ", "))
	}
	if len(setsToImport) == 0 {
		msg := fmt.Sprintf("None of the specified sets were found: %s", strings.Join(setIDs, ", "))
		_ = s.UpdateImportRun(ctx, runID, "failed", 0, 0, 1, msg)
		return fmt.Errorf("%s", msg)
	}
	s.logger.Info("Starting import of specific sets", "sets_to_import", len(setsToImport), "requested", len(setIDs))
	result := s.processSets(ctx, setsToImport)
	var extraNote string
	if len(notFound) > 0 {
		extraNote = fmt.Sprintf("Not found: %s", strings.Join(notFound, ", "))
	}
	if err := s.completeImportRun(ctx, runID, result, extraNote); err != nil {
		return fmt.Errorf("failed to update import run: %w", err)
	}
	s.logger.Info("Specific sets import completed", "sets_processed", result.setsProcessed, "cards_imported", result.totalCardsImported, "errors", result.errorCount)
	return nil
}

func (s *ImportService) findRequestedSets(allSets []Set, setIDs []string) ([]Set, []string) {
	setMap := make(map[string]Set)
	for _, set := range allSets {
		setMap[set.ID] = set
	}
	var setsToImport []Set
	var notFound []string
	for _, setID := range setIDs {
		if set, exists := setMap[setID]; exists {
			setsToImport = append(setsToImport, set)
		} else {
			notFound = append(notFound, setID)
		}
	}
	return setsToImport, notFound
}

func (s *ImportService) DeleteSetByAPIID(ctx context.Context, setAPIID string) error {
	dbSetID, err := s.setService.GetSetIDByAPIID(ctx, setAPIID)
	if err == sql.ErrNoRows {
		s.logger.Debug("set not found in database, nothing to delete", "api_id", setAPIID)
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to delete set %s: %w", setAPIID, err)
	}
	deleteCardsQuery := "DELETE FROM cards WHERE set_id = ?"
	slog.Debug("exec", "query", logging.SanitizeQuery(deleteCardsQuery), "args", []any{dbSetID})
	_, err = s.db.ExecContext(ctx, deleteCardsQuery, dbSetID)
	if err != nil {
		return fmt.Errorf("failed to delete cards for set %s: %w", setAPIID, err)
	}
	deleteSetQuery := "DELETE FROM sets WHERE id = ?"
	slog.Debug("exec", "query", logging.SanitizeQuery(deleteSetQuery), "args", []any{dbSetID})
	_, err = s.db.ExecContext(ctx, deleteSetQuery, dbSetID)
	if err != nil {
		return fmt.Errorf("failed to delete set %s: %w", setAPIID, err)
	}
	s.logger.Info("deleted set from database", "api_id", setAPIID, "db_id", dbSetID)
	return nil
}
