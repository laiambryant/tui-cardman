// Package gameimporter defines the shared interface for game-specific importers,
// decoupling the TUI import view from concrete game implementations.
package gameimporter

import "context"

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
	CheckSetInDB(ctx context.Context, apiID string) (inDB bool, hasCollections bool, err error)
	ImportSet(ctx context.Context, apiID string) error
	DeleteSet(ctx context.Context, apiID string) error
	ImportAll(ctx context.Context) error
	ImportNew(ctx context.Context) error
	ImportSpecific(ctx context.Context, apiIDs []string) error
}
