package tui

import (
	"github.com/laiambryant/tui-cardman/internal/model"
	"github.com/laiambryant/tui-cardman/internal/pokemontcg"
)

type fetchSetsSuccessMsg struct {
	sets []pokemontcg.Set
}

type fetchSetsErrorMsg struct {
	err error
}

type fetchDatabaseSetsSuccessMsg struct {
	apiIDs []string
}

type fetchDatabaseSetsErrorMsg struct {
	err error
}

type fetchSetDetailsSuccessMsg struct {
	set *model.Set
}

type fetchSetDetailsErrorMsg struct {
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
