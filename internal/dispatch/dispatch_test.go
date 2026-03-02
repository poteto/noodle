package dispatch

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/poteto/noodle/internal/ingest"
	"github.com/poteto/noodle/internal/state"
)

func TestPlanDispatches(t *testing.T) {
	runningAttempt := state.AttemptNode{AttemptID: "att-running", Status: state.AttemptRunning}

	tests := []struct {
		name          string
		input         state.State
		maxConcurrent int
		failedOrders  map[string]bool
		wantCand      []DispatchCandidate
		wantBlocked   []BlockedCandidate
		wantRemaining int
	}{
		{
			name: "picks first pending stage per order",
			input: state.State{
				Orders: map[string]state.OrderNode{
					"order-a": {
						OrderID: "order-a",
						Status:  state.OrderActive,
						Stages: []state.StageNode{
							{StageIndex: 0, Status: state.StageCompleted, Group: "g1"},
							{StageIndex: 1, Status: state.StagePending, Runtime: "", Group: "g2"},
							{StageIndex: 2, Status: state.StagePending, Runtime: "cursor", Group: "g2"},
						},
					},
					"order-b": {
						OrderID: "order-b",
						Status:  state.OrderActive,
						Stages: []state.StageNode{
							{StageIndex: 0, Status: state.StagePending, Runtime: "sprites"},
						},
					},
					"order-c": {
						OrderID: "order-c",
						Status:  state.OrderCompleted,
						Stages: []state.StageNode{
							{StageIndex: 0, Status: state.StagePending},
						},
					},
				},
			},
			maxConcurrent: 5,
			wantCand: []DispatchCandidate{
				{OrderID: "order-a", StageIndex: 1, Runtime: "process"},
				{OrderID: "order-b", StageIndex: 0, Runtime: "sprites"},
			},
			wantBlocked:   []BlockedCandidate{},
			wantRemaining: 3,
		},
		{
			name: "skips busy orders with active stages",
			input: state.State{
				Orders: map[string]state.OrderNode{
					"order-a": {
						OrderID: "order-a",
						Status:  state.OrderActive,
						Stages: []state.StageNode{
							{StageIndex: 0, Status: state.StageRunning, Attempts: []state.AttemptNode{runningAttempt}},
							{StageIndex: 1, Status: state.StagePending},
						},
					},
					"order-b": {
						OrderID: "order-b",
						Status:  state.OrderActive,
						Stages: []state.StageNode{
							{StageIndex: 0, Status: state.StagePending},
						},
					},
				},
			},
			maxConcurrent: 3,
			wantCand: []DispatchCandidate{
				{OrderID: "order-b", StageIndex: 0, Runtime: "process"},
			},
			wantBlocked: []BlockedCandidate{
				{OrderID: "order-a", StageIndex: 1, Reason: "busy"},
			},
			wantRemaining: 1,
		},
		{
			name: "skips failed orders",
			input: state.State{
				Orders: map[string]state.OrderNode{
					"order-a": {
						OrderID: "order-a",
						Status:  state.OrderActive,
						Stages: []state.StageNode{
							{StageIndex: 0, Status: state.StagePending},
						},
					},
					"order-b": {
						OrderID: "order-b",
						Status:  state.OrderActive,
						Stages: []state.StageNode{
							{StageIndex: 0, Status: state.StagePending},
						},
					},
				},
			},
			maxConcurrent: 2,
			failedOrders:  map[string]bool{"order-a": true},
			wantCand: []DispatchCandidate{
				{OrderID: "order-b", StageIndex: 0, Runtime: "process"},
			},
			wantBlocked: []BlockedCandidate{
				{OrderID: "order-a", StageIndex: 0, Reason: "failed"},
			},
			wantRemaining: 1,
		},
		{
			name: "respects capacity limit",
			input: state.State{
				Orders: map[string]state.OrderNode{
					"order-a": {
						OrderID: "order-a",
						Status:  state.OrderActive,
						Stages: []state.StageNode{
							{StageIndex: 0, Status: state.StagePending},
						},
					},
					"order-b": {
						OrderID: "order-b",
						Status:  state.OrderActive,
						Stages: []state.StageNode{
							{StageIndex: 0, Status: state.StagePending},
						},
					},
					"order-c": {
						OrderID: "order-c",
						Status:  state.OrderActive,
						Stages: []state.StageNode{
							{StageIndex: 0, Status: state.StagePending},
						},
					},
					"order-z": {
						OrderID: "order-z",
						Status:  state.OrderActive,
						Stages: []state.StageNode{
							{StageIndex: 0, Status: state.StageRunning, Attempts: []state.AttemptNode{runningAttempt}},
							{StageIndex: 1, Status: state.StagePending},
						},
					},
				},
			},
			maxConcurrent: 2,
			wantCand: []DispatchCandidate{
				{OrderID: "order-a", StageIndex: 0, Runtime: "process"},
			},
			wantBlocked: []BlockedCandidate{
				{OrderID: "order-b", StageIndex: 0, Reason: "capacity"},
				{OrderID: "order-c", StageIndex: 0, Reason: "capacity"},
				{OrderID: "order-z", StageIndex: 1, Reason: "busy"},
			},
			wantRemaining: 0,
		},
		{
			name: "returns blocked candidates with reasons",
			input: state.State{
				Orders: map[string]state.OrderNode{
					"order-a": {
						OrderID: "order-a",
						Status:  state.OrderActive,
						Stages: []state.StageNode{
							{StageIndex: 0, Status: state.StageRunning, Attempts: []state.AttemptNode{runningAttempt}},
							{StageIndex: 1, Status: state.StagePending},
						},
					},
					"order-b": {
						OrderID: "order-b",
						Status:  state.OrderActive,
						Stages: []state.StageNode{
							{StageIndex: 0, Status: state.StagePending},
						},
					},
					"order-c": {
						OrderID: "order-c",
						Status:  state.OrderActive,
						Stages: []state.StageNode{
							{StageIndex: 0, Status: state.StageCompleted},
						},
					},
					"order-d": {
						OrderID: "order-d",
						Status:  state.OrderActive,
						Stages: []state.StageNode{
							{StageIndex: 0, Status: state.StagePending},
						},
					},
					"order-e": {
						OrderID: "order-e",
						Status:  state.OrderActive,
						Stages: []state.StageNode{
							{StageIndex: 0, Status: state.StagePending},
						},
					},
				},
			},
			maxConcurrent: 2,
			failedOrders:  map[string]bool{"order-b": true},
			wantCand: []DispatchCandidate{
				{OrderID: "order-d", StageIndex: 0, Runtime: "process"},
			},
			wantBlocked: []BlockedCandidate{
				{OrderID: "order-a", StageIndex: 1, Reason: "busy"},
				{OrderID: "order-b", StageIndex: 0, Reason: "failed"},
				{OrderID: "order-c", StageIndex: -1, Reason: "no pending stage"},
				{OrderID: "order-e", StageIndex: 0, Reason: "capacity"},
			},
			wantRemaining: 0,
		},
		{
			name:          "empty state returns empty plan",
			input:         state.State{},
			maxConcurrent: 3,
			wantCand:      []DispatchCandidate{},
			wantBlocked:   []BlockedCandidate{},
			wantRemaining: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PlanDispatches(tt.input, tt.maxConcurrent, tt.failedOrders)
			assertCandidates(t, got.Candidates, tt.wantCand)
			if !reflect.DeepEqual(got.Blocked, tt.wantBlocked) {
				t.Fatalf("blocked mismatch:\n got=%+v\nwant=%+v", got.Blocked, tt.wantBlocked)
			}
			if got.CapacityRemaining != tt.wantRemaining {
				t.Fatalf("capacity remaining mismatch: got %d want %d", got.CapacityRemaining, tt.wantRemaining)
			}
		})
	}
}

func TestRouteCompletion(t *testing.T) {
	now := time.Date(2026, 2, 28, 20, 0, 0, 0, time.UTC)
	exitZero := 0
	exitOne := 1

	tests := []struct {
		name         string
		input        state.State
		rec          CompletionRecord
		wantStage    state.StageLifecycleStatus
		wantAttempt  state.AttemptStatus
		wantOrder    state.OrderLifecycleStatus
		wantEvent    ingest.EventType
		wantEventNum int
	}{
		{
			name: "success marks attempt and stage completed",
			input: state.State{
				Orders: map[string]state.OrderNode{
					"order-1": {
						OrderID: "order-1",
						Status:  state.OrderActive,
						Stages: []state.StageNode{
							{
								StageIndex: 0,
								Status:     state.StageRunning,
								Attempts: []state.AttemptNode{
									{AttemptID: "att-1", Status: state.AttemptRunning},
								},
							},
						},
					},
				},
			},
			rec: CompletionRecord{
				OrderID:     "order-1",
				StageIndex:  0,
				AttemptID:   "att-1",
				Status:      state.AttemptCompleted,
				ExitCode:    &exitZero,
				CompletedAt: now,
			},
			wantStage:    state.StageCompleted,
			wantAttempt:  state.AttemptCompleted,
			wantOrder:    state.OrderActive,
			wantEvent:    ingest.EventStageCompleted,
			wantEventNum: 1,
		},
		{
			name: "failure generates stage_failed",
			input: state.State{
				Orders: map[string]state.OrderNode{
					"order-1": {
						OrderID: "order-1",
						Status:  state.OrderActive,
						Stages: []state.StageNode{
							{
								StageIndex: 0,
								Status:     state.StageRunning,
								Attempts: []state.AttemptNode{
									{AttemptID: "att-1", Status: state.AttemptRunning},
								},
							},
						},
					},
				},
			},
			rec: CompletionRecord{
				OrderID:     "order-1",
				StageIndex:  0,
				AttemptID:   "att-1",
				Status:      state.AttemptFailed,
				ExitCode:    &exitOne,
				Error:       "runtime exited",
				CompletedAt: now,
			},
			wantStage:    state.StageFailed,
			wantAttempt:  state.AttemptFailed,
			wantOrder:    state.OrderFailed,
			wantEvent:    ingest.EventStageFailed,
			wantEventNum: 1,
		},
		{
			name: "failure without retry generates stage_failed",
			input: state.State{
				Orders: map[string]state.OrderNode{
					"order-1": {
						OrderID: "order-1",
						Status:  state.OrderActive,
						Stages: []state.StageNode{
							{
								StageIndex: 0,
								Status:     state.StageRunning,
								Attempts: []state.AttemptNode{
									{AttemptID: "att-1", Status: state.AttemptFailed},
									{AttemptID: "att-2", Status: state.AttemptRunning},
								},
							},
						},
					},
				},
			},
			rec: CompletionRecord{
				OrderID:     "order-1",
				StageIndex:  0,
				AttemptID:   "att-2",
				Status:      state.AttemptFailed,
				ExitCode:    &exitOne,
				Error:       "retry exhausted",
				CompletedAt: now,
			},
			wantStage:    state.StageFailed,
			wantAttempt:  state.AttemptFailed,
			wantOrder:    state.OrderFailed,
			wantEvent:    ingest.EventStageFailed,
			wantEventNum: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotState, gotEvents, err := RouteCompletion(tt.input, tt.rec)
			if err != nil {
				t.Fatalf("route completion: %v", err)
			}
			stage := gotState.Orders["order-1"].Stages[0]
			if stage.Status != tt.wantStage {
				t.Fatalf("stage status mismatch: got %q want %q", stage.Status, tt.wantStage)
			}
			lastAttempt := stage.Attempts[len(stage.Attempts)-1]
			if lastAttempt.Status != tt.wantAttempt {
				t.Fatalf("attempt status mismatch: got %q want %q", lastAttempt.Status, tt.wantAttempt)
			}
			if gotState.Orders["order-1"].Status != tt.wantOrder {
				t.Fatalf("order status mismatch: got %q want %q", gotState.Orders["order-1"].Status, tt.wantOrder)
			}
			if len(gotEvents) != tt.wantEventNum {
				t.Fatalf("event count mismatch: got %d want %d", len(gotEvents), tt.wantEventNum)
			}
			if len(gotEvents) > 0 && ingest.EventType(gotEvents[0].Type) != tt.wantEvent {
				t.Fatalf("event type mismatch: got %q want %q", gotEvents[0].Type, tt.wantEvent)
			}
		})
	}
}

func TestAdvanceOrder(t *testing.T) {
	tests := []struct {
		name        string
		input       state.State
		orderID     string
		wantRemoved bool
		wantStatus  state.OrderLifecycleStatus
		wantStage1  state.StageLifecycleStatus
	}{
		{
			name: "advances to next pending stage",
			input: state.State{
				Orders: map[string]state.OrderNode{
					"order-1": {
						OrderID: "order-1",
						Status:  state.OrderActive,
						Stages: []state.StageNode{
							{StageIndex: 0, Status: state.StageCompleted},
							{StageIndex: 1, Status: ""},
						},
					},
				},
			},
			orderID:     "order-1",
			wantRemoved: false,
			wantStatus:  state.OrderActive,
			wantStage1:  state.StagePending,
		},
		{
			name: "completes order when all stages done",
			input: state.State{
				Orders: map[string]state.OrderNode{
					"order-1": {
						OrderID: "order-1",
						Status:  state.OrderActive,
						Stages: []state.StageNode{
							{StageIndex: 0, Status: state.StageCompleted},
							{StageIndex: 1, Status: state.StageCompleted},
						},
					},
				},
			},
			orderID:     "order-1",
			wantRemoved: true,
			wantStatus:  state.OrderCompleted,
		},
		{
			name: "handles single-stage orders",
			input: state.State{
				Orders: map[string]state.OrderNode{
					"order-1": {
						OrderID: "order-1",
						Status:  state.OrderActive,
						Stages: []state.StageNode{
							{StageIndex: 0, Status: state.StageCompleted},
						},
					},
				},
			},
			orderID:     "order-1",
			wantRemoved: true,
			wantStatus:  state.OrderCompleted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotState, gotRemoved := AdvanceOrder(tt.input, tt.orderID)
			if gotRemoved != tt.wantRemoved {
				t.Fatalf("removed mismatch: got %v want %v", gotRemoved, tt.wantRemoved)
			}
			order := gotState.Orders[tt.orderID]
			if order.Status != tt.wantStatus {
				t.Fatalf("order status mismatch: got %q want %q", order.Status, tt.wantStatus)
			}
			if len(order.Stages) > 1 && tt.wantStage1 != "" && order.Stages[1].Status != tt.wantStage1 {
				t.Fatalf("stage[1] status mismatch: got %q want %q", order.Stages[1].Status, tt.wantStage1)
			}
		})
	}
}

func TestRouteFailure(t *testing.T) {
	input := state.State{
		Orders: map[string]state.OrderNode{
			"order-1": {
				OrderID: "order-1",
				Status:  state.OrderActive,
				Stages: []state.StageNode{
					{
						StageIndex: 0,
						Status:     state.StageRunning,
						Attempts: []state.AttemptNode{
							{AttemptID: "att-1", Status: state.AttemptFailed},
							{AttemptID: "att-2", Status: state.AttemptRunning},
						},
					},
				},
			},
		},
	}

	got := RouteFailure(input, "order-1", 0, "failed at runtime")
	stage := got.Orders["order-1"].Stages[0]
	if stage.Status != state.StageFailed {
		t.Fatalf("stage status mismatch: got %q want %q", stage.Status, state.StageFailed)
	}
	if got.Orders["order-1"].Status != state.OrderFailed {
		t.Fatalf("order status mismatch: got %q want %q", got.Orders["order-1"].Status, state.OrderFailed)
	}
}

func TestEdgeCases(t *testing.T) {
	now := time.Date(2026, 2, 28, 21, 0, 0, 0, time.UTC)

	t.Run("missing orders", func(t *testing.T) {
		s := state.State{Orders: map[string]state.OrderNode{}}

		next, removed := AdvanceOrder(s, "missing")
		if removed {
			t.Fatal("missing order should not be removed")
		}
		if !reflect.DeepEqual(next, s) {
			t.Fatalf("advance missing order should no-op:\n got=%+v\nwant=%+v", next, s)
		}

		failed := RouteFailure(s, "missing", 0, "x")
		if !reflect.DeepEqual(failed, s) {
			t.Fatalf("route failure missing order should no-op:\n got=%+v\nwant=%+v", failed, s)
		}
	})

	t.Run("terminal states no-op in route completion", func(t *testing.T) {
		s := state.State{
			Orders: map[string]state.OrderNode{
				"order-1": {
					OrderID: "order-1",
					Status:  state.OrderCompleted,
					Stages: []state.StageNode{
						{StageIndex: 0, Status: state.StageCompleted},
					},
				},
			},
		}

		next, events, err := RouteCompletion(s, CompletionRecord{
			OrderID:     "order-1",
			StageIndex:  0,
			AttemptID:   "att-1",
			Status:      state.AttemptCompleted,
			CompletedAt: now,
		})
		if err != nil {
			t.Fatalf("route completion: %v", err)
		}
		if !reflect.DeepEqual(next, s) {
			t.Fatalf("terminal route completion should no-op:\n got=%+v\nwant=%+v", next, s)
		}
		if len(events) != 0 {
			t.Fatalf("terminal route completion should emit no events, got %d", len(events))
		}
	})

	t.Run("empty stages and grouped stages", func(t *testing.T) {
		s := state.State{
			Orders: map[string]state.OrderNode{
				"empty": {
					OrderID: "empty",
					Status:  state.OrderActive,
					Stages:  []state.StageNode{},
				},
				"grouped": {
					OrderID: "grouped",
					Status:  state.OrderActive,
					Stages: []state.StageNode{
						{StageIndex: 0, Status: state.StageCompleted, Group: "1"},
						{StageIndex: 1, Status: state.StagePending, Group: "2"},
						{StageIndex: 2, Status: state.StagePending, Group: "2"},
					},
				},
			},
		}

		plan := PlanDispatches(s, 2, nil)
		assertCandidates(t, plan.Candidates, []DispatchCandidate{
			{OrderID: "grouped", StageIndex: 1, Runtime: "process"},
		})
		if len(plan.Blocked) != 1 || plan.Blocked[0].OrderID != "empty" || plan.Blocked[0].Reason != "no pending stage" {
			t.Fatalf("unexpected blocked entries: %+v", plan.Blocked)
		}
	})
}

func assertCandidates(t *testing.T, got, want []DispatchCandidate) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("candidate count mismatch: got %d want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].OrderID != want[i].OrderID {
			t.Fatalf("candidate[%d] order mismatch: got %q want %q", i, got[i].OrderID, want[i].OrderID)
		}
		if got[i].StageIndex != want[i].StageIndex {
			t.Fatalf("candidate[%d] stage index mismatch: got %d want %d", i, got[i].StageIndex, want[i].StageIndex)
		}
		if got[i].Runtime != want[i].Runtime {
			t.Fatalf("candidate[%d] runtime mismatch: got %q want %q", i, got[i].Runtime, want[i].Runtime)
		}
	}
}

func TestRouteCompletionEventPayloadUsesSnakeCase(t *testing.T) {
	now := time.Date(2026, 2, 28, 22, 0, 0, 0, time.UTC)
	exitZero := 0
	s := state.State{
		Orders: map[string]state.OrderNode{
			"order-1": {
				OrderID: "order-1",
				Status:  state.OrderActive,
				Stages: []state.StageNode{
					{
						StageIndex: 0,
						Status:     state.StageRunning,
						Attempts: []state.AttemptNode{
							{AttemptID: "att-1", Status: state.AttemptRunning},
						},
					},
				},
			},
		},
	}

	_, events, err := RouteCompletion(s, CompletionRecord{
		OrderID:     "order-1",
		StageIndex:  0,
		AttemptID:   "att-1",
		Status:      state.AttemptCompleted,
		ExitCode:    &exitZero,
		CompletedAt: now,
	})
	if err != nil {
		t.Fatalf("route completion: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	var payload map[string]json.RawMessage
	if err := json.Unmarshal(events[0].Payload, &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	for _, key := range []string{"order_id", "stage_index", "attempt_id", "error", "exit_code"} {
		if _, ok := payload[key]; !ok {
			t.Fatalf("expected snake_case key %q in payload", key)
		}
	}
}
