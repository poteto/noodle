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
		Mode:      RunModeAuto,
		ModeEpoch: 2,
		ModeTransitions: []ModeTransitionRecord{
			{
				FromMode:    RunModeManual,
				ToMode:      RunModeSupervised,
				Epoch:       1,
				RequestedBy: "user:alice",
				Reason:      "escalation",
				AppliedAt:   now.Add(-time.Hour),
			},
			{
				FromMode:    RunModeSupervised,
				ToMode:      RunModeAuto,
				Epoch:       2,
				RequestedBy: "system",
				Reason:      "resolved",
				AppliedAt:   now.Add(-30 * time.Minute),
			},
		},
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

// --- Mode epoch tests ---

func TestModeTransitionRecordJSONFields(t *testing.T) {
	now := time.Date(2026, 2, 28, 12, 0, 0, 0, time.UTC)
	r := ModeTransitionRecord{
		FromMode:    RunModeAuto,
		ToMode:      RunModeManual,
		Epoch:       3,
		RequestedBy: "user:bob",
		Reason:      "incident",
		AppliedAt:   now,
	}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	raw := string(data)
	for _, field := range []string{`"from_mode"`, `"to_mode"`, `"epoch"`, `"requested_by"`, `"reason"`, `"applied_at"`} {
		if !contains(raw, field) {
			t.Errorf("expected JSON field %s in output: %s", field, raw)
		}
	}
}

func TestModeEpochInState(t *testing.T) {
	s := fullState()
	if s.ModeEpoch != 2 {
		t.Fatalf("expected ModeEpoch 2, got %d", s.ModeEpoch)
	}
	if len(s.ModeTransitions) != 2 {
		t.Fatalf("expected 2 transitions, got %d", len(s.ModeTransitions))
	}
	if s.ModeTransitions[0].FromMode != RunModeManual || s.ModeTransitions[0].ToMode != RunModeSupervised {
		t.Fatalf("transition[0] mismatch: %s -> %s", s.ModeTransitions[0].FromMode, s.ModeTransitions[0].ToMode)
	}
	if s.ModeTransitions[1].FromMode != RunModeSupervised || s.ModeTransitions[1].ToMode != RunModeAuto {
		t.Fatalf("transition[1] mismatch: %s -> %s", s.ModeTransitions[1].FromMode, s.ModeTransitions[1].ToMode)
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
		Orders:          map[string]OrderNode{},
		Mode:            RunModeAuto,
		ModeEpoch:       1,
		ModeTransitions: []ModeTransitionRecord{},
		SchemaVersion:   1,
		LastEventID:     "e1",
	}
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	raw := string(data)
	for _, field := range []string{`"orders"`, `"mode"`, `"mode_epoch"`, `"mode_transitions"`, `"schema_version"`, `"last_event_id"`} {
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

// --- Clone tests ---

func TestCloneImmutabilityMetadata(t *testing.T) {
	original := fullState()
	clone := original.Clone()

	// Mutate clone's metadata.
	clone.Orders["order-1"] = func() OrderNode {
		o := clone.Orders["order-1"]
		o.Metadata["new-key"] = "new-value"
		return o
	}()

	// Original must be unchanged.
	if _, ok := original.Orders["order-1"].Metadata["new-key"]; ok {
		t.Fatal("mutating clone metadata affected the original")
	}
}

func TestCloneImmutabilityExitCode(t *testing.T) {
	original := fullState()
	clone := original.Clone()

	// Mutate clone's exit code pointer.
	cloneOrder := clone.Orders["order-1"]
	*cloneOrder.Stages[0].Attempts[0].ExitCode = 42

	// Original must be unchanged.
	origCode := original.Orders["order-1"].Stages[0].Attempts[0].ExitCode
	if origCode == nil || *origCode != 0 {
		t.Fatalf("mutating clone exit code affected the original: got %v", origCode)
	}
}

func TestCloneImmutabilityModeTransitions(t *testing.T) {
	original := fullState()
	clone := original.Clone()

	// Append to clone's mode transitions.
	clone.ModeTransitions = append(clone.ModeTransitions, ModeTransitionRecord{
		FromMode: RunModeAuto,
		ToMode:   RunModeManual,
		Epoch:    99,
	})

	// Original must be unchanged.
	if len(original.ModeTransitions) != 2 {
		t.Fatalf("mutating clone mode transitions affected the original: got %d", len(original.ModeTransitions))
	}
}

func TestCloneImmutabilityStages(t *testing.T) {
	original := fullState()
	clone := original.Clone()

	// Append a stage to clone.
	cloneOrder := clone.Orders["order-1"]
	cloneOrder.Stages = append(cloneOrder.Stages, StageNode{StageIndex: 99})
	clone.Orders["order-1"] = cloneOrder

	// Original must be unchanged.
	if len(original.Orders["order-1"].Stages) != 2 {
		t.Fatalf("mutating clone stages affected the original: got %d", len(original.Orders["order-1"].Stages))
	}
}

func TestCloneNilOrders(t *testing.T) {
	original := State{Mode: RunModeAuto}
	clone := original.Clone()

	if clone.Orders != nil {
		t.Fatal("clone of nil orders should be nil")
	}
	if clone.Mode != RunModeAuto {
		t.Fatalf("expected mode auto, got %q", clone.Mode)
	}
}

// --- IsTerminal tests ---

func TestOrderLifecycleStatusIsTerminal(t *testing.T) {
	terminal := []OrderLifecycleStatus{OrderCompleted, OrderFailed, OrderCancelled}
	nonTerminal := []OrderLifecycleStatus{OrderPending, OrderActive, ""}

	for _, s := range terminal {
		if !s.IsTerminal() {
			t.Errorf("expected %q to be terminal", s)
		}
	}
	for _, s := range nonTerminal {
		if s.IsTerminal() {
			t.Errorf("expected %q to not be terminal", s)
		}
	}
}

func TestStageLifecycleStatusIsTerminal(t *testing.T) {
	terminal := []StageLifecycleStatus{StageCompleted, StageFailed, StageSkipped, StageCancelled}
	nonTerminal := []StageLifecycleStatus{StagePending, StageDispatching, StageRunning, StageMerging, StageReview, ""}

	for _, s := range terminal {
		if !s.IsTerminal() {
			t.Errorf("expected %q to be terminal", s)
		}
	}
	for _, s := range nonTerminal {
		if s.IsTerminal() {
			t.Errorf("expected %q to not be terminal", s)
		}
	}
}

// --- IsBusy tests ---

func TestStageLifecycleStatusIsBusy(t *testing.T) {
	busy := []StageLifecycleStatus{StageDispatching, StageRunning, StageMerging, StageReview}
	notBusy := []StageLifecycleStatus{StagePending, StageCompleted, StageFailed, StageSkipped, StageCancelled, ""}

	for _, s := range busy {
		if !s.IsBusy() {
			t.Errorf("expected %q to be busy", s)
		}
	}
	for _, s := range notBusy {
		if s.IsBusy() {
			t.Errorf("expected %q to not be busy", s)
		}
	}
}

func TestStagePendingIsNotBusy(t *testing.T) {
	// Explicit regression test: StagePending must NOT be classified as busy.
	// This protects dispatch capacity logic.
	if StagePending.IsBusy() {
		t.Fatal("StagePending classified as busy — this breaks dispatch capacity logic")
	}
}

// --- LookupStage tests ---

func TestLookupStageFound(t *testing.T) {
	s := fullState()
	order, stage, ok := s.LookupStage("order-1", 0)
	if !ok {
		t.Fatal("expected to find order-1 stage 0")
	}
	if order.OrderID != "order-1" {
		t.Fatalf("expected order-1, got %q", order.OrderID)
	}
	if stage.StageIndex != 0 {
		t.Fatalf("expected stage index 0, got %d", stage.StageIndex)
	}
	if stage.Skill != "lint" {
		t.Fatalf("expected skill lint, got %q", stage.Skill)
	}
}

func TestLookupStageNotFoundOrder(t *testing.T) {
	s := fullState()
	_, _, ok := s.LookupStage("nonexistent", 0)
	if ok {
		t.Fatal("expected not found for nonexistent order")
	}
}

func TestLookupStageNotFoundIndex(t *testing.T) {
	s := fullState()
	_, _, ok := s.LookupStage("order-1", 99)
	if ok {
		t.Fatal("expected not found for out-of-range stage index")
	}
}

func TestLookupStageNegativeIndex(t *testing.T) {
	s := fullState()
	_, _, ok := s.LookupStage("order-1", -1)
	if ok {
		t.Fatal("expected not found for negative stage index")
	}
}

func TestLookupStageNilOrders(t *testing.T) {
	s := State{}
	_, _, ok := s.LookupStage("any", 0)
	if ok {
		t.Fatal("expected not found for nil orders map")
	}
}

// --- ClonedExitCode tests ---

func TestClonedExitCodeNil(t *testing.T) {
	if got := ClonedExitCode(nil); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestClonedExitCodeValue(t *testing.T) {
	v := 42
	got := ClonedExitCode(&v)
	if got == nil || *got != 42 {
		t.Fatalf("expected 42, got %v", got)
	}
	// Mutate original, clone must be unaffected.
	v = 99
	if *got != 42 {
		t.Fatal("ClonedExitCode did not create independent copy")
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
