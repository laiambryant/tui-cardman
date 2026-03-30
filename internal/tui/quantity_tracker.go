package tui

type QuantityTracker struct {
	db   map[int64]int
	temp map[int64]int
}

func newQuantityTracker() QuantityTracker {
	return QuantityTracker{
		db:   make(map[int64]int),
		temp: make(map[int64]int),
	}
}

func (qt *QuantityTracker) increment(id int64) {
	qt.temp[id]++
}

// decrement decrements the quantity for id. Returns false if the combined quantity is already 0.
func (qt *QuantityTracker) decrement(id int64) bool {
	if qt.db[id]+qt.temp[id] <= 0 {
		return false
	}
	qt.temp[id]--
	return true
}

func (qt *QuantityTracker) total(id int64) int {
	return qt.db[id] + qt.temp[id]
}

// totalCards returns the sum of all db quantities plus all pending deltas.
func (qt *QuantityTracker) totalCards() int {
	total := 0
	for _, qty := range qt.db {
		total += qty
	}
	for _, delta := range qt.temp {
		total += delta
	}
	return total
}

// pendingCount returns the number of cards with a non-zero pending delta.
func (qt *QuantityTracker) pendingCount() int {
	count := 0
	for _, delta := range qt.temp {
		if delta != 0 {
			count++
		}
	}
	return count
}

// buildUpdates returns the final quantity for each card with a pending change.
func (qt *QuantityTracker) buildUpdates() map[int64]int {
	updates := make(map[int64]int, len(qt.temp))
	for id, delta := range qt.temp {
		updates[id] = qt.db[id] + delta
	}
	return updates
}

// snapshot returns the full current state of all cards, including pending changes.
func (qt *QuantityTracker) snapshot() map[int64]int {
	combined := make(map[int64]int, len(qt.db))
	for id, qty := range qt.db {
		combined[id] = qty
	}
	for id, delta := range qt.temp {
		combined[id] += delta
	}
	return combined
}

// commit applies updates to the db state and clears pending changes.
func (qt *QuantityTracker) commit(updates map[int64]int) {
	for id, qty := range updates {
		qt.db[id] = qty
	}
	qt.temp = make(map[int64]int)
}

// load replaces the db state with the given quantities and clears pending changes.
func (qt *QuantityTracker) load(quantities map[int64]int) {
	qt.db = quantities
	qt.temp = make(map[int64]int)
}

// reset clears all db and pending state.
func (qt *QuantityTracker) reset() {
	qt.db = make(map[int64]int)
	qt.temp = make(map[int64]int)
}

// applyImport adds the given quantities to the pending changes.
func (qt *QuantityTracker) applyImport(quantities map[int64]int) {
	for id, qty := range quantities {
		qt.temp[id] += qty
	}
}
