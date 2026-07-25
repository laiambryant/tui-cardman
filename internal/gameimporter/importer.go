// Package gameimporter defines the shared interface for game-specific importers,
// decoupling the TUI import view from concrete game implementations.
package gameimporter

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/laiambryant/tui-cardman/internal/services/sets"
)

// GameSet is a game-agnostic representation of a card set for use in the TUI.
type GameSet struct {
	APIID string
	Name  string
	Code  string
	Total int
}

// GameImporter is the interface that each card game's import backend must implement.
type GameImporter interface {
	FetchSets(ctx context.Context) ([]GameSet, error)
	GetImportedSetIDs(ctx context.Context) (map[string]bool, error)
	GetImportedSetCounts(ctx context.Context) (map[string]int, error)
	CheckSetInDB(ctx context.Context, apiID string) (inDB, hasCollections bool, err error)
	ImportSet(ctx context.Context, apiID string) error
	DeleteSet(ctx context.Context, apiID string) error
	ImportAll(ctx context.Context) error
	ImportNew(ctx context.Context) error
	ImportSpecific(ctx context.Context, apiIDs []string) error
}

// ImportRunner is the subset of a game's import service used by Base.
type ImportRunner interface {
	ImportAllSets(ctx context.Context) error
	ImportNewSets(ctx context.Context) error
	ImportSpecificSets(ctx context.Context, apiIDs []string) error
	DeleteSetByAPIID(ctx context.Context, apiID string) error
}

// Base implements GameImporter for any game given an ImportRunner, a set
// service and a game-specific set fetcher.
type Base struct {
	Runner      ImportRunner
	SetService  sets.SetService
	FetchSetsFn func(ctx context.Context) ([]GameSet, error)
}

func (b *Base) FetchSets(ctx context.Context) ([]GameSet, error) {
	return b.FetchSetsFn(ctx)
}

func (b *Base) GetImportedSetIDs(ctx context.Context) (map[string]bool, error) {
	apiIDs, err := b.SetService.GetAllSetAPIIDs(ctx)
	if err != nil {
		return nil, err
	}
	result := make(map[string]bool, len(apiIDs))
	for _, id := range apiIDs {
		result[id] = true
	}
	return result, nil
}

func (b *Base) GetImportedSetCounts(ctx context.Context) (map[string]int, error) {
	return b.SetService.GetAllSetAPIIDsWithCounts(ctx)
}

func (b *Base) CheckSetInDB(ctx context.Context, apiID string) (bool, bool, error) {
	dbSetID, err := b.SetService.GetSetIDByAPIID(ctx, apiID)
	if err == sql.ErrNoRows {
		return false, false, nil
	}
	if err != nil {
		return false, false, fmt.Errorf("failed to look up set %s: %w", apiID, err)
	}
	hasCollections, err := b.SetService.SetHasUserCollections(ctx, dbSetID)
	if err != nil {
		return true, false, fmt.Errorf("failed to check collections for set %s: %w", apiID, err)
	}
	return true, hasCollections, nil
}

func (b *Base) ImportSet(ctx context.Context, apiID string) error {
	return b.Runner.ImportSpecificSets(ctx, []string{apiID})
}

func (b *Base) DeleteSet(ctx context.Context, apiID string) error {
	return b.Runner.DeleteSetByAPIID(ctx, apiID)
}

func (b *Base) ImportAll(ctx context.Context) error {
	return b.Runner.ImportAllSets(ctx)
}

func (b *Base) ImportNew(ctx context.Context) error {
	return b.Runner.ImportNewSets(ctx)
}

func (b *Base) ImportSpecific(ctx context.Context, apiIDs []string) error {
	return b.Runner.ImportSpecificSets(ctx, apiIDs)
}
