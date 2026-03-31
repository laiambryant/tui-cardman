package yugioh

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	"github.com/laiambryant/tui-cardman/internal/model"
	card "github.com/laiambryant/tui-cardman/internal/services/cards"
	"github.com/laiambryant/tui-cardman/internal/services/importruns"
	"github.com/laiambryant/tui-cardman/internal/services/sets"
	"github.com/laiambryant/tui-cardman/internal/services/yugiohcard"
)

// ImportService handles importing Yu-Gi-Oh! cards from the API into the database.
type ImportService struct {
	db               *sql.DB
	client           *Client
	logger           *slog.Logger
	importRunService importruns.ImportRunService
	setService       sets.SetService
	cardService      card.CardService
	yugiohCardSvc    yugiohcard.YuGiOhCardService
	yugiohGameID     int64
}

// NewImportService creates a new YGO ImportService and initializes the game ID.
func NewImportService(
	db *sql.DB,
	client *Client,
	logger *slog.Logger,
	importRunService importruns.ImportRunService,
	setService sets.SetService,
	cardService card.CardService,
	yugiohCardSvc yugiohcard.YuGiOhCardService,
) *ImportService {
	svc := &ImportService{
		db:               db,
		client:           client,
		logger:           logger,
		importRunService: importRunService,
		setService:       setService,
		cardService:      cardService,
		yugiohCardSvc:    yugiohCardSvc,
	}
	if err := svc.initYuGiOhGameID(context.Background()); err != nil {
		logger.Error("failed to initialize Yu-Gi-Oh! game ID", "error", err)
	}
	return svc
}

func (s *ImportService) initYuGiOhGameID(ctx context.Context) error {
	var gameID int64
	err := s.db.QueryRowContext(ctx, "SELECT id FROM card_games WHERE name = ?", "Yu-Gi-Oh!").Scan(&gameID)
	if err != nil {
		return fmt.Errorf("failed to get Yu-Gi-Oh! game ID: %w", err)
	}
	s.yugiohGameID = gameID
	s.logger.Debug("initialized Yu-Gi-Oh! game ID", "id", gameID)
	return nil
}

// ImportSet imports all cards for a single YGO set.
func (s *ImportService) ImportSet(ctx context.Context, set YGOSet) (int, error) {
	s.logger.Info("importing YGO set", "set_code", set.SetCode, "set_name", set.SetName)

	setID, err := s.setService.UpsertSet(ctx, set.SetCode, set.SetCode, set.SetName, set.NumCards, set.NumCards)
	if err != nil {
		return 0, fmt.Errorf("failed to upsert set %s: %w", set.SetCode, err)
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
	offset := 0
	for {
		cards, morePages, fetchErr := s.client.GetCardsForSet(ctx, set.SetName, set.SetCode, offset)
		if fetchErr != nil {
			return cardsImported, fmt.Errorf("failed to fetch cards for set %s at offset %d: %w", set.SetCode, offset, fetchErr)
		}
		for _, c := range cards {
			if upsertErr := s.upsertCardTx(ctx, tx, c, setID); upsertErr != nil {
				s.logger.Error("failed to upsert YGO card", "card_id", c.ID, "name", c.Name, "error", upsertErr)
				continue
			}
			cardsImported++
		}
		s.logger.Info("imported YGO cards page", "set_code", set.SetCode, "offset", offset, "count", len(cards))
		if !morePages {
			break
		}
		offset += pageSize
	}

	if err = tx.Commit(); err != nil {
		return cardsImported, fmt.Errorf("failed to commit set transaction for %s: %w", set.SetCode, err)
	}

	s.logger.Info("completed YGO set import", "set_code", set.SetCode, "total_cards", cardsImported)
	return cardsImported, nil
}

func (s *ImportService) upsertCardTx(ctx context.Context, tx *sql.Tx, c YGOCard, setID int64) error {
	apiID := fmt.Sprintf("ygo-%d-%s", c.ID, c.SetCode)
	cardID, err := s.cardService.UpsertCard(ctx, tx, apiID, setID, c.CardNumber, c.Name, c.Rarity, "", s.yugiohGameID)
	if err != nil {
		return err
	}

	yc := &model.YuGiOhCard{
		CardID:      cardID,
		CardType:    c.Type,
		FrameType:   c.FrameType,
		Description: c.Desc,
		ATK:         c.ATK,
		DEF:         c.DEF,
		Level:       c.Level,
		Scale:       c.Scale,
		LinkVal:     c.LinkVal,
		LinkMarkers: c.LinkMarkers,
	}
	if c.Attribute != nil {
		yc.Attribute = c.Attribute
	}
	if c.Race != "" {
		race := c.Race
		yc.Race = &race
	}

	return s.yugiohCardSvc.UpsertYuGiOhCard(ctx, tx, cardID, yc)
}

type importResult struct {
	setsProcessed      int
	totalCardsImported int
	errorCount         int
	errorMessages      []string
}

func (s *ImportService) processSets(ctx context.Context, setsToImport []YGOSet) *importResult {
	result := &importResult{}
	for _, set := range setsToImport {
		cardsImported, err := s.ImportSet(ctx, set)
		if err != nil {
			s.logger.Error("failed to import YGO set", "set_code", set.SetCode, "error", err)
			result.errorCount++
			result.errorMessages = append(result.errorMessages, fmt.Sprintf("Set %s: %v", set.SetCode, err))
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

// ImportAllSets imports all available YGO sets.
func (s *ImportService) ImportAllSets(ctx context.Context) error {
	runID, err := s.importRunService.CreateImportRun(ctx, "import-full")
	if err != nil {
		return fmt.Errorf("failed to create import run: %w", err)
	}
	allSets, err := s.client.GetSets(ctx)
	if err != nil {
		_ = s.importRunService.UpdateImportRun(ctx, runID, "failed", 0, 0, 1, fmt.Sprintf("failed to fetch sets: %v", err))
		return fmt.Errorf("failed to fetch YGO sets: %w", err)
	}
	result := s.processSets(ctx, allSets)
	return s.completeImportRun(ctx, runID, result)
}

// ImportNewSets imports only sets not already in the database.
func (s *ImportService) ImportNewSets(ctx context.Context) error {
	runID, err := s.importRunService.CreateImportRun(ctx, "import-updates")
	if err != nil {
		return fmt.Errorf("failed to create import run: %w", err)
	}
	allSets, err := s.client.GetSets(ctx)
	if err != nil {
		_ = s.importRunService.UpdateImportRun(ctx, runID, "failed", 0, 0, 1, fmt.Sprintf("failed to fetch sets: %v", err))
		return fmt.Errorf("failed to fetch YGO sets: %w", err)
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

	var newSets []YGOSet
	for _, set := range allSets {
		if !existingMap[set.SetCode] {
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

// ImportSpecificSets imports a set of YGO sets identified by their set codes.
func (s *ImportService) ImportSpecificSets(ctx context.Context, setCodes []string) error {
	runID, err := s.importRunService.CreateImportRun(ctx, "import-specific")
	if err != nil {
		return fmt.Errorf("failed to create import run: %w", err)
	}
	allSets, err := s.client.GetSets(ctx)
	if err != nil {
		_ = s.importRunService.UpdateImportRun(ctx, runID, "failed", 0, 0, 1, fmt.Sprintf("failed to fetch sets: %v", err))
		return fmt.Errorf("failed to fetch YGO sets: %w", err)
	}

	setMap := make(map[string]YGOSet, len(allSets))
	for _, set := range allSets {
		setMap[set.SetCode] = set
	}

	var setsToImport []YGOSet
	var notFound []string
	for _, code := range setCodes {
		if set, ok := setMap[code]; ok {
			setsToImport = append(setsToImport, set)
		} else {
			notFound = append(notFound, code)
		}
	}
	if len(notFound) > 0 {
		s.logger.Warn("some YGO sets not found", "not_found", strings.Join(notFound, ", "))
	}
	if len(setsToImport) == 0 {
		msg := fmt.Sprintf("none of the specified sets were found: %s", strings.Join(setCodes, ", "))
		_ = s.importRunService.UpdateImportRun(ctx, runID, "failed", 0, 0, 1, msg)
		return fmt.Errorf("%s", msg)
	}

	result := s.processSets(ctx, setsToImport)
	return s.completeImportRun(ctx, runID, result)
}

// DeleteSetByAPIID removes a YGO set (by set code) and its cards from the database.
func (s *ImportService) DeleteSetByAPIID(ctx context.Context, setCode string) error {
	dbSetID, err := s.setService.GetSetIDByAPIID(ctx, setCode)
	if err == sql.ErrNoRows {
		s.logger.Debug("YGO set not in database, nothing to delete", "set_code", setCode)
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to look up set %s: %w", setCode, err)
	}
	if _, err := s.db.ExecContext(ctx, "DELETE FROM cards WHERE set_id = ?", dbSetID); err != nil {
		return fmt.Errorf("failed to delete cards for YGO set %s: %w", setCode, err)
	}
	if _, err := s.db.ExecContext(ctx, "DELETE FROM sets WHERE id = ?", dbSetID); err != nil {
		return fmt.Errorf("failed to delete YGO set %s: %w", setCode, err)
	}
	s.logger.Info("deleted YGO set from database", "set_code", setCode, "db_id", dbSetID)
	return nil
}
