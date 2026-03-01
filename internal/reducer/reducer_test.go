package reducer

import (
	"encoding/json"
	"reflect"
	"strconv"
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

	firstState, firstEffects := r(input, event)
	secondState, secondEffects := r(input, event)

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

	next, effects := Reduce(current, event)
	if !reflect.DeepEqual(current, next) {
		t.Fatalf("unknown event changed state:\ncurrent=%+v\nnext=%+v", current, next)
	}
	if len(effects) != 0 {
		t.Fatalf("unknown event should emit no effects, got %d", len(effects))
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
			name: "mode_changed updates run mode",
			current: state.State{
				Mode: state.RunModeAuto,
			},
			event: mustStateEvent(17, ingest.EventModeChanged, map[string]any{"mode": "manual"}, ts),
			assertFn: func(t *testing.T, next state.State, effects []Effect) {
				if next.Mode != state.RunModeManual {
					t.Fatalf("mode mismatch: got %q", next.Mode)
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
			next, effects := Reduce(tt.current, tt.event)
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
			next, effects := Reduce(tt.current, tt.event)
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

	_, effectsA := Reduce(current, event)
	_, effectsB := Reduce(current, event)

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
