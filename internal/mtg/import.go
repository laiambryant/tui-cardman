package mtg

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	"github.com/laiambryant/tui-cardman/internal/model"
	card "github.com/laiambryant/tui-cardman/internal/services/cards"
	"github.com/laiambryant/tui-cardman/internal/services/importruns"
	"github.com/laiambryant/tui-cardman/internal/services/mtgcard"
	"github.com/laiambryant/tui-cardman/internal/services/sets"
)

type ImportService struct {
	db               *sql.DB
	client           *Client
	logger           *slog.Logger
	importRunService importruns.ImportRunService
	setService       sets.SetService
	cardService      card.CardService
	mtgCardSvc       mtgcard.MTGCardService
	mtgGameID        int64
}

func NewImportService(
	db *sql.DB,
	client *Client,
	logger *slog.Logger,
	importRunService importruns.ImportRunService,
	setService sets.SetService,
	cardService card.CardService,
	mtgCardSvc mtgcard.MTGCardService,
) *ImportService {
	svc := &ImportService{
		db:               db,
		client:           client,
		logger:           logger,
		importRunService: importRunService,
		setService:       setService,
		cardService:      cardService,
		mtgCardSvc:       mtgCardSvc,
	}
	if err := svc.initMTGGameID(context.Background()); err != nil {
		logger.Error("failed to initialize MTG game ID", "error", err)
	}
	return svc
}

func (s *ImportService) initMTGGameID(ctx context.Context) error {
	var gameID int64
	err := s.db.QueryRowContext(ctx, "SELECT id FROM card_games WHERE name = ?", "Magic: The Gathering").Scan(&gameID)
	if err != nil {
		return fmt.Errorf("failed to get MTG game ID: %w", err)
	}
	s.mtgGameID = gameID
	s.logger.Debug("initialized MTG game ID", "id", gameID)
	return nil
}

func (s *ImportService) ImportSet(ctx context.Context, set MTGSet) (int, error) {
	s.logger.Info("importing MTG set", "set_code", set.SetCode, "set_name", set.Name)
	setID, err := s.setService.UpsertSet(ctx, set.SetCode, set.SetCode, set.Name, set.NumCards, set.NumCards)
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
	page := 1
	for {
		cards, hasMore, totalCount, fetchErr := s.client.GetCardsForSet(ctx, set.SetCode, page)
		if fetchErr != nil {
			return cardsImported, fmt.Errorf("failed to fetch cards for set %s page %d: %w", set.SetCode, page, fetchErr)
		}
		if page == 1 && totalCount > 0 {
			_, _ = s.setService.UpsertSet(ctx, set.SetCode, set.SetCode, set.Name, totalCount, totalCount)
		}
		for _, c := range cards {
			if upsertErr := s.upsertCardTx(ctx, tx, c, setID); upsertErr != nil {
				s.logger.Error("failed to upsert MTG card", "card_id", c.ID, "name", c.Name, "error", upsertErr)
				continue
			}
			cardsImported++
		}
		s.logger.Info("imported MTG cards page", "set_code", set.SetCode, "page", page, "count", len(cards))
		if !hasMore {
			break
		}
		page++
	}
	if err = tx.Commit(); err != nil {
		return cardsImported, fmt.Errorf("failed to commit set transaction for %s: %w", set.SetCode, err)
	}
	if cardsImported == 0 {
		s.logger.Warn("MTG set import completed with zero cards", "set_code", set.SetCode)
	}
	s.logger.Info("completed MTG set import", "set_code", set.SetCode, "total_cards", cardsImported)
	return cardsImported, nil
}

func (s *ImportService) upsertCardTx(ctx context.Context, tx *sql.Tx, c MTGCard, setID int64) error {
	cardID, err := s.cardService.UpsertCard(ctx, tx, c.ID, setID, c.Number, c.Name, c.Rarity, c.Artist, s.mtgGameID)
	if err != nil {
		return err
	}
	legalities := make([]model.MTGLegality, 0, len(c.Legalities))
	for _, l := range c.Legalities {
		legalities = append(legalities, model.MTGLegality{Format: l.Format, LegalityName: l.LegalityName})
	}
	mc := &model.MagicCard{
		CardID:        cardID,
		ManaCost:      c.ManaCost,
		CMC:           c.CMC,
		Colors:        c.Colors,
		ColorIdentity: c.ColorIdentity,
		TypeLine:      c.Type,
		Types:         c.Types,
		Supertypes:    c.Supertypes,
		Subtypes:      c.Subtypes,
		Text:          c.Text,
		Flavor:        c.Flavor,
		Power:         c.Power,
		Toughness:     c.Toughness,
		Loyalty:       c.Loyalty,
		Layout:        c.Layout,
		Legalities:    legalities,
	}
	return s.mtgCardSvc.UpsertMTGCard(ctx, tx, cardID, mc)
}

type importResult struct {
	setsProcessed      int
	totalCardsImported int
	errorCount         int
	errorMessages      []string
}

func (s *ImportService) processSets(ctx context.Context, setsToImport []MTGSet) *importResult {
	result := &importResult{}
	for _, set := range setsToImport {
		cardsImported, err := s.ImportSet(ctx, set)
		if err != nil {
			s.logger.Error("failed to import MTG set", "set_code", set.SetCode, "error", err)
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

func (s *ImportService) ImportAllSets(ctx context.Context) error {
	runID, err := s.importRunService.CreateImportRun(ctx, "import-full")
	if err != nil {
		return fmt.Errorf("failed to create import run: %w", err)
	}
	allSets, err := s.client.GetSets(ctx)
	if err != nil {
		_ = s.importRunService.UpdateImportRun(ctx, runID, "failed", 0, 0, 1, fmt.Sprintf("failed to fetch sets: %v", err))
		return fmt.Errorf("failed to fetch MTG sets: %w", err)
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
		return fmt.Errorf("failed to fetch MTG sets: %w", err)
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
	var newSets []MTGSet
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

func (s *ImportService) ImportSpecificSets(ctx context.Context, setCodes []string) error {
	runID, err := s.importRunService.CreateImportRun(ctx, "import-specific")
	if err != nil {
		return fmt.Errorf("failed to create import run: %w", err)
	}
	allSets, err := s.client.GetSets(ctx)
	if err != nil {
		_ = s.importRunService.UpdateImportRun(ctx, runID, "failed", 0, 0, 1, fmt.Sprintf("failed to fetch sets: %v", err))
		return fmt.Errorf("failed to fetch MTG sets: %w", err)
	}
	setMap := make(map[string]MTGSet, len(allSets))
	for _, set := range allSets {
		setMap[set.SetCode] = set
	}
	var setsToImport []MTGSet
	var notFound []string
	for _, code := range setCodes {
		if set, ok := setMap[code]; ok {
			setsToImport = append(setsToImport, set)
		} else {
			notFound = append(notFound, code)
		}
	}
	if len(notFound) > 0 {
		s.logger.Warn("some MTG sets not found", "not_found", strings.Join(notFound, ", "))
	}
	if len(setsToImport) == 0 {
		msg := fmt.Sprintf("none of the specified sets were found: %s", strings.Join(setCodes, ", "))
		_ = s.importRunService.UpdateImportRun(ctx, runID, "failed", 0, 0, 1, msg)
		return fmt.Errorf("%s", msg)
	}
	result := s.processSets(ctx, setsToImport)
	return s.completeImportRun(ctx, runID, result)
}

func (s *ImportService) DeleteSetByAPIID(ctx context.Context, setCode string) error {
	dbSetID, err := s.setService.GetSetIDByAPIID(ctx, setCode)
	if err == sql.ErrNoRows {
		s.logger.Debug("MTG set not in database, nothing to delete", "set_code", setCode)
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to look up set %s: %w", setCode, err)
	}
	if _, err := s.db.ExecContext(ctx, "DELETE FROM cards WHERE set_id = ?", dbSetID); err != nil {
		return fmt.Errorf("failed to delete cards for MTG set %s: %w", setCode, err)
	}
	if _, err := s.db.ExecContext(ctx, "DELETE FROM sets WHERE id = ?", dbSetID); err != nil {
		return fmt.Errorf("failed to delete MTG set %s: %w", setCode, err)
	}
	s.logger.Info("deleted MTG set from database", "set_code", setCode, "db_id", dbSetID)
	return nil
}
