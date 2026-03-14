package onepiece

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	"github.com/laiambryant/tui-cardman/internal/model"
	card "github.com/laiambryant/tui-cardman/internal/services/cards"
	"github.com/laiambryant/tui-cardman/internal/services/importruns"
	"github.com/laiambryant/tui-cardman/internal/services/onepiececard"
	"github.com/laiambryant/tui-cardman/internal/services/sets"
)

type ImportService struct {
	db               *sql.DB
	client           *Client
	logger           *slog.Logger
	importRunService importruns.ImportRunService
	setService       sets.SetService
	cardService      card.CardService
	opCardSvc        onepiececard.OnePieceCardService
	opGameID         int64
}

func NewImportService(
	db *sql.DB,
	client *Client,
	logger *slog.Logger,
	importRunService importruns.ImportRunService,
	setService sets.SetService,
	cardService card.CardService,
	opCardSvc onepiececard.OnePieceCardService,
) *ImportService {
	svc := &ImportService{
		db:               db,
		client:           client,
		logger:           logger,
		importRunService: importRunService,
		setService:       setService,
		cardService:      cardService,
		opCardSvc:        opCardSvc,
	}
	if err := svc.initGameID(context.Background()); err != nil {
		logger.Error("failed to initialise One Piece game ID", "error", err)
	}
	return svc
}

func (s *ImportService) initGameID(ctx context.Context) error {
	var gameID int64
	err := s.db.QueryRowContext(ctx, "SELECT id FROM card_games WHERE name = ?", "One Piece").Scan(&gameID)
	if err != nil {
		return fmt.Errorf("failed to get One Piece game ID: %w", err)
	}
	s.opGameID = gameID
	s.logger.Debug("initialised One Piece game ID", "id", gameID)
	return nil
}

func (s *ImportService) ImportSet(ctx context.Context, set OPSet) (int, error) {
	s.logger.Info("importing One Piece set", "set_id", set.SetID, "set_name", set.SetName)
	cards, err := s.client.GetCardsForSet(ctx, set)
	if err != nil {
		return 0, err
	}
	setID, err := s.setService.UpsertSet(ctx, set.SetID, set.SetID, set.SetName, len(cards), len(cards))
	if err != nil {
		return 0, fmt.Errorf("failed to upsert set %s: %w", set.SetID, err)
	}
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
	for _, c := range cards {
		if upsertErr := s.upsertCardTx(ctx, tx, c, setID); upsertErr != nil {
			s.logger.Error("failed to upsert One Piece card", "card_set_id", c.CardSetID, "name", c.CardName, "error", upsertErr)
			continue
		}
		cardsImported++
	}
	if err = tx.Commit(); err != nil {
		return cardsImported, fmt.Errorf("failed to commit set transaction for %s: %w", set.SetID, err)
	}
	if cardsImported == 0 {
		s.logger.Warn("One Piece set import completed with zero cards", "set_id", set.SetID)
	}
	s.logger.Info("completed One Piece set import", "set_id", set.SetID, "total_cards", cardsImported)
	return cardsImported, nil
}

func (s *ImportService) upsertCardTx(ctx context.Context, tx *sql.Tx, c OPCard, setID int64) error {
	cardID, err := s.cardService.UpsertCard(ctx, tx, c.CardSetID, setID, c.CardSetID, c.CardName, c.Rarity, "", s.opGameID)
	if err != nil {
		return err
	}
	oc := &model.OnePieceCard{
		CardID:        cardID,
		CardColor:     c.CardColor,
		CardType:      c.CardType,
		CardText:      c.CardText,
		SubTypes:      c.SubTypes,
		Attribute:     c.Attribute,
		Life:          c.Life,
		CardCost:      c.CardCost,
		CardPower:     c.CardPower,
		CounterAmount: c.CounterAmount,
	}
	return s.opCardSvc.UpsertOnePieceCard(ctx, tx, cardID, oc)
}

type importResult struct {
	setsProcessed      int
	totalCardsImported int
	errorCount         int
	errorMessages      []string
}

func (s *ImportService) processSets(ctx context.Context, setsToImport []OPSet) *importResult {
	result := &importResult{}
	for _, set := range setsToImport {
		cardsImported, err := s.ImportSet(ctx, set)
		if err != nil {
			s.logger.Error("failed to import One Piece set", "set_id", set.SetID, "error", err)
			result.errorCount++
			result.errorMessages = append(result.errorMessages, fmt.Sprintf("Set %s: %v", set.SetID, err))
			continue
		}
		result.totalCardsImported += cardsImported
		result.setsProcessed++
	}
	return result
}

func (s *ImportService) completeImportRun(ctx context.Context, runID int64, result *importResult) error {
	status := "completed"
	notes := fmt.Sprintf("Imported %d sets with %d total cards", result.setsProcessed, result.totalCardsImported)
	if result.errorCount > 0 {
		status = "completed_with_errors"
		notes += fmt.Sprintf(". Errors: %s", strings.Join(result.errorMessages, "; "))
	}
	return s.importRunService.UpdateImportRun(ctx, runID, status, result.setsProcessed, result.totalCardsImported, result.errorCount, notes)
}

func (s *ImportService) ImportAllSets(ctx context.Context) error {
	runID, err := s.importRunService.CreateImportRun(ctx, "import-full")
	if err != nil {
		return fmt.Errorf("failed to create import run: %w", err)
	}
	allSets, err := s.client.GetSets(ctx)
	if err != nil {
		_ = s.importRunService.UpdateImportRun(ctx, runID, "failed", 0, 0, 1, fmt.Sprintf("failed to fetch sets: %v", err))
		return fmt.Errorf("failed to fetch One Piece sets: %w", err)
	}
	result := s.processSets(ctx, allSets)
	return s.completeImportRun(ctx, runID, result)
}

func (s *ImportService) ImportNewSets(ctx context.Context) error {
	runID, err := s.importRunService.CreateImportRun(ctx, "import-updates")
	if err != nil {
		return fmt.Errorf("failed to create import run: %w", err)
	}
	allSets, err := s.client.GetSets(ctx)
	if err != nil {
		_ = s.importRunService.UpdateImportRun(ctx, runID, "failed", 0, 0, 1, fmt.Sprintf("failed to fetch sets: %v", err))
		return fmt.Errorf("failed to fetch One Piece sets: %w", err)
	}
	existing, err := s.setService.GetAllSetAPIIDs(ctx)
	if err != nil {
		_ = s.importRunService.UpdateImportRun(ctx, runID, "failed", 0, 0, 1, fmt.Sprintf("failed to query existing sets: %v", err))
		return fmt.Errorf("failed to query existing sets: %w", err)
	}
	existingMap := make(map[string]bool, len(existing))
	for _, id := range existing {
		existingMap[id] = true
	}
	var newSets []OPSet
	for _, set := range allSets {
		if !existingMap[set.SetID] {
			newSets = append(newSets, set)
		}
	}
	if len(newSets) == 0 {
		_ = s.importRunService.UpdateImportRun(ctx, runID, "completed", 0, 0, 0, "no new sets found")
		return nil
	}
	result := s.processSets(ctx, newSets)
	return s.completeImportRun(ctx, runID, result)
}

func (s *ImportService) ImportSpecificSets(ctx context.Context, setIDs []string) error {
	runID, err := s.importRunService.CreateImportRun(ctx, "import-specific")
	if err != nil {
		return fmt.Errorf("failed to create import run: %w", err)
	}
	allSets, err := s.client.GetSets(ctx)
	if err != nil {
		_ = s.importRunService.UpdateImportRun(ctx, runID, "failed", 0, 0, 1, fmt.Sprintf("failed to fetch sets: %v", err))
		return fmt.Errorf("failed to fetch One Piece sets: %w", err)
	}
	setMap := make(map[string]OPSet, len(allSets))
	for _, set := range allSets {
		setMap[set.SetID] = set
	}
	var setsToImport []OPSet
	var notFound []string
	for _, id := range setIDs {
		if set, ok := setMap[id]; ok {
			setsToImport = append(setsToImport, set)
		} else {
			notFound = append(notFound, id)
		}
	}
	if len(notFound) > 0 {
		s.logger.Warn("some One Piece sets not found", "not_found", strings.Join(notFound, ", "))
	}
	if len(setsToImport) == 0 {
		msg := fmt.Sprintf("none of the specified sets were found: %s", strings.Join(setIDs, ", "))
		_ = s.importRunService.UpdateImportRun(ctx, runID, "failed", 0, 0, 1, msg)
		return fmt.Errorf("%s", msg)
	}
	result := s.processSets(ctx, setsToImport)
	return s.completeImportRun(ctx, runID, result)
}

func (s *ImportService) DeleteSetByAPIID(ctx context.Context, setID string) error {
	dbSetID, err := s.setService.GetSetIDByAPIID(ctx, setID)
	if err == sql.ErrNoRows {
		s.logger.Debug("One Piece set not in database, nothing to delete", "set_id", setID)
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to look up set %s: %w", setID, err)
	}
	if _, err := s.db.ExecContext(ctx, "DELETE FROM cards WHERE set_id = ?", dbSetID); err != nil {
		return fmt.Errorf("failed to delete cards for One Piece set %s: %w", setID, err)
	}
	if _, err := s.db.ExecContext(ctx, "DELETE FROM sets WHERE id = ?", dbSetID); err != nil {
		return fmt.Errorf("failed to delete One Piece set %s: %w", setID, err)
	}
	s.logger.Info("deleted One Piece set from database", "set_id", setID, "db_id", dbSetID)
	return nil
}
