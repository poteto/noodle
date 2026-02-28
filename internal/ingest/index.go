package ingest

import "sync"

// AppliedEventIndex tracks first-applied event IDs by idempotency key.
type AppliedEventIndex struct {
	mu      sync.RWMutex
	applied map[string]EventID
}

// NewAppliedEventIndex creates an empty index.
func NewAppliedEventIndex() *AppliedEventIndex {
	return &AppliedEventIndex{
		applied: make(map[string]EventID),
	}
}

// MarkApplied records key -> event ID. The first applied ID is retained.
func (i *AppliedEventIndex) MarkApplied(key string, id EventID) {
	i.mu.Lock()
	defer i.mu.Unlock()
	if _, exists := i.applied[key]; exists {
		return
	}
	i.applied[key] = id
}

// IsDuplicate reports whether the key has already been applied.
func (i *AppliedEventIndex) IsDuplicate(key string) (EventID, bool) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	id, exists := i.applied[key]
	return id, exists
}
