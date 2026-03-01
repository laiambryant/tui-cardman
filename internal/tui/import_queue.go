package tui

const (
	queueStatusPending   = "pending"
	queueStatusImporting = "importing"
	queueStatusCompleted = "completed"
	queueStatusError     = "error"
)

type importQueueItem struct {
	setID   string
	setName string
	status  string
	err     error
}

func (m *ImportModel) addToQueue(setID, setName string) {
	for _, item := range m.importQueue {
		if item.setID == setID {
			return
		}
	}
	m.importQueue = append(m.importQueue, importQueueItem{
		setID:   setID,
		setName: setName,
		status:  queueStatusPending,
	})
}

func (m *ImportModel) removeFromQueue(setID string) {
	for i, item := range m.importQueue {
		if item.setID == setID {
			m.importQueue = append(m.importQueue[:i], m.importQueue[i+1:]...)
			return
		}
	}
}

func (m *ImportModel) clearCompletedFromQueue() {
	var remaining []importQueueItem
	for _, item := range m.importQueue {
		if item.status != queueStatusCompleted && item.status != queueStatusError {
			remaining = append(remaining, item)
		}
	}
	m.importQueue = remaining
}

func (m *ImportModel) queuePendingCount() int {
	count := 0
	for _, item := range m.importQueue {
		if item.status == queueStatusPending {
			count++
		}
	}
	return count
}

func (m *ImportModel) isInQueue(setID string) bool {
	for _, item := range m.importQueue {
		if item.setID == setID {
			return true
		}
	}
	return false
}

func queueItemIcon(status string) string {
	switch status {
	case queueStatusCompleted:
		return SuccessIcon
	case queueStatusImporting:
		return ImportIcon
	case queueStatusError:
		return FailureIcon
	default:
		return PendingIcon
	}
}
