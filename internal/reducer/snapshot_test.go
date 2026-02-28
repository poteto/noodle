package reducer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/poteto/noodle/internal/state"
)

func TestBuildSnapshotIncludesLedger(t *testing.T) {
	ledger := NewEffectLedger()
	effect := Effect{EffectID: "event-1-effect-0", Type: EffectAck, Payload: []byte(`{"order_id":"o1"}`)}
	ledger.Record(effect)

	s := BuildSnapshot(state.State{Mode: state.RunModeManual}, ledger, fixtureTime())
	if s.State.Mode != state.RunModeManual {
		t.Fatalf("snapshot state mismatch: %+v", s.State)
	}
	if len(s.EffectLedger) != 1 {
		t.Fatalf("snapshot ledger count mismatch: %d", len(s.EffectLedger))
	}
	if s.EffectLedger[0].EffectID != effect.EffectID {
		t.Fatalf("snapshot ledger effect mismatch: %+v", s.EffectLedger[0])
	}
}

func TestWriteSnapshotAtomic(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "state.snapshot.json")
	snapshot := DurableSnapshot{
		State: state.State{Mode: state.RunModeAuto},
		EffectLedger: []EffectLedgerRecord{
			{EffectID: "event-1-effect-0", Status: EffectLedgerPending},
		},
		GeneratedAt: time.Date(2026, 2, 28, 20, 0, 0, 0, time.UTC),
	}

	if err := WriteSnapshotAtomic(path, snapshot); err != nil {
		t.Fatalf("write snapshot: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read snapshot file: %v", err)
	}

	text := string(data)
	for _, field := range []string{"\"state\"", "\"effect_ledger\"", "\"generated_at\""} {
		if !strings.Contains(text, field) {
			t.Fatalf("snapshot json missing field %s: %s", field, text)
		}
	}

	var decoded DurableSnapshot
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("decode snapshot json: %v", err)
	}
	if decoded.State.Mode != state.RunModeAuto {
		t.Fatalf("decoded snapshot mode mismatch: %q", decoded.State.Mode)
	}
	if len(decoded.EffectLedger) != 1 || decoded.EffectLedger[0].EffectID != "event-1-effect-0" {
		t.Fatalf("decoded snapshot ledger mismatch: %+v", decoded.EffectLedger)
	}
}
