package reducer

import (
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/poteto/noodle/internal/ingest"
	"github.com/poteto/noodle/internal/state"
	"github.com/poteto/noodle/internal/statever"
)

func TestReducerPureDeterministic(t *testing.T) {
	r := DefaultReducer()
	at := time.Date(2026, 2, 28, 18, 0, 0, 0, time.UTC)

	input := state.State{
		Orders: map[string]state.OrderNode{
			"order-1": {
				OrderID:   "order-1",
				Status:    state.OrderActive,
				CreatedAt: at,
				UpdatedAt: at,
				Stages: []state.StageNode{
					{StageIndex: 0, Status: state.StagePending},
				},
			},
		},
		Mode:          state.RunModeAuto,
		SchemaVersion: statever.Current,
	}

	event := mustStateEvent(7, ingest.EventDispatchRequested, map[string]any{
		"order_id":    "order-1",
		"stage_index": 0,
		"attempt_id":  "attempt-1",
	}, at)

	firstState, firstEffects, err := r(input, event)
	if err != nil {
		t.Fatalf("first reduce: %v", err)
	}
	secondState, secondEffects, err := r(input, event)
	if err != nil {
		t.Fatalf("second reduce: %v", err)
	}

	if !reflect.DeepEqual(firstState, secondState) {
		t.Fatalf("reducer output state changed across identical runs:\nfirst=%+v\nsecond=%+v", firstState, secondState)
	}
	if !reflect.DeepEqual(firstEffects, secondEffects) {
		t.Fatalf("reducer effects changed across identical runs:\nfirst=%+v\nsecond=%+v", firstEffects, secondEffects)
	}

	// Input must remain unchanged.
	if got := input.Orders["order-1"].Stages[0].Status; got != state.StagePending {
		t.Fatalf("input state mutated in-place, stage status=%q", got)
	}
}

func TestReducerUnknownEventNoop(t *testing.T) {
	current := fixtureStateForLifecycle()
	event := mustStateEvent(1, ingest.EventType("not_known"), map[string]any{"x": "y"}, fixtureTime())

	next, effects, err := Reduce(current, event)
	if err != nil {
		t.Fatalf("reduce unknown event: %v", err)
	}
	if !reflect.DeepEqual(current, next) {
		t.Fatalf("unknown event changed state:\ncurrent=%+v\nnext=%+v", current, next)
	}
	if len(effects) != 0 {
		t.Fatalf("unknown event should emit no effects, got %d", len(effects))
	}
}

func TestReducerHasHandlersForAllKnownEventTypes(t *testing.T) {
	for _, eventType := range ingest.AllEventTypes() {
		if _, ok := reduceHandlers[eventType]; !ok {
			t.Fatalf("missing reducer handler for event type %q", eventType)
		}
	}
}

func TestReducerPayloadDecodeFailureSurfaced(t *testing.T) {
	current := fixtureStateForLifecycle()
	event := ingest.StateEvent{
		ID:        2,
		Type:      string(ingest.EventDispatchRequested),
		Payload:   json.RawMessage(`{"order_id":`),
		Timestamp: fixtureTime(),
		Applied:   true,
	}

	next, effects, err := Reduce(current, event)
	if err == nil {
		t.Fatal("expected decode failure error")
	}
	if !reflect.DeepEqual(next, current) {
		t.Fatalf("decode failure changed state:\ncurrent=%+v\nnext=%+v", current, next)
	}
	if len(effects) != 0 {
		t.Fatalf("decode failure should emit no effects, got %d", len(effects))
	}
	if got := err.Error(); !containsAll(got, "dispatch_requested", "payload unreadable") {
		t.Fatalf("decode failure error missing context: %q", got)
	}
}

func TestReducerEmptyPayloadFailureSurfaced(t *testing.T) {
	current := fixtureStateForLifecycle()
	event := ingest.StateEvent{
		ID:        3,
		Type:      string(ingest.EventDispatchRequested),
		Timestamp: fixtureTime(),
		Applied:   true,
	}

	next, effects, err := Reduce(current, event)
	if err == nil {
		t.Fatal("expected empty payload error")
	}
	if !reflect.DeepEqual(next, current) {
		t.Fatalf("empty payload changed state:\ncurrent=%+v\nnext=%+v", current, next)
	}
	if len(effects) != 0 {
		t.Fatalf("empty payload should emit no effects, got %d", len(effects))
	}
	if got := err.Error(); !containsAll(got, "dispatch_requested", "payload unavailable") {
		t.Fatalf("empty payload error missing context: %q", got)
	}
}

func TestReducerTransitionsTable(t *testing.T) {
	ts := fixtureTime()

	tests := []struct {
		name     string
		current  state.State
		event    ingest.StateEvent
		assertFn func(t *testing.T, next state.State, effects []Effect)
	}{
		{
			name: "dispatch_requested sets stage dispatching and emits dispatch effect",
			current: state.State{
				Orders: map[string]state.OrderNode{
					"o1": {
						OrderID:   "o1",
						Status:    state.OrderActive,
						CreatedAt: ts,
						UpdatedAt: ts,
						Stages:    []state.StageNode{{StageIndex: 0, Status: state.StagePending}},
					},
				},
			},
			event: mustStateEvent(11, ingest.EventDispatchRequested, map[string]any{
				"order_id":    "o1",
				"stage_index": 0,
				"attempt_id":  "attempt-x",
			}, ts),
			assertFn: func(t *testing.T, next state.State, effects []Effect) {
				if got := next.Orders["o1"].Stages[0].Status; got != state.StageDispatching {
					t.Fatalf("stage status mismatch: got %q", got)
				}
				assertEffectTypes(t, effects, EffectDispatch)
				assertEffectIDs(t, effects, "event-11-effect-0")
			},
		},
		{
			name: "dispatch_completed sets stage running and creates attempt",
			current: state.State{
				Orders: map[string]state.OrderNode{
					"o1": {
						OrderID:   "o1",
						Status:    state.OrderActive,
						CreatedAt: ts,
						UpdatedAt: ts,
						Stages:    []state.StageNode{{StageIndex: 0, Status: state.StageDispatching}},
					},
				},
			},
			event: mustStateEvent(12, ingest.EventDispatchCompleted, map[string]any{
				"order_id":      "o1",
				"stage_index":   0,
				"attempt_id":    "attempt-a",
				"session_id":    "session-a",
				"worktree_name": "wt-a",
			}, ts),
			assertFn: func(t *testing.T, next state.State, effects []Effect) {
				if got := next.Orders["o1"].Stages[0].Status; got != state.StageRunning {
					t.Fatalf("stage status mismatch: got %q", got)
				}
				attempts := next.Orders["o1"].Stages[0].Attempts
				if len(attempts) != 1 {
					t.Fatalf("attempt count mismatch: got %d", len(attempts))
				}
				if attempts[0].Status != state.AttemptRunning {
					t.Fatalf("attempt status mismatch: got %q", attempts[0].Status)
				}
				if len(effects) != 0 {
					t.Fatalf("dispatch_completed should not emit effects, got %d", len(effects))
				}
			},
		},
		{
			name: "stage_completed sets merging and emits merge effect when mergeable",
			current: state.State{
				Orders: map[string]state.OrderNode{
					"o1": {
						OrderID:   "o1",
						Status:    state.OrderActive,
						CreatedAt: ts,
						UpdatedAt: ts,
						Stages: []state.StageNode{{
							StageIndex: 0,
							Status:     state.StageRunning,
							Attempts: []state.AttemptNode{{
								AttemptID:    "a1",
								Status:       state.AttemptRunning,
								StartedAt:    ts,
								WorktreeName: "wt-merge",
							}},
						}},
					},
				},
			},
			event: mustStateEvent(13, ingest.EventStageCompleted, map[string]any{
				"order_id":    "o1",
				"stage_index": 0,
			}, ts),
			assertFn: func(t *testing.T, next state.State, effects []Effect) {
				if got := next.Orders["o1"].Stages[0].Status; got != state.StageMerging {
					t.Fatalf("stage status mismatch: got %q, want %q", got, state.StageMerging)
				}
				assertEffectTypes(t, effects, EffectMerge)
				assertEffectIDs(t, effects, "event-13-effect-0")
			},
		},
		{
			name: "stage_completed non-mergeable completes order when last stage",
			current: state.State{
				Orders: map[string]state.OrderNode{
					"o1": {
						OrderID:   "o1",
						Status:    state.OrderActive,
						CreatedAt: ts,
						UpdatedAt: ts,
						Stages: []state.StageNode{{
							StageIndex: 0,
							Status:     state.StageRunning,
							Attempts: []state.AttemptNode{{
								AttemptID: "a1",
								Status:    state.AttemptRunning,
								StartedAt: ts,
							}},
						}},
					},
				},
			},
			event: mustStateEvent(40, ingest.EventStageCompleted, map[string]any{
				"order_id":    "o1",
				"stage_index": 0,
				"mergeable":   false,
			}, ts),
			assertFn: func(t *testing.T, next state.State, effects []Effect) {
				if got := next.Orders["o1"].Stages[0].Status; got != state.StageCompleted {
					t.Fatalf("stage status mismatch: got %q", got)
				}
				if got := next.Orders["o1"].Status; got != state.OrderCompleted {
					t.Fatalf("order status mismatch: got %q, want %q", got, state.OrderCompleted)
				}
				assertEffectTypes(t, effects, EffectWriteProjection, EffectAck)
			},
		},
		{
			name: "stage_completed non-mergeable keeps order active when more stages remain",
			current: state.State{
				Orders: map[string]state.OrderNode{
					"o1": {
						OrderID:   "o1",
						Status:    state.OrderActive,
						CreatedAt: ts,
						UpdatedAt: ts,
						Stages: []state.StageNode{
							{
								StageIndex: 0,
								Status:     state.StageRunning,
								Attempts: []state.AttemptNode{{
									AttemptID: "a1",
									Status:    state.AttemptRunning,
									StartedAt: ts,
								}},
							},
							{StageIndex: 1, Status: state.StagePending},
						},
					},
				},
			},
			event: mustStateEvent(41, ingest.EventStageCompleted, map[string]any{
				"order_id":    "o1",
				"stage_index": 0,
				"mergeable":   false,
			}, ts),
			assertFn: func(t *testing.T, next state.State, effects []Effect) {
				if got := next.Orders["o1"].Stages[0].Status; got != state.StageCompleted {
					t.Fatalf("stage status mismatch: got %q", got)
				}
				if got := next.Orders["o1"].Status; got != state.OrderActive {
					t.Fatalf("order status mismatch: got %q, want %q", got, state.OrderActive)
				}
				if len(effects) != 0 {
					t.Fatalf("non-mergeable stage with remaining stages should emit no effects, got %d", len(effects))
				}
			},
		},
		{
			name: "stage_failed sets failed and emits cleanup",
			current: state.State{
				Orders: map[string]state.OrderNode{
					"o1": {
						OrderID:   "o1",
						Status:    state.OrderActive,
						CreatedAt: ts,
						UpdatedAt: ts,
						Stages: []state.StageNode{{
							StageIndex: 0,
							Status:     state.StageRunning,
							Attempts: []state.AttemptNode{{
								AttemptID: "a1",
								Status:    state.AttemptRunning,
								StartedAt: ts,
							}},
						}},
					},
				},
			},
			event: mustStateEvent(14, ingest.EventStageFailed, map[string]any{
				"order_id":    "o1",
				"stage_index": 0,
				"error":       "runtime exited",
			}, ts),
			assertFn: func(t *testing.T, next state.State, effects []Effect) {
				if got := next.Orders["o1"].Stages[0].Status; got != state.StageFailed {
					t.Fatalf("stage status mismatch: got %q", got)
				}
				assertEffectTypes(t, effects, EffectCleanup)
			},
		},
		{
			name: "order_completed sets completed and emits projection+ack",
			current: state.State{
				Orders: map[string]state.OrderNode{
					"o1": {
						OrderID:   "o1",
						Status:    state.OrderActive,
						CreatedAt: ts,
						UpdatedAt: ts,
					},
				},
			},
			event: mustStateEvent(15, ingest.EventOrderCompleted, map[string]any{"order_id": "o1"}, ts),
			assertFn: func(t *testing.T, next state.State, effects []Effect) {
				if got := next.Orders["o1"].Status; got != state.OrderCompleted {
					t.Fatalf("order status mismatch: got %q", got)
				}
				assertEffectTypes(t, effects, EffectWriteProjection, EffectAck)
				assertEffectIDs(t, effects, "event-15-effect-0", "event-15-effect-1")
			},
		},
		{
			name: "order_failed sets failed and emits projection+ack",
			current: state.State{
				Orders: map[string]state.OrderNode{
					"o1": {
						OrderID:   "o1",
						Status:    state.OrderActive,
						CreatedAt: ts,
						UpdatedAt: ts,
					},
				},
			},
			event: mustStateEvent(16, ingest.EventOrderFailed, map[string]any{"order_id": "o1"}, ts),
			assertFn: func(t *testing.T, next state.State, effects []Effect) {
				if got := next.Orders["o1"].Status; got != state.OrderFailed {
					t.Fatalf("order status mismatch: got %q", got)
				}
				assertEffectTypes(t, effects, EffectWriteProjection, EffectAck)
			},
		},
		{
			name: "mode_changed updates run mode and increments epoch",
			current: state.State{
				Mode:      state.RunModeAuto,
				ModeEpoch: 0,
			},
			event: mustStateEvent(17, ingest.EventModeChanged, map[string]any{
				"mode":         "manual",
				"requested_by": "user:alice",
				"reason":       "incident",
			}, ts),
			assertFn: func(t *testing.T, next state.State, effects []Effect) {
				if next.Mode != state.RunModeManual {
					t.Fatalf("mode mismatch: got %q", next.Mode)
				}
				if next.ModeEpoch != 1 {
					t.Fatalf("mode epoch mismatch: got %d, want 1", next.ModeEpoch)
				}
				if len(next.ModeTransitions) != 1 {
					t.Fatalf("transition count mismatch: got %d, want 1", len(next.ModeTransitions))
				}
				tr := next.ModeTransitions[0]
				if tr.FromMode != state.RunModeAuto || tr.ToMode != state.RunModeManual {
					t.Fatalf("transition mismatch: %s -> %s", tr.FromMode, tr.ToMode)
				}
				if tr.Epoch != 1 {
					t.Fatalf("transition epoch mismatch: got %d, want 1", tr.Epoch)
				}
				if tr.RequestedBy != "user:alice" {
					t.Fatalf("transition requested_by mismatch: got %q", tr.RequestedBy)
				}
				if tr.Reason != "incident" {
					t.Fatalf("transition reason mismatch: got %q", tr.Reason)
				}
				if len(effects) != 0 {
					t.Fatalf("mode_changed should emit no effects, got %d", len(effects))
				}
			},
		},
		{
			name: "merge_completed advances to next stage",
			current: state.State{
				Orders: map[string]state.OrderNode{
					"o1": {
						OrderID:   "o1",
						Status:    state.OrderActive,
						CreatedAt: ts,
						UpdatedAt: ts,
						Stages: []state.StageNode{
							{StageIndex: 0, Status: state.StageMerging},
							{StageIndex: 1, Status: state.StagePending},
						},
					},
				},
			},
			event: mustStateEvent(18, ingest.EventMergeCompleted, map[string]any{"order_id": "o1", "stage_index": 0}, ts),
			assertFn: func(t *testing.T, next state.State, effects []Effect) {
				if got := next.Orders["o1"].Stages[0].Status; got != state.StageCompleted {
					t.Fatalf("stage status mismatch: got %q", got)
				}
				if got := next.Orders["o1"].Status; got != state.OrderActive {
					t.Fatalf("order status mismatch: got %q", got)
				}
				if len(effects) != 0 {
					t.Fatalf("merge_completed should emit no effects, got %d", len(effects))
				}
			},
		},
		{
			name: "merge_completed completes order at final stage",
			current: state.State{
				Orders: map[string]state.OrderNode{
					"o1": {
						OrderID:   "o1",
						Status:    state.OrderActive,
						CreatedAt: ts,
						UpdatedAt: ts,
						Stages:    []state.StageNode{{StageIndex: 0, Status: state.StageMerging}},
					},
				},
			},
			event: mustStateEvent(19, ingest.EventMergeCompleted, map[string]any{"order_id": "o1", "stage_index": 0}, ts),
			assertFn: func(t *testing.T, next state.State, effects []Effect) {
				if got := next.Orders["o1"].Status; got != state.OrderCompleted {
					t.Fatalf("order status mismatch: got %q", got)
				}
				if len(effects) != 0 {
					t.Fatalf("merge_completed final stage should emit no effects, got %d", len(effects))
				}
			},
		},
		{
			name: "merge_failed parks stage in review and emits ack",
			current: state.State{
				Orders: map[string]state.OrderNode{
					"o1": {
						OrderID:   "o1",
						Status:    state.OrderActive,
						CreatedAt: ts,
						UpdatedAt: ts,
						Stages:    []state.StageNode{{StageIndex: 0, Status: state.StageMerging}},
					},
				},
			},
			event: mustStateEvent(20, ingest.EventMergeFailed, map[string]any{"order_id": "o1", "stage_index": 0}, ts),
			assertFn: func(t *testing.T, next state.State, effects []Effect) {
				if got := next.Orders["o1"].Stages[0].Status; got != state.StageReview {
					t.Fatalf("stage status mismatch: got %q", got)
				}
				assertEffectTypes(t, effects, EffectAck)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next, effects, err := Reduce(tt.current, tt.event)
			if err != nil {
				t.Fatalf("reduce: %v", err)
			}
			tt.assertFn(t, next, effects)
			if next.LastEventID != "" {
				want := strconv.FormatUint(uint64(tt.event.ID), 10)
				if next.LastEventID != want {
					t.Fatalf("last event id mismatch: got %q want %q", next.LastEventID, want)
				}
			}
		})
	}
}

func TestReducerEdgeCases(t *testing.T) {
	ts := fixtureTime()

	tests := []struct {
		name    string
		current state.State
		event   ingest.StateEvent
	}{
		{
			name:    "missing order",
			current: state.State{Orders: map[string]state.OrderNode{}},
			event: mustStateEvent(30, ingest.EventDispatchRequested, map[string]any{
				"order_id": "missing", "stage_index": 0,
			}, ts),
		},
		{
			name: "terminal order",
			current: state.State{Orders: map[string]state.OrderNode{
				"done": {
					OrderID: "done",
					Status:  state.OrderCompleted,
					Stages:  []state.StageNode{{StageIndex: 0, Status: state.StageCompleted}},
				},
			}},
			event: mustStateEvent(31, ingest.EventDispatchRequested, map[string]any{
				"order_id": "done", "stage_index": 0,
			}, ts),
		},
		{
			name:    "empty state",
			current: state.State{},
			event: mustStateEvent(32, ingest.EventModeChanged, map[string]any{
				"mode": "unknown",
			}, ts),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next, effects, err := Reduce(tt.current, tt.event)
			if err != nil {
				t.Fatalf("reduce: %v", err)
			}
			if !reflect.DeepEqual(next, tt.current) {
				t.Fatalf("edge-case event should no-op:\ncurrent=%+v\nnext=%+v", tt.current, next)
			}
			if len(effects) != 0 {
				t.Fatalf("edge-case event should emit no effects, got %d", len(effects))
			}
		})
	}
}

func TestEffectIDDeterministic(t *testing.T) {
	current := state.State{
		Orders: map[string]state.OrderNode{
			"order-1": {
				OrderID: "order-1",
				Status:  state.OrderActive,
				Stages:  []state.StageNode{{StageIndex: 0, Status: state.StagePending}},
			},
		},
	}
	event := mustStateEvent(88, ingest.EventDispatchRequested, map[string]any{
		"order_id": "order-1", "stage_index": 0,
	}, fixtureTime())

	_, effectsA, err := Reduce(current, event)
	if err != nil {
		t.Fatalf("reduce A: %v", err)
	}
	_, effectsB, err := Reduce(current, event)
	if err != nil {
		t.Fatalf("reduce B: %v", err)
	}

	if len(effectsA) != len(effectsB) {
		t.Fatalf("effect count mismatch: %d vs %d", len(effectsA), len(effectsB))
	}
	for i := range effectsA {
		if effectsA[i].EffectID != effectsB[i].EffectID {
			t.Fatalf("effect id mismatch at %d: %q vs %q", i, effectsA[i].EffectID, effectsB[i].EffectID)
		}
	}
	if len(effectsA) > 0 && effectsA[0].EffectID != "event-88-effect-0" {
		t.Fatalf("unexpected effect id: %s", effectsA[0].EffectID)
	}
}

func assertEffectTypes(t *testing.T, effects []Effect, want ...EffectType) {
	t.Helper()
	if len(effects) != len(want) {
		t.Fatalf("effect count mismatch: got %d want %d", len(effects), len(want))
	}
	for i := range want {
		if effects[i].Type != want[i] {
			t.Fatalf("effect type mismatch at %d: got %q want %q", i, effects[i].Type, want[i])
		}
	}
}

func assertEffectIDs(t *testing.T, effects []Effect, want ...string) {
	t.Helper()
	if len(effects) != len(want) {
		t.Fatalf("effect count mismatch for ids: got %d want %d", len(effects), len(want))
	}
	for i := range want {
		if effects[i].EffectID != want[i] {
			t.Fatalf("effect id mismatch at %d: got %q want %q", i, effects[i].EffectID, want[i])
		}
	}
}

func mustStateEvent(id ingest.EventID, eventType ingest.EventType, payload map[string]any, ts time.Time) ingest.StateEvent {
	data, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}
	return ingest.StateEvent{
		ID:        id,
		Type:      string(eventType),
		Payload:   data,
		Timestamp: ts,
		Applied:   true,
	}
}

func containsAll(input string, want ...string) bool {
	for _, token := range want {
		if !strings.Contains(input, token) {
			return false
		}
	}
	return true
}

// --- Finding 6: control_received, schedule_promoted, session_adopted ---

func TestControlReceivedModeChange(t *testing.T) {
	ts := fixtureTime()
	current := state.State{
		Mode:      state.RunModeAuto,
		ModeEpoch: 0,
	}

	event := mustStateEvent(50, ingest.EventControlReceived, map[string]any{
		"command": "mode_change",
		"mode":    "supervised",
	}, ts)

	next, effects, err := Reduce(current, event)
	if err != nil {
		t.Fatalf("reduce: %v", err)
	}
	if next.Mode != state.RunModeSupervised {
		t.Fatalf("mode mismatch: got %q, want supervised", next.Mode)
	}
	if next.ModeEpoch != 1 {
		t.Fatalf("mode epoch mismatch: got %d, want 1", next.ModeEpoch)
	}
	if len(next.ModeTransitions) != 1 {
		t.Fatalf("transition count: got %d, want 1", len(next.ModeTransitions))
	}
	tr := next.ModeTransitions[0]
	if tr.RequestedBy != "control" || tr.Reason != "control_received" {
		t.Fatalf("transition metadata: requested_by=%q reason=%q", tr.RequestedBy, tr.Reason)
	}
	if len(effects) != 0 {
		t.Fatalf("control mode_change should emit no effects, got %d", len(effects))
	}
}

func TestControlReceivedUnknownCommand(t *testing.T) {
	ts := fixtureTime()
	current := state.State{Mode: state.RunModeAuto}

	event := mustStateEvent(51, ingest.EventControlReceived, map[string]any{
		"command": "shutdown",
	}, ts)

	next, effects, err := Reduce(current, event)
	if err != nil {
		t.Fatalf("reduce: %v", err)
	}
	if !reflect.DeepEqual(next, current) {
		t.Fatal("unknown control command changed state")
	}
	if len(effects) != 0 {
		t.Fatalf("unknown control command should emit no effects, got %d", len(effects))
	}
}

func TestControlReceivedInvalidMode(t *testing.T) {
	ts := fixtureTime()
	current := state.State{Mode: state.RunModeAuto}

	event := mustStateEvent(52, ingest.EventControlReceived, map[string]any{
		"command": "mode_change",
		"mode":    "turbo",
	}, ts)

	next, _, err := Reduce(current, event)
	if err != nil {
		t.Fatalf("reduce: %v", err)
	}
	if !reflect.DeepEqual(next, current) {
		t.Fatal("invalid mode in control_received changed state")
	}
}

func TestSchedulePromotedAddsNewOrder(t *testing.T) {
	ts := fixtureTime()
	current := state.State{
		Orders: map[string]state.OrderNode{},
	}

	event := mustStateEvent(60, ingest.EventSchedulePromoted, map[string]any{
		"order_id": "new-order",
		"stages": []map[string]any{
			{"skill": "lint", "runtime": "node"},
			{"skill": "test", "runtime": "go"},
		},
		"metadata": map[string]string{"branch": "feat/x"},
	}, ts)

	next, effects, err := Reduce(current, event)
	if err != nil {
		t.Fatalf("reduce: %v", err)
	}
	order, ok := next.Orders["new-order"]
	if !ok {
		t.Fatal("order not found in state")
	}
	if order.Status != state.OrderPending {
		t.Fatalf("order status: got %q, want pending", order.Status)
	}
	if len(order.Stages) != 2 {
		t.Fatalf("stage count: got %d, want 2", len(order.Stages))
	}
	if order.Stages[0].StageIndex != 0 || order.Stages[1].StageIndex != 1 {
		t.Fatalf("stage indexes: got %d, %d", order.Stages[0].StageIndex, order.Stages[1].StageIndex)
	}
	if order.Stages[0].Status != state.StagePending {
		t.Fatalf("stage 0 status: got %q, want pending", order.Stages[0].Status)
	}
	if order.Metadata["branch"] != "feat/x" {
		t.Fatalf("metadata mismatch: got %v", order.Metadata)
	}
	if len(effects) != 0 {
		t.Fatalf("schedule_promoted should emit no effects, got %d", len(effects))
	}
}

func TestSchedulePromotedRejectsDuplicate(t *testing.T) {
	ts := fixtureTime()
	current := state.State{
		Orders: map[string]state.OrderNode{
			"existing": {
				OrderID: "existing",
				Status:  state.OrderActive,
			},
		},
	}

	event := mustStateEvent(61, ingest.EventSchedulePromoted, map[string]any{
		"order_id": "existing",
		"stages":   []map[string]any{},
	}, ts)

	next, effects, err := Reduce(current, event)
	if err != nil {
		t.Fatalf("reduce: %v", err)
	}
	if !reflect.DeepEqual(next, current) {
		t.Fatal("duplicate schedule_promoted changed state")
	}
	if len(effects) != 0 {
		t.Fatalf("duplicate schedule_promoted should emit no effects, got %d", len(effects))
	}
}

func TestSchedulePromotedEmptyOrderID(t *testing.T) {
	ts := fixtureTime()
	current := state.State{}

	event := mustStateEvent(62, ingest.EventSchedulePromoted, map[string]any{
		"order_id": "",
	}, ts)

	next, _, err := Reduce(current, event)
	if err != nil {
		t.Fatalf("reduce: %v", err)
	}
	if !reflect.DeepEqual(next, current) {
		t.Fatal("empty order_id schedule_promoted changed state")
	}
}

func TestSchedulePromotedNilOrders(t *testing.T) {
	ts := fixtureTime()
	current := state.State{}

	event := mustStateEvent(63, ingest.EventSchedulePromoted, map[string]any{
		"order_id": "fresh",
		"stages":   []map[string]any{{"skill": "build"}},
	}, ts)

	next, _, err := Reduce(current, event)
	if err != nil {
		t.Fatalf("reduce: %v", err)
	}
	if _, ok := next.Orders["fresh"]; !ok {
		t.Fatal("order not created when state.Orders was nil")
	}
}

func TestSessionAdoptedUpdatesExistingAttempt(t *testing.T) {
	ts := fixtureTime()
	current := state.State{
		Orders: map[string]state.OrderNode{
			"o1": {
				OrderID: "o1",
				Status:  state.OrderActive,
				Stages: []state.StageNode{{
					StageIndex: 0,
					Status:     state.StageRunning,
					Attempts: []state.AttemptNode{{
						AttemptID: "att-1",
						SessionID: "old-session",
						Status:    state.AttemptLaunching,
					}},
				}},
			},
		},
	}

	event := mustStateEvent(70, ingest.EventSessionAdopted, map[string]any{
		"order_id":    "o1",
		"stage_index": 0,
		"attempt_id":  "att-1",
		"session_id":  "new-session",
	}, ts)

	next, effects, err := Reduce(current, event)
	if err != nil {
		t.Fatalf("reduce: %v", err)
	}
	attempt := next.Orders["o1"].Stages[0].Attempts[0]
	if attempt.SessionID != "new-session" {
		t.Fatalf("session id mismatch: got %q", attempt.SessionID)
	}
	if attempt.Status != state.AttemptRunning {
		t.Fatalf("attempt status mismatch: got %q, want running", attempt.Status)
	}
	if len(effects) != 0 {
		t.Fatalf("session_adopted should emit no effects, got %d", len(effects))
	}
}

func TestSessionAdoptedCreatesNewAttempt(t *testing.T) {
	ts := fixtureTime()
	current := state.State{
		Orders: map[string]state.OrderNode{
			"o1": {
				OrderID: "o1",
				Status:  state.OrderActive,
				Stages: []state.StageNode{{
					StageIndex: 0,
					Status:     state.StageDispatching,
				}},
			},
		},
	}

	event := mustStateEvent(71, ingest.EventSessionAdopted, map[string]any{
		"order_id":    "o1",
		"stage_index": 0,
		"attempt_id":  "att-new",
		"session_id":  "sess-new",
	}, ts)

	next, _, err := Reduce(current, event)
	if err != nil {
		t.Fatalf("reduce: %v", err)
	}
	stage := next.Orders["o1"].Stages[0]
	if len(stage.Attempts) != 1 {
		t.Fatalf("attempt count: got %d, want 1", len(stage.Attempts))
	}
	if stage.Attempts[0].AttemptID != "att-new" || stage.Attempts[0].SessionID != "sess-new" {
		t.Fatalf("attempt mismatch: %+v", stage.Attempts[0])
	}
	if stage.Status != state.StageRunning {
		t.Fatalf("stage status: got %q, want running", stage.Status)
	}
}

func TestSessionAdoptedPromotesPendingOrder(t *testing.T) {
	ts := fixtureTime()
	current := state.State{
		Orders: map[string]state.OrderNode{
			"o1": {
				OrderID: "o1",
				Status:  state.OrderPending,
				Stages: []state.StageNode{{
					StageIndex: 0,
					Status:     state.StagePending,
				}},
			},
		},
	}

	event := mustStateEvent(72, ingest.EventSessionAdopted, map[string]any{
		"order_id":    "o1",
		"stage_index": 0,
		"attempt_id":  "att-adopt",
		"session_id":  "sess-adopt",
	}, ts)

	next, _, err := Reduce(current, event)
	if err != nil {
		t.Fatalf("reduce: %v", err)
	}
	if next.Orders["o1"].Status != state.OrderActive {
		t.Fatalf("order status: got %q, want active", next.Orders["o1"].Status)
	}
	if next.Orders["o1"].Stages[0].Status != state.StageRunning {
		t.Fatalf("stage status: got %q, want running", next.Orders["o1"].Stages[0].Status)
	}
}

func TestSessionAdoptedMissingFields(t *testing.T) {
	ts := fixtureTime()
	current := state.State{
		Orders: map[string]state.OrderNode{
			"o1": {
				OrderID: "o1",
				Status:  state.OrderActive,
				Stages: []state.StageNode{{
					StageIndex: 0,
					Status:     state.StagePending,
				}},
			},
		},
	}

	tests := []struct {
		name    string
		payload map[string]any
	}{
		{"empty attempt_id", map[string]any{"order_id": "o1", "stage_index": 0, "attempt_id": "", "session_id": "s"}},
		{"empty session_id", map[string]any{"order_id": "o1", "stage_index": 0, "attempt_id": "a", "session_id": ""}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := mustStateEvent(73, ingest.EventSessionAdopted, tt.payload, ts)
			next, _, err := Reduce(current, event)
			if err != nil {
				t.Fatalf("reduce: %v", err)
			}
			if !reflect.DeepEqual(next, current) {
				t.Fatalf("session_adopted with %s changed state", tt.name)
			}
		})
	}
}

func TestSessionAdoptedTerminalOrder(t *testing.T) {
	ts := fixtureTime()
	current := state.State{
		Orders: map[string]state.OrderNode{
			"o1": {
				OrderID: "o1",
				Status:  state.OrderCompleted,
				Stages: []state.StageNode{{
					StageIndex: 0,
					Status:     state.StageCompleted,
				}},
			},
		},
	}

	event := mustStateEvent(74, ingest.EventSessionAdopted, map[string]any{
		"order_id": "o1", "stage_index": 0, "attempt_id": "a", "session_id": "s",
	}, ts)

	next, _, err := Reduce(current, event)
	if err != nil {
		t.Fatalf("reduce: %v", err)
	}
	if !reflect.DeepEqual(next, current) {
		t.Fatal("session_adopted on terminal order changed state")
	}
}

func fixtureStateForLifecycle() state.State {
	ts := fixtureTime()
	return state.State{
		Orders: map[string]state.OrderNode{
			"order-1": {
				OrderID:   "order-1",
				Status:    state.OrderActive,
				CreatedAt: ts,
				UpdatedAt: ts,
				Stages: []state.StageNode{
					{StageIndex: 0, Status: state.StagePending},
				},
			},
		},
		Mode:          state.RunModeAuto,
		SchemaVersion: statever.Current,
	}
}

func fixtureTime() time.Time {
	return time.Date(2026, 2, 28, 18, 30, 0, 0, time.UTC)
}
