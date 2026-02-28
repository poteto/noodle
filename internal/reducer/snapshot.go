package reducer

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/poteto/noodle/internal/filex"
	"github.com/poteto/noodle/internal/state"
)

// DurableSnapshot is the canonical persisted form used by the reducer loop
// checkpoint step in the crash-consistency protocol.
type DurableSnapshot struct {
	State        state.State          `json:"state"`
	EffectLedger []EffectLedgerRecord `json:"effect_ledger"`
	GeneratedAt  time.Time            `json:"generated_at"`
}

// BuildSnapshot materializes state + embedded effect ledger into one payload.
func BuildSnapshot(canonical state.State, ledger *EffectLedger, generatedAt time.Time) DurableSnapshot {
	records := []EffectLedgerRecord{}
	if ledger != nil {
		records = ledger.All()
	}
	return DurableSnapshot{
		State:        canonical,
		EffectLedger: records,
		GeneratedAt:  generatedAt,
	}
}

// WriteSnapshotAtomic atomically persists canonical state and effect ledger
// together (temp file + rename) to keep reducer recovery deterministic.
func WriteSnapshotAtomic(path string, snapshot DurableSnapshot) error {
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("encode reducer snapshot: %w", err)
	}
	if err := filex.WriteFileAtomic(path, append(data, '\n')); err != nil {
		return fmt.Errorf("persist reducer snapshot: %w", err)
	}
	return nil
}
