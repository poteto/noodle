package mode

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/poteto/noodle/internal/state"
)

func TestGateBehaviorMatrix(t *testing.T) {
	gate := ModeGate{}

	tests := []struct {
		action  string
		mode    state.RunMode
		allowed bool
		reason  string
	}{
		// Schedule
		{ActionSchedule, state.RunModeAuto, true, ""},
		{ActionSchedule, state.RunModeSupervised, true, ""},
		{ActionSchedule, state.RunModeManual, false, "manual mode requires explicit scheduling"},

		// Dispatch
		{ActionDispatch, state.RunModeAuto, true, ""},
		{ActionDispatch, state.RunModeSupervised, true, ""},
		{ActionDispatch, state.RunModeManual, false, "manual mode requires explicit dispatch"},

		// AutoMerge
		{ActionAutoMerge, state.RunModeAuto, true, ""},
		{ActionAutoMerge, state.RunModeSupervised, false, "supervised mode requires merge approval"},
		{ActionAutoMerge, state.RunModeManual, false, "manual mode requires merge approval"},
	}

	for _, tt := range tests {
		t.Run(tt.action+"_"+string(tt.mode), func(t *testing.T) {
			var got bool
			switch tt.action {
			case ActionSchedule:
				got = gate.CanSchedule(tt.mode)
			case ActionDispatch:
				got = gate.CanDispatch(tt.mode)
			case ActionAutoMerge:
				got = gate.CanAutoMerge(tt.mode)
			}
			if got != tt.allowed {
				t.Errorf("Can%s(%s) = %v, want %v", tt.action, tt.mode, got, tt.allowed)
			}

			reason := gate.BlockedReason(tt.mode, tt.action)
			if reason != tt.reason {
				t.Errorf("BlockedReason(%s, %s) = %q, want %q", tt.mode, tt.action, reason, tt.reason)
			}
		})
	}
}

func TestBlockedReasonDescriptiveMessages(t *testing.T) {
	gate := ModeGate{}

	// Verify that all blocked actions produce non-empty, descriptive messages.
	blocked := []struct {
		mode   state.RunMode
		action string
	}{
		{state.RunModeManual, ActionSchedule},
		{state.RunModeManual, ActionDispatch},
		{state.RunModeSupervised, ActionAutoMerge},
		{state.RunModeManual, ActionAutoMerge},
	}

	for _, tt := range blocked {
		reason := gate.BlockedReason(tt.mode, tt.action)
		if reason == "" {
			t.Errorf("BlockedReason(%s, %s) returned empty string for blocked action", tt.mode, tt.action)
		}
	}
}

func TestBlockedReasonUnknownAction(t *testing.T) {
	gate := ModeGate{}
	reason := gate.BlockedReason(state.RunModeManual, "unknown_action")
	if reason != "" {
		t.Errorf("BlockedReason for unknown action = %q, want empty", reason)
	}
}

func TestNewModeState(t *testing.T) {
	ms := NewModeState(state.RunModeSupervised)
	if ms.RequestedMode != state.RunModeSupervised {
		t.Errorf("RequestedMode = %q, want %q", ms.RequestedMode, state.RunModeSupervised)
	}
	if ms.EffectiveMode != state.RunModeSupervised {
		t.Errorf("EffectiveMode = %q, want %q", ms.EffectiveMode, state.RunModeSupervised)
	}
	if ms.Epoch != 0 {
		t.Errorf("Epoch = %d, want 0", ms.Epoch)
	}
	if ms.Transitions == nil {
		t.Error("Transitions is nil, want empty slice")
	}
	if len(ms.Transitions) != 0 {
		t.Errorf("len(Transitions) = %d, want 0", len(ms.Transitions))
	}
}

func TestTransitionModeIncrementsEpoch(t *testing.T) {
	now := time.Date(2026, 2, 28, 12, 0, 0, 0, time.UTC)
	ms := NewModeState(state.RunModeAuto)

	ms = TransitionMode(ms, state.RunModeSupervised, "user:alice", "escalation", now)
	if ms.Epoch != 1 {
		t.Errorf("Epoch = %d, want 1", ms.Epoch)
	}
	if ms.EffectiveMode != state.RunModeSupervised {
		t.Errorf("EffectiveMode = %q, want %q", ms.EffectiveMode, state.RunModeSupervised)
	}
	if ms.RequestedMode != state.RunModeSupervised {
		t.Errorf("RequestedMode = %q, want %q", ms.RequestedMode, state.RunModeSupervised)
	}
}

func TestTransitionModeMultipleMaintainsHistory(t *testing.T) {
	now := time.Date(2026, 2, 28, 12, 0, 0, 0, time.UTC)
	ms := NewModeState(state.RunModeAuto)

	ms = TransitionMode(ms, state.RunModeSupervised, "user:alice", "escalation", now)
	ms = TransitionMode(ms, state.RunModeManual, "user:bob", "incident", now.Add(time.Minute))
	ms = TransitionMode(ms, state.RunModeAuto, "system", "resolved", now.Add(2*time.Minute))

	if ms.Epoch != 3 {
		t.Errorf("Epoch = %d, want 3", ms.Epoch)
	}
	if len(ms.Transitions) != 3 {
		t.Fatalf("len(Transitions) = %d, want 3", len(ms.Transitions))
	}

	// Verify transition order.
	if ms.Transitions[0].FromMode != state.RunModeAuto || ms.Transitions[0].ToMode != state.RunModeSupervised {
		t.Errorf("Transition[0]: %s -> %s, want auto -> supervised", ms.Transitions[0].FromMode, ms.Transitions[0].ToMode)
	}
	if ms.Transitions[1].FromMode != state.RunModeSupervised || ms.Transitions[1].ToMode != state.RunModeManual {
		t.Errorf("Transition[1]: %s -> %s, want supervised -> manual", ms.Transitions[1].FromMode, ms.Transitions[1].ToMode)
	}
	if ms.Transitions[2].FromMode != state.RunModeManual || ms.Transitions[2].ToMode != state.RunModeAuto {
		t.Errorf("Transition[2]: %s -> %s, want manual -> auto", ms.Transitions[2].FromMode, ms.Transitions[2].ToMode)
	}
	if ms.Transitions[2].RequestedBy != "system" {
		t.Errorf("Transition[2].RequestedBy = %q, want %q", ms.Transitions[2].RequestedBy, "system")
	}
}

func TestTransitionToSameModeIncrementsEpoch(t *testing.T) {
	now := time.Date(2026, 2, 28, 12, 0, 0, 0, time.UTC)
	ms := NewModeState(state.RunModeAuto)

	ms = TransitionMode(ms, state.RunModeAuto, "user:alice", "refresh", now)
	if ms.Epoch != 1 {
		t.Errorf("Epoch = %d, want 1 after same-mode transition", ms.Epoch)
	}
	if len(ms.Transitions) != 1 {
		t.Errorf("len(Transitions) = %d, want 1", len(ms.Transitions))
	}
	if ms.Transitions[0].FromMode != state.RunModeAuto || ms.Transitions[0].ToMode != state.RunModeAuto {
		t.Errorf("Transition: %s -> %s, want auto -> auto", ms.Transitions[0].FromMode, ms.Transitions[0].ToMode)
	}
}

func TestTransitionHistoryCapped(t *testing.T) {
	now := time.Date(2026, 2, 28, 12, 0, 0, 0, time.UTC)
	ms := NewModeState(state.RunModeAuto)

	modes := []state.RunMode{state.RunModeAuto, state.RunModeSupervised, state.RunModeManual}
	for i := 0; i < 60; i++ {
		ms = TransitionMode(ms, modes[i%len(modes)], "test", "churn", now.Add(time.Duration(i)*time.Second))
	}

	if len(ms.Transitions) != maxTransitionHistory {
		t.Errorf("len(Transitions) = %d, want %d", len(ms.Transitions), maxTransitionHistory)
	}
	if ms.Epoch != 60 {
		t.Errorf("Epoch = %d, want 60", ms.Epoch)
	}
	// The oldest retained transition should be epoch 11 (transitions 1..60, capped to last 50).
	if ms.Transitions[0].Epoch != 11 {
		t.Errorf("oldest Transition.Epoch = %d, want 11", ms.Transitions[0].Epoch)
	}
}

func TestTransitionDoesNotMutateOriginal(t *testing.T) {
	now := time.Date(2026, 2, 28, 12, 0, 0, 0, time.UTC)
	original := NewModeState(state.RunModeAuto)
	_ = TransitionMode(original, state.RunModeManual, "user", "test", now)

	if original.Epoch != 0 {
		t.Errorf("original.Epoch = %d, want 0 (mutation detected)", original.Epoch)
	}
	if len(original.Transitions) != 0 {
		t.Errorf("original.Transitions length = %d, want 0 (mutation detected)", len(original.Transitions))
	}
}

func TestEpochValidation(t *testing.T) {
	tests := []struct {
		name         string
		effectEpoch  ModeEpoch
		currentEpoch ModeEpoch
		want         EpochResult
	}{
		{"same epoch is valid", 5, 5, EpochValid},
		{"older epoch is stale", 3, 5, EpochStale},
		{"zero epoch same", 0, 0, EpochValid},
		{"zero effect vs nonzero current", 0, 1, EpochStale},
		{"future epoch is deferred", 7, 5, EpochDeferred},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateEpoch(tt.effectEpoch, tt.currentEpoch)
			if got != tt.want {
				t.Errorf("ValidateEpoch(%d, %d) = %q, want %q", tt.effectEpoch, tt.currentEpoch, got, tt.want)
			}
		})
	}
}

func TestStampEffect(t *testing.T) {
	stamped := StampEffect(42, "effect-abc")
	if stamped.Epoch != 42 {
		t.Errorf("Epoch = %d, want 42", stamped.Epoch)
	}
	if stamped.EffectID != "effect-abc" {
		t.Errorf("EffectID = %q, want %q", stamped.EffectID, "effect-abc")
	}
}

func TestModeStateSerializationRoundTrip(t *testing.T) {
	now := time.Date(2026, 2, 28, 12, 0, 0, 0, time.UTC)
	ms := NewModeState(state.RunModeAuto)
	ms = TransitionMode(ms, state.RunModeSupervised, "user:alice", "escalation", now)
	ms = TransitionMode(ms, state.RunModeManual, "user:bob", "incident", now.Add(time.Minute))

	data, err := json.Marshal(ms)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var restored ModeState
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if restored.RequestedMode != ms.RequestedMode {
		t.Errorf("RequestedMode = %q, want %q", restored.RequestedMode, ms.RequestedMode)
	}
	if restored.EffectiveMode != ms.EffectiveMode {
		t.Errorf("EffectiveMode = %q, want %q", restored.EffectiveMode, ms.EffectiveMode)
	}
	if restored.Epoch != ms.Epoch {
		t.Errorf("Epoch = %d, want %d", restored.Epoch, ms.Epoch)
	}
	if len(restored.Transitions) != len(ms.Transitions) {
		t.Fatalf("len(Transitions) = %d, want %d", len(restored.Transitions), len(ms.Transitions))
	}

	for i, tr := range restored.Transitions {
		orig := ms.Transitions[i]
		if tr.FromMode != orig.FromMode || tr.ToMode != orig.ToMode || tr.Epoch != orig.Epoch {
			t.Errorf("Transition[%d] mismatch: got %+v, want %+v", i, tr, orig)
		}
		if tr.RequestedBy != orig.RequestedBy || tr.Reason != orig.Reason {
			t.Errorf("Transition[%d] metadata mismatch: got by=%q reason=%q, want by=%q reason=%q",
				i, tr.RequestedBy, tr.Reason, orig.RequestedBy, orig.Reason)
		}
		if !tr.AppliedAt.Equal(orig.AppliedAt) {
			t.Errorf("Transition[%d] AppliedAt = %v, want %v", i, tr.AppliedAt, orig.AppliedAt)
		}
	}
}

func TestModeStateJSONSnakeCase(t *testing.T) {
	ms := NewModeState(state.RunModeAuto)
	data, err := json.Marshal(ms)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal to map: %v", err)
	}

	expectedKeys := []string{"requested_mode", "effective_mode", "epoch", "transitions"}
	for _, key := range expectedKeys {
		if _, ok := raw[key]; !ok {
			t.Errorf("JSON missing expected snake_case key %q; got keys: %v", key, keysOf(raw))
		}
	}
}

func TestModeTransitionRecordJSONSnakeCase(t *testing.T) {
	tr := ModeTransitionRecord{
		FromMode:    state.RunModeAuto,
		ToMode:      state.RunModeManual,
		Epoch:       1,
		RequestedBy: "test",
		Reason:      "test",
		AppliedAt:   time.Now(),
	}
	data, err := json.Marshal(tr)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal to map: %v", err)
	}

	expectedKeys := []string{"from_mode", "to_mode", "epoch", "requested_by", "reason", "applied_at"}
	for _, key := range expectedKeys {
		if _, ok := raw[key]; !ok {
			t.Errorf("JSON missing expected snake_case key %q; got keys: %v", key, keysOf(raw))
		}
	}
}

func TestStampedEffectJSONSnakeCase(t *testing.T) {
	se := StampedEffect{EffectID: "e1", Epoch: 5}
	data, err := json.Marshal(se)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal to map: %v", err)
	}

	expectedKeys := []string{"effect_id", "epoch"}
	for _, key := range expectedKeys {
		if _, ok := raw[key]; !ok {
			t.Errorf("JSON missing expected snake_case key %q; got keys: %v", key, keysOf(raw))
		}
	}
}

func TestEmptyTransitionsList(t *testing.T) {
	ms := NewModeState(state.RunModeManual)

	// Validate via epoch — should still work at epoch 0.
	result := ValidateEpoch(0, ms.Epoch)
	if result != EpochValid {
		t.Errorf("ValidateEpoch(0, 0) = %q, want %q", result, EpochValid)
	}

	// Serialize/deserialize empty transitions.
	data, err := json.Marshal(ms)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var restored ModeState
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if restored.Transitions == nil || len(restored.Transitions) != 0 {
		t.Errorf("restored.Transitions = %v, want empty non-nil slice", restored.Transitions)
	}
}

func keysOf(m map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
