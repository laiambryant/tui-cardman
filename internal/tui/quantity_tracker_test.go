package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewQuantityTracker(t *testing.T) {
	qt := newQuantityTracker()
	assert.Equal(t, 0, qt.total(1))
	assert.Equal(t, 0, qt.pendingCount())
	assert.Equal(t, 0, qt.totalCards())
}

func TestIncrement(t *testing.T) {
	qt := newQuantityTracker()
	qt.increment(1)
	qt.increment(1)
	qt.increment(2)
	assert.Equal(t, 2, qt.total(1))
	assert.Equal(t, 1, qt.total(2))
}

func TestDecrement_Success(t *testing.T) {
	qt := newQuantityTracker()
	qt.increment(1)
	ok := qt.decrement(1)
	assert.True(t, ok)
	assert.Equal(t, 0, qt.total(1))
}

func TestDecrement_AtZero(t *testing.T) {
	qt := newQuantityTracker()
	ok := qt.decrement(1)
	assert.False(t, ok)
	assert.Equal(t, 0, qt.total(1))
}

func TestDecrement_BelowZeroBlocked(t *testing.T) {
	qt := newQuantityTracker()
	qt.increment(1)
	qt.decrement(1)
	ok := qt.decrement(1)
	assert.False(t, ok)
	assert.Equal(t, 0, qt.total(1))
}

func TestPendingCount(t *testing.T) {
	qt := newQuantityTracker()
	assert.Equal(t, 0, qt.pendingCount())
	qt.increment(1)
	assert.Equal(t, 1, qt.pendingCount())
	qt.increment(2)
	assert.Equal(t, 2, qt.pendingCount())
	qt.decrement(1)
	assert.Equal(t, 1, qt.pendingCount())
}

func TestBuildUpdates(t *testing.T) {
	qt := newQuantityTracker()
	qt.load(map[int64]int{1: 3, 2: 1})
	qt.increment(1)
	qt.decrement(2)
	updates := qt.buildUpdates()
	assert.Equal(t, 4, updates[1])
	assert.Equal(t, 0, updates[2])
}

func TestCommit(t *testing.T) {
	qt := newQuantityTracker()
	qt.load(map[int64]int{1: 2})
	qt.increment(1)
	updates := qt.buildUpdates()
	qt.commit(updates)
	assert.Equal(t, 3, qt.total(1))
	assert.Equal(t, 0, qt.pendingCount())
}

func TestLoad(t *testing.T) {
	qt := newQuantityTracker()
	qt.increment(99)
	qt.load(map[int64]int{1: 5})
	assert.Equal(t, 5, qt.total(1))
	assert.Equal(t, 0, qt.total(99))
	assert.Equal(t, 0, qt.pendingCount())
}

func TestReset(t *testing.T) {
	qt := newQuantityTracker()
	qt.load(map[int64]int{1: 10})
	qt.increment(1)
	qt.reset()
	assert.Equal(t, 0, qt.total(1))
	assert.Equal(t, 0, qt.totalCards())
}

func TestTotalCards(t *testing.T) {
	qt := newQuantityTracker()
	qt.load(map[int64]int{1: 4, 2: 3})
	qt.increment(1)
	qt.decrement(2)
	assert.Equal(t, 7, qt.totalCards())
}

func TestSnapshot(t *testing.T) {
	qt := newQuantityTracker()
	qt.load(map[int64]int{1: 2, 2: 3})
	qt.increment(1)
	snap := qt.snapshot()
	assert.Equal(t, 3, snap[1])
	assert.Equal(t, 3, snap[2])
}

func TestApplyImport(t *testing.T) {
	qt := newQuantityTracker()
	qt.applyImport(map[int64]int{1: 2, 2: 1})
	qt.applyImport(map[int64]int{1: 1})
	assert.Equal(t, 3, qt.total(1))
	assert.Equal(t, 1, qt.total(2))
}
