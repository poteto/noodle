package loop

import (
	"testing"
	"time"

	"github.com/poteto/noodle/adapter"
	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/event"
	"github.com/poteto/noodle/mise"
)

func digestOrFail(t *testing.T, brief mise.Brief, orders OrdersFile) string {
	t.Helper()
	digest, err := scheduleDecisionDigest(brief, orders)
	if err != nil {
		t.Fatalf("scheduleDecisionDigest: %v", err)
	}
	return digest
}

func backlogBrief(items ...adapter.BacklogItem) mise.Brief {
	return mise.Brief{
		Backlog:   items,
		TaskTypes: []mise.TaskTypeSummary{{Key: "execute", Schedule: "when ready"}},
		Routing:   mise.RoutingSnapshot{Defaults: mise.RoutingPolicy{Provider: "claude", Model: "claude-opus-4-6"}},
	}
}

func TestScheduleDecisionDigestIgnoresScheduleLifecycle(t *testing.T) {
	brief := backlogBrief(adapter.BacklogItem{ID: "1", Title: "fix login", Status: "open"})
	orders := OrdersFile{
		Orders: []Order{
			{ID: "42", Status: OrderStatusActive, Stages: []Stage{{TaskKey: "execute", Status: StageStatusPending}}},
		},
	}
	base := digestOrFail(t, brief, orders)

	// Everything the schedule cycle mutates by running: a fresh brief
	// timestamp, the scheduler's own order, the resulting active summary,
	// history and lifecycle events.
	churned := brief
	churned.GeneratedAt = time.Now()
	churned.ActiveSummary = mise.ActiveSummary{Total: 1, ByTaskKey: map[string]int{"schedule": 1}}
	churned.Resources = mise.ResourceSnapshot{MaxConcurrency: 4, Active: 1, Available: 3}
	churned.RecentHistory = []mise.HistoryItem{{SessionID: "schedule-1", TaskKey: "schedule", Status: "completed"}}
	churned.RecentEvents = []mise.RecentEvent{
		{Type: "schedule.completed", Seq: 7, Summary: "order schedule completed"},
		{Type: "stage.completed", Seq: 8, Summary: "schedule stage completed for schedule"},
	}
	churnedOrders := OrdersFile{
		GeneratedAt: time.Now(),
		Orders: append([]Order{
			scheduleOrder(config.DefaultConfig(), "chef says hi"),
			{ID: scheduleBootstrapOrderID, Status: OrderStatusActive, Stages: []Stage{{TaskKey: "oops", Status: StageStatusActive}}},
		}, orders.Orders...),
	}
	// The cook itself moved pending → active → merging.
	churnedOrders.Orders[2].Stages[0].Status = StageStatusMerging

	if got := digestOrFail(t, churned, churnedOrders); got != base {
		t.Fatalf("schedule lifecycle churn changed digest:\nbase = %s\ngot  = %s", base, got)
	}
}

func TestScheduleDecisionDigestIgnoresStageProgress(t *testing.T) {
	brief := backlogBrief(adapter.BacklogItem{ID: "1", Title: "fix login", Status: "open"})
	pending := OrdersFile{
		Orders: []Order{{ID: "42", Status: OrderStatusActive, Stages: []Stage{
			{TaskKey: "execute", Status: StageStatusPending},
			{TaskKey: "review", Status: StageStatusPending},
		}}},
	}
	advanced := OrdersFile{
		Orders: []Order{{ID: "42", Status: OrderStatusActive, Stages: []Stage{
			{TaskKey: "execute", Status: StageStatusCompleted},
			{TaskKey: "review", Status: StageStatusActive},
		}}},
	}

	if digestOrFail(t, brief, pending) != digestOrFail(t, brief, advanced) {
		t.Fatal("the loop executing an already-scheduled pipeline changed the digest")
	}
}

func TestScheduleDecisionDigestChangesOnDecisionRelevantState(t *testing.T) {
	item := adapter.BacklogItem{ID: "1", Title: "fix login", Status: "open"}
	brief := backlogBrief(item)
	orders := OrdersFile{
		Orders: []Order{
			{ID: "42", Status: OrderStatusActive, Stages: []Stage{{TaskKey: "execute", Status: StageStatusPending}}},
		},
	}
	base := digestOrFail(t, brief, orders)

	cases := []struct {
		name   string
		brief  mise.Brief
		orders OrdersFile
	}{
		{
			name:   "backlog item added",
			brief:  backlogBrief(item, adapter.BacklogItem{ID: "2", Title: "add metrics", Status: "open"}),
			orders: orders,
		},
		{
			name:   "backlog item state changed",
			brief:  backlogBrief(adapter.BacklogItem{ID: "1", Title: "fix login", Status: "blocked"}),
			orders: orders,
		},
		{
			name: "backlog dependencies changed",
			brief: backlogBrief(adapter.BacklogItem{
				ID: "1", Title: "fix login", Status: "open",
				Extra: map[string]any{"depends_on": []any{"7"}},
			}),
			orders: orders,
		},
		{
			name: "ticket blocks a target",
			brief: func() mise.Brief {
				b := backlogBrief(item)
				b.Tickets = []event.Ticket{{Target: "42", Status: event.TicketStatusBlocked, Reason: "needs decision"}}
				return b
			}(),
			orders: orders,
		},
		{
			name:  "active non-schedule order added",
			brief: brief,
			orders: OrdersFile{Orders: append(append([]Order{}, orders.Orders...), Order{
				ID: "43", Status: OrderStatusActive, Stages: []Stage{{TaskKey: "execute", Status: StageStatusPending}},
			})},
		},
		{
			name:  "order landed",
			brief: brief,
			orders: OrdersFile{Orders: []Order{
				{ID: "42", Status: OrderStatusCompleted, Stages: []Stage{{TaskKey: "execute", Status: StageStatusCompleted}}},
			}},
		},
		{
			name:  "stage failed",
			brief: brief,
			orders: OrdersFile{Orders: []Order{
				{ID: "42", Status: OrderStatusActive, Stages: []Stage{{TaskKey: "execute", Status: StageStatusFailed}}},
			}},
		},
		{
			name: "task type added",
			brief: func() mise.Brief {
				b := backlogBrief(item)
				b.TaskTypes = append(b.TaskTypes, mise.TaskTypeSummary{Key: "review", Schedule: "after execute"})
				return b
			}(),
			orders: orders,
		},
		{
			name: "routing default changed",
			brief: func() mise.Brief {
				b := backlogBrief(item)
				b.Routing.Defaults.Model = "gpt-5.4"
				return b
			}(),
			orders: orders,
		},
		{
			name: "sync warning raised",
			brief: func() mise.Brief {
				b := backlogBrief(item)
				b.Warnings = []string{"backlog sync script missing; returning empty backlog"}
				return b
			}(),
			orders: orders,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := digestOrFail(t, tc.brief, tc.orders); got == base {
				t.Fatalf("digest unchanged for %s", tc.name)
			}
		})
	}
}

func TestScheduleDecisionDigestIsOrderIndependent(t *testing.T) {
	first := adapter.BacklogItem{ID: "1", Title: "fix login", Status: "open"}
	second := adapter.BacklogItem{ID: "2", Title: "add metrics", Status: "open"}
	orders := OrdersFile{
		Orders: []Order{
			{ID: "42", Status: OrderStatusActive, Stages: []Stage{{TaskKey: "execute", Status: StageStatusPending}}},
			{ID: "43", Status: OrderStatusActive, Stages: []Stage{{TaskKey: "execute", Status: StageStatusPending}}},
		},
	}
	reversed := OrdersFile{Orders: []Order{orders.Orders[1], orders.Orders[0]}}

	if digestOrFail(t, backlogBrief(first, second), orders) != digestOrFail(t, backlogBrief(second, first), reversed) {
		t.Fatal("digest depends on adapter iteration order")
	}
}
