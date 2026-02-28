package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/poteto/noodle/internal/statever"
)

// fullState returns a State with active and completed orders for testing.
func fullState() State {
	now := time.Date(2026, 2, 28, 12, 0, 0, 0, time.UTC)
	exitZero := 0
	return State{
		Orders: map[string]OrderNode{
			"order-1": {
				OrderID:   "order-1",
				Status:    OrderActive,
				CreatedAt: now,
				UpdatedAt: now,
				Metadata:  map[string]string{"key": "value"},
				Stages: []StageNode{
					{
						StageIndex: 0,
						Status:     StageCompleted,
						Skill:      "lint",
						Runtime:    "node",
						Group:      "a",
						Attempts: []AttemptNode{
							{
								AttemptID:    "att-1",
								SessionID:    "sess-1",
								Status:       AttemptCompleted,
								StartedAt:    now,
								CompletedAt:  now.Add(time.Minute),
								ExitCode:     &exitZero,
								WorktreeName: "wt-1",
							},
						},
					},
					{
						StageIndex: 1,
						Status:     StageRunning,
						Skill:      "test",
						Runtime:    "go",
						Group:      "b",
						Attempts: []AttemptNode{
							{
								AttemptID:    "att-2",
								SessionID:    "sess-2",
								Status:       AttemptRunning,
								StartedAt:    now,
								WorktreeName: "wt-2",
							},
						},
					},
				},
			},
			"order-2": {
				OrderID:   "order-2",
				Status:    OrderCompleted,
				CreatedAt: now.Add(-time.Hour),
				UpdatedAt: now,
				Stages: []StageNode{
					{
						StageIndex: 0,
						Status:     StageCompleted,
						Skill:      "build",
						Runtime:    "go",
						Attempts: []AttemptNode{
							{
								AttemptID:   "att-3",
								SessionID:   "sess-3",
								Status:      AttemptCompleted,
								StartedAt:   now.Add(-time.Hour),
								CompletedAt: now.Add(-30 * time.Minute),
								ExitCode:    &exitZero,
							},
						},
					},
				},
			},
		},
		Mode:          RunModeAuto,
		SchemaVersion: statever.Current,
		LastEventID:   "evt-42",
	}
}

func TestSerializationRoundTrip(t *testing.T) {
	s := fullState()
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got State
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !reflect.DeepEqual(s, got) {
		t.Fatalf("round-trip mismatch:\n  original: %+v\n  decoded:  %+v", s, got)
	}
}

func TestPersistenceRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	s := fullState()
	if err := s.Persist(path); err != nil {
		t.Fatalf("persist: %v", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !reflect.DeepEqual(s, got) {
		t.Fatalf("persistence round-trip mismatch:\n  original: %+v\n  loaded:   %+v", s, got)
	}
}

func TestLoadMissingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "does-not-exist.json")

	got, err := Load(path)
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	if got.Orders != nil || got.Mode != "" || got.LastEventID != "" {
		t.Fatalf("expected zero State, got: %+v", got)
	}
}

func TestLoadEmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.json")
	if err := os.WriteFile(path, []byte("  \n"), 0o644); err != nil {
		t.Fatalf("write empty file: %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("expected no error for empty file, got: %v", err)
	}
	if got.Orders != nil {
		t.Fatalf("expected zero State for empty file, got: %+v", got)
	}
}

func TestLoadCorruptedFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "corrupt.json")
	if err := os.WriteFile(path, []byte("{not valid json"), 0o644); err != nil {
		t.Fatalf("write corrupt file: %v", err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for corrupted file, got nil")
	}
}

// --- Validate tests ---

func TestValidateEmptyState(t *testing.T) {
	s := State{}
	if err := s.Validate(); err != nil {
		t.Fatalf("empty state should validate, got: %v", err)
	}
}

func TestValidateCompletedOrders(t *testing.T) {
	s := State{
		Orders: map[string]OrderNode{
			"done": {
				OrderID: "done",
				Status:  OrderCompleted,
				Stages: []StageNode{
					{StageIndex: 0, Status: StageCompleted},
				},
			},
		},
	}
	if err := s.Validate(); err != nil {
		t.Fatalf("completed order should validate, got: %v", err)
	}
}

func TestValidateActivePipeline(t *testing.T) {
	s := fullState()
	if err := s.Validate(); err != nil {
		t.Fatalf("valid active pipeline should validate, got: %v", err)
	}
}

func TestValidateActiveOrderAllTerminalStages(t *testing.T) {
	s := State{
		Orders: map[string]OrderNode{
			"bad": {
				OrderID: "bad",
				Status:  OrderActive,
				Stages: []StageNode{
					{StageIndex: 0, Status: StageCompleted},
					{StageIndex: 1, Status: StageFailed},
				},
			},
		},
	}
	err := s.Validate()
	if err == nil {
		t.Fatal("expected error for active order with all terminal stages")
	}
	if got := err.Error(); got != `order "bad" has status active but all stages are terminal` {
		t.Fatalf("unexpected error: %s", got)
	}
}

func TestValidateActiveOrderNoStages(t *testing.T) {
	s := State{
		Orders: map[string]OrderNode{
			"empty": {
				OrderID: "empty",
				Status:  OrderActive,
				Stages:  []StageNode{},
			},
		},
	}
	err := s.Validate()
	if err == nil {
		t.Fatal("expected error for active order with no stages")
	}
}

func TestValidateRunningStageNoRunningAttempts(t *testing.T) {
	s := State{
		Orders: map[string]OrderNode{
			"x": {
				OrderID: "x",
				Status:  OrderActive,
				Stages: []StageNode{
					{
						StageIndex: 0,
						Status:     StageRunning,
						Attempts: []AttemptNode{
							{AttemptID: "a1", Status: AttemptFailed},
						},
					},
					{StageIndex: 1, Status: StagePending},
				},
			},
		},
	}
	err := s.Validate()
	if err == nil {
		t.Fatal("expected error for running stage with no running attempts")
	}
	if got := err.Error(); got != `order "x" stage 0 has status running but no attempt is running` {
		t.Fatalf("unexpected error: %s", got)
	}
}

func TestValidateRunningStageNoAttempts(t *testing.T) {
	s := State{
		Orders: map[string]OrderNode{
			"y": {
				OrderID: "y",
				Status:  OrderActive,
				Stages: []StageNode{
					{StageIndex: 0, Status: StageRunning},
					{StageIndex: 1, Status: StagePending},
				},
			},
		},
	}
	err := s.Validate()
	if err == nil {
		t.Fatal("expected error for running stage with nil attempts")
	}
}

func TestValidateDuplicateAttemptIDs(t *testing.T) {
	s := State{
		Orders: map[string]OrderNode{
			"o1": {
				OrderID: "o1",
				Status:  OrderCompleted,
				Stages: []StageNode{
					{
						StageIndex: 0,
						Status:     StageCompleted,
						Attempts: []AttemptNode{
							{AttemptID: "dup", Status: AttemptCompleted},
						},
					},
				},
			},
			"o2": {
				OrderID: "o2",
				Status:  OrderCompleted,
				Stages: []StageNode{
					{
						StageIndex: 0,
						Status:     StageCompleted,
						Attempts: []AttemptNode{
							{AttemptID: "dup", Status: AttemptCompleted},
						},
					},
				},
			},
		},
	}
	err := s.Validate()
	if err == nil {
		t.Fatal("expected error for duplicate attempt IDs")
	}
	if got := err.Error(); !contains(got, "attempt \"dup\" appears in both") {
		t.Fatalf("unexpected error: %s", got)
	}
}

func TestValidateNonSequentialStageIndexes(t *testing.T) {
	s := State{
		Orders: map[string]OrderNode{
			"z": {
				OrderID: "z",
				Status:  OrderCompleted,
				Stages: []StageNode{
					{StageIndex: 0, Status: StageCompleted},
					{StageIndex: 5, Status: StageCompleted},
				},
			},
		},
	}
	err := s.Validate()
	if err == nil {
		t.Fatal("expected error for non-sequential stage indexes")
	}
	if got := err.Error(); got != `order "z" stage 1 has index 5 (expected 1)` {
		t.Fatalf("unexpected error: %s", got)
	}
}

// --- Index tests ---

func TestOrderBusyIndex(t *testing.T) {
	s := fullState()
	idx := s.OrderBusyIndex()

	// order-1 has stage 1 running.
	if got, ok := idx["order-1"]; !ok || got != 1 {
		t.Fatalf("expected order-1 -> 1, got %d (ok=%v)", got, ok)
	}
	// order-2 is completed, no busy stages.
	if _, ok := idx["order-2"]; ok {
		t.Fatal("order-2 should not appear in busy index")
	}
}

func TestOrderBusyIndexMultipleBusyStages(t *testing.T) {
	s := State{
		Orders: map[string]OrderNode{
			"multi": {
				OrderID: "multi",
				Status:  OrderActive,
				Stages: []StageNode{
					{StageIndex: 0, Status: StageDispatching},
					{StageIndex: 1, Status: StageRunning, Attempts: []AttemptNode{{AttemptID: "a1", Status: AttemptRunning}}},
				},
			},
		},
	}
	idx := s.OrderBusyIndex()
	// First busy stage wins.
	if got := idx["multi"]; got != 0 {
		t.Fatalf("expected first busy stage (0), got %d", got)
	}
}

func TestOrderBusyIndexEmptyState(t *testing.T) {
	s := State{}
	idx := s.OrderBusyIndex()
	if len(idx) != 0 {
		t.Fatalf("expected empty index, got %d entries", len(idx))
	}
}

func TestAttemptBySessionIndex(t *testing.T) {
	s := fullState()
	idx := s.AttemptBySessionIndex()

	if a, ok := idx["sess-1"]; !ok {
		t.Fatal("sess-1 not found")
	} else if a.AttemptID != "att-1" {
		t.Fatalf("expected att-1, got %s", a.AttemptID)
	}

	if a, ok := idx["sess-2"]; !ok {
		t.Fatal("sess-2 not found")
	} else if a.AttemptID != "att-2" {
		t.Fatalf("expected att-2, got %s", a.AttemptID)
	}

	if a, ok := idx["sess-3"]; !ok {
		t.Fatal("sess-3 not found")
	} else if a.AttemptID != "att-3" {
		t.Fatalf("expected att-3, got %s", a.AttemptID)
	}
}

func TestAttemptBySessionIndexEmptySessionID(t *testing.T) {
	s := State{
		Orders: map[string]OrderNode{
			"o": {
				OrderID: "o",
				Status:  OrderCompleted,
				Stages: []StageNode{
					{
						StageIndex: 0,
						Status:     StageCompleted,
						Attempts: []AttemptNode{
							{AttemptID: "a", SessionID: "", Status: AttemptCompleted},
						},
					},
				},
			},
		},
	}
	idx := s.AttemptBySessionIndex()
	if len(idx) != 0 {
		t.Fatalf("empty session IDs should be excluded, got %d entries", len(idx))
	}
}

func TestAttemptBySessionIndexNilAttempts(t *testing.T) {
	s := State{
		Orders: map[string]OrderNode{
			"o": {
				OrderID: "o",
				Status:  OrderCompleted,
				Stages: []StageNode{
					{StageIndex: 0, Status: StageCompleted, Attempts: nil},
				},
			},
		},
	}
	idx := s.AttemptBySessionIndex()
	if len(idx) != 0 {
		t.Fatalf("nil attempts should produce empty index, got %d entries", len(idx))
	}
}

func TestPendingEffectIndex(t *testing.T) {
	s := fullState()
	idx := s.PendingEffectIndex()
	if len(idx) != 0 {
		t.Fatalf("stub should return empty map, got %d entries", len(idx))
	}
}

// --- Edge cases ---

func TestValidateEmptyOrdersMap(t *testing.T) {
	s := State{Orders: map[string]OrderNode{}}
	if err := s.Validate(); err != nil {
		t.Fatalf("empty orders map should validate, got: %v", err)
	}
}

func TestValidateTerminalStates(t *testing.T) {
	for _, status := range []OrderLifecycleStatus{OrderCompleted, OrderFailed, OrderCancelled} {
		s := State{
			Orders: map[string]OrderNode{
				"t": {
					OrderID: "t",
					Status:  status,
					Stages: []StageNode{
						{StageIndex: 0, Status: StageCompleted},
					},
				},
			},
		}
		if err := s.Validate(); err != nil {
			t.Fatalf("terminal order status %q should validate, got: %v", status, err)
		}
	}
}

func TestValidatePendingOrderAllTerminalStages(t *testing.T) {
	// Only active orders enforce the non-terminal stage invariant.
	s := State{
		Orders: map[string]OrderNode{
			"p": {
				OrderID: "p",
				Status:  OrderPending,
				Stages: []StageNode{
					{StageIndex: 0, Status: StageCompleted},
				},
			},
		},
	}
	if err := s.Validate(); err != nil {
		t.Fatalf("pending order with terminal stages should validate, got: %v", err)
	}
}

func TestOrderBusyIndexAllBusyStatuses(t *testing.T) {
	for i, status := range []StageLifecycleStatus{StageDispatching, StageRunning, StageMerging, StageReview} {
		s := State{
			Orders: map[string]OrderNode{
				"o": {
					OrderID: "o",
					Status:  OrderActive,
					Stages: []StageNode{
						{StageIndex: 0, Status: status},
						{StageIndex: 1, Status: StagePending},
					},
				},
			},
		}
		idx := s.OrderBusyIndex()
		if got, ok := idx["o"]; !ok || got != 0 {
			t.Fatalf("status %q (case %d): expected o -> 0, got %d (ok=%v)", status, i, got, ok)
		}
	}
}

func TestSerializationJSONFieldNames(t *testing.T) {
	s := State{
		Orders:        map[string]OrderNode{},
		Mode:          RunModeAuto,
		SchemaVersion: 1,
		LastEventID:   "e1",
	}
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	raw := string(data)
	for _, field := range []string{`"orders"`, `"mode"`, `"schema_version"`, `"last_event_id"`} {
		if !contains(raw, field) {
			t.Errorf("expected JSON field %s in output: %s", field, raw)
		}
	}
}

func TestPersistCreatesParentDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "deep", "state.json")

	s := State{Mode: RunModeManual}
	if err := s.Persist(path); err != nil {
		t.Fatalf("persist to nested path: %v", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("load from nested path: %v", err)
	}
	if got.Mode != RunModeManual {
		t.Fatalf("expected mode manual, got %q", got.Mode)
	}
}

func TestValidatePartiallyActivePipeline(t *testing.T) {
	s := State{
		Orders: map[string]OrderNode{
			"partial": {
				OrderID: "partial",
				Status:  OrderActive,
				Stages: []StageNode{
					{StageIndex: 0, Status: StageCompleted},
					{StageIndex: 1, Status: StagePending},
					{StageIndex: 2, Status: StageSkipped},
				},
			},
		},
	}
	if err := s.Validate(); err != nil {
		t.Fatalf("partially active pipeline should validate, got: %v", err)
	}
}

// contains reports whether s contains substr.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
