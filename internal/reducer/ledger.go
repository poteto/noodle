package reducer

import (
	"fmt"
	"sync"
	"time"
)

// EffectLedger tracks effect lifecycle transitions for retry/idempotency.
type EffectLedger struct {
	mu      sync.RWMutex
	records map[string]EffectLedgerRecord
	order   []string
}

// NewEffectLedger creates an empty effect ledger.
func NewEffectLedger() *EffectLedger {
	return &EffectLedger{
		records: make(map[string]EffectLedgerRecord),
		order:   []string{},
	}
}

// Record adds a new effect as pending. Existing effect IDs are preserved.
func (l *EffectLedger) Record(effect Effect) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.records == nil {
		l.records = make(map[string]EffectLedgerRecord)
	}
	if _, exists := l.records[effect.EffectID]; exists {
		return
	}

	record := EffectLedgerRecord{
		EffectID: effect.EffectID,
		Effect:   effect,
		Status:   EffectLedgerPending,
	}
	l.records[effect.EffectID] = record
	l.order = append(l.order, effect.EffectID)
}

// MarkRunning transitions effect status to running.
func (l *EffectLedger) MarkRunning(effectID string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	record, exists := l.records[effectID]
	if !exists {
		return fmt.Errorf("effect record not found: %s", effectID)
	}
	if record.Status == EffectLedgerPending || record.Status == EffectLedgerDeferred {
		record.Status = EffectLedgerRunning
		record.Attempts++
		record.LastAttemptAt = time.Now().UTC()
		l.records[effectID] = record
		return nil
	}
	return fmt.Errorf("effect transition blocked: %s from %s to %s", effectID, record.Status, EffectLedgerRunning)
}

// MarkDone transitions effect status to done and stores the final result.
func (l *EffectLedger) MarkDone(effectID string, result EffectResult) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	record, exists := l.records[effectID]
	if !exists {
		return fmt.Errorf("effect record not found: %s", effectID)
	}
	if record.Status != EffectLedgerRunning {
		return fmt.Errorf("effect transition blocked: %s from %s to %s", effectID, record.Status, EffectLedgerDone)
	}

	resultCopy := result
	record.Status = EffectLedgerDone
	record.Result = &resultCopy
	if !result.Timestamp.IsZero() {
		record.LastAttemptAt = result.Timestamp
	}
	l.records[effectID] = record
	return nil
}

// MarkFailed transitions effect status to failed and stores the final result.
func (l *EffectLedger) MarkFailed(effectID string, result EffectResult) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	record, exists := l.records[effectID]
	if !exists {
		return fmt.Errorf("effect record not found: %s", effectID)
	}
	if record.Status != EffectLedgerRunning {
		return fmt.Errorf("effect transition blocked: %s from %s to %s", effectID, record.Status, EffectLedgerFailed)
	}

	resultCopy := result
	record.Status = EffectLedgerFailed
	record.Result = &resultCopy
	if !result.Timestamp.IsZero() {
		record.LastAttemptAt = result.Timestamp
	}
	l.records[effectID] = record
	return nil
}

// Pending returns pending effects.
func (l *EffectLedger) Pending() []EffectLedgerRecord {
	l.mu.RLock()
	defer l.mu.RUnlock()

	out := make([]EffectLedgerRecord, 0, len(l.records))
	for _, effectID := range l.order {
		record := l.records[effectID]
		if record.Status == EffectLedgerPending {
			out = append(out, cloneLedgerRecord(record))
		}
	}
	return out
}

// InFlight returns running effects.
func (l *EffectLedger) InFlight() []EffectLedgerRecord {
	l.mu.RLock()
	defer l.mu.RUnlock()

	out := make([]EffectLedgerRecord, 0, len(l.records))
	for _, effectID := range l.order {
		record := l.records[effectID]
		if record.Status == EffectLedgerRunning {
			out = append(out, cloneLedgerRecord(record))
		}
	}
	return out
}

// All returns every record in insertion order.
func (l *EffectLedger) All() []EffectLedgerRecord {
	l.mu.RLock()
	defer l.mu.RUnlock()

	out := make([]EffectLedgerRecord, 0, len(l.records))
	for _, effectID := range l.order {
		record := l.records[effectID]
		out = append(out, cloneLedgerRecord(record))
	}
	return out
}

func cloneLedgerRecord(in EffectLedgerRecord) EffectLedgerRecord {
	out := in
	if in.Result != nil {
		resultCopy := *in.Result
		out.Result = &resultCopy
	}
	if in.Effect.Payload != nil {
		payloadCopy := append([]byte(nil), in.Effect.Payload...)
		out.Effect.Payload = payloadCopy
	}
	return out
}
