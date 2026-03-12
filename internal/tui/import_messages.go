package tui

import (
	"github.com/laiambryant/tui-cardman/internal/gameimporter"
)

type fetchSetsSuccessMsg struct {
	sets []gameimporter.GameSet
}

type fetchSetsErrorMsg struct {
	err error
}

type fetchDatabaseSetsSuccessMsg struct {
	apiIDs map[string]bool
}

type fetchDatabaseSetsErrorMsg struct {
	err error
}

type checkSetInDBMsg struct {
	hasCollections bool
}

type checkSetNotInDBMsg struct{}

type checkSetInCollectionErrorMsg struct {
	err error
}

type importSetSuccessMsg struct {
	setID string
}

type importSetErrorMsg struct {
	setID string
	err   error
}

type deleteSetSuccessMsg struct {
	setID string
}

type deleteSetErrorMsg struct {
	setID string
	err   error
}

type importProgressMsg struct {
	setID         string
	setsCompleted int
	totalSets     int
	cardsImported int
}

type importAllSetsSuccessMsg struct{}

type importAllSetsErrorMsg struct {
	err error
}

type importNewSetsSuccessMsg struct{}

type importNewSetsErrorMsg struct {
	err error
}

type importConfirmedMsg struct{}
