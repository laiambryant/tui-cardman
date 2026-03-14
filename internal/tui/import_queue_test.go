package tui

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func newTestImportModel() ImportModel {
	return ImportModel{
		databaseSetIDs:    make(map[string]bool),
		databaseSetCounts: make(map[string]int),
	}
}

func TestAddToQueue(t *testing.T) {
	m := newTestImportModel()
	m.addToQueue("KLD", "Kaladesh")

	assert.Len(t, m.importQueue, 1)
	assert.Equal(t, "KLD", m.importQueue[0].setID)
	assert.Equal(t, "Kaladesh", m.importQueue[0].setName)
	assert.Equal(t, queueStatusPending, m.importQueue[0].status)
}

func TestAddToQueue_NoDuplicates(t *testing.T) {
	m := newTestImportModel()
	m.addToQueue("KLD", "Kaladesh")
	m.addToQueue("KLD", "Kaladesh")

	assert.Len(t, m.importQueue, 1)
}

func TestAddToQueue_Multiple(t *testing.T) {
	m := newTestImportModel()
	m.addToQueue("KLD", "Kaladesh")
	m.addToQueue("AER", "Aether Revolt")
	m.addToQueue("M21", "Core Set 2021")

	assert.Len(t, m.importQueue, 3)
}

func TestRemoveFromQueue(t *testing.T) {
	m := newTestImportModel()
	m.addToQueue("KLD", "Kaladesh")
	m.addToQueue("AER", "Aether Revolt")
	m.removeFromQueue("KLD")

	assert.Len(t, m.importQueue, 1)
	assert.Equal(t, "AER", m.importQueue[0].setID)
}

func TestRemoveFromQueue_NotFound(t *testing.T) {
	m := newTestImportModel()
	m.addToQueue("KLD", "Kaladesh")
	m.removeFromQueue("XYZ")

	assert.Len(t, m.importQueue, 1)
}

func TestRemoveFromQueue_Empty(t *testing.T) {
	m := newTestImportModel()
	m.removeFromQueue("KLD")

	assert.Empty(t, m.importQueue)
}

func TestClearCompletedFromQueue(t *testing.T) {
	m := newTestImportModel()
	m.importQueue = []importQueueItem{
		{setID: "KLD", status: queueStatusCompleted},
		{setID: "AER", status: queueStatusPending},
		{setID: "M21", status: queueStatusError, err: errors.New("fail")},
		{setID: "XLN", status: queueStatusImporting},
	}
	m.clearCompletedFromQueue()

	assert.Len(t, m.importQueue, 2)
	assert.Equal(t, "AER", m.importQueue[0].setID)
	assert.Equal(t, "XLN", m.importQueue[1].setID)
}

func TestClearCompletedFromQueue_AllCompleted(t *testing.T) {
	m := newTestImportModel()
	m.importQueue = []importQueueItem{
		{setID: "KLD", status: queueStatusCompleted},
		{setID: "AER", status: queueStatusCompleted},
	}
	m.clearCompletedFromQueue()

	assert.Empty(t, m.importQueue)
}

func TestClearCompletedFromQueue_NoneCompleted(t *testing.T) {
	m := newTestImportModel()
	m.importQueue = []importQueueItem{
		{setID: "KLD", status: queueStatusPending},
		{setID: "AER", status: queueStatusImporting},
	}
	m.clearCompletedFromQueue()

	assert.Len(t, m.importQueue, 2)
}

func TestQueuePendingCount(t *testing.T) {
	m := newTestImportModel()
	m.importQueue = []importQueueItem{
		{setID: "KLD", status: queueStatusPending},
		{setID: "AER", status: queueStatusCompleted},
		{setID: "M21", status: queueStatusPending},
		{setID: "XLN", status: queueStatusImporting},
	}

	assert.Equal(t, 2, m.queuePendingCount())
}

func TestQueuePendingCount_Empty(t *testing.T) {
	m := newTestImportModel()
	assert.Equal(t, 0, m.queuePendingCount())
}

func TestIsInQueue(t *testing.T) {
	m := newTestImportModel()
	m.addToQueue("KLD", "Kaladesh")

	assert.True(t, m.isInQueue("KLD"))
	assert.False(t, m.isInQueue("AER"))
}

func TestIsInQueue_Empty(t *testing.T) {
	m := newTestImportModel()
	assert.False(t, m.isInQueue("KLD"))
}

func TestQueueItemIcon(t *testing.T) {
	assert.Equal(t, SuccessIcon, queueItemIcon(queueStatusCompleted))
	assert.Equal(t, ImportIcon, queueItemIcon(queueStatusImporting))
	assert.Equal(t, FailureIcon, queueItemIcon(queueStatusError))
	assert.Equal(t, PendingIcon, queueItemIcon(queueStatusPending))
	assert.Equal(t, PendingIcon, queueItemIcon("unknown"))
}
