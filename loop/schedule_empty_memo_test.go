package loop

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/poteto/noodle/adapter"
	"github.com/poteto/noodle/event"
	"github.com/poteto/noodle/mise"
)

func TestScheduleDecisionDigestIgnoresScheduleLifecycleAndOrdering(t *testing.T) {
	logger, _ := newTestLogger()
	tc := newTestLoop(t, logger)
	brief := mise.Brief{
		GeneratedAt: time.Unix(10, 0),
		Backlog: []adapter.BacklogItem{
			{ID: "2", Title: "second", Status: adapter.BacklogStatusOpen},
			{ID: "1", Title: "first", Status: adapter.BacklogStatusOpen},
		},
		ActiveSummary: mise.ActiveSummary{Total: 1, ByTaskKey: map[string]int{"schedule": 1}},
		Resources:     mise.ResourceSnapshot{MaxConcurrency: 3, Active: 1, Available: 2},
		Routing: mise.RoutingSnapshot{
			Defaults:          mise.RoutingPolicy{Provider: "codex", Model: "gpt-5"},
			AvailableRuntimes: []string{"sprites", "process"},
		},
		TaskTypes: []mise.TaskTypeSummary{
			{Key: "review", Schedule: "after execute"},
			{Key: "execute", Schedule: "ready work"},
		},
		RecentHistory: []mise.HistoryItem{{SessionID: "schedule-1", TaskKey: scheduleOrderID, Status: "completed"}},
	}
	orders := OrdersFile{Orders: []Order{
		testOrder("work-2", "execute", "execute", "codex", "gpt-5"),
		testOrder(scheduleOrderID, scheduleOrderID, scheduleOrderID, "codex", "gpt-5"),
		testOrder("work-1", "review", "review", "codex", "gpt-5"),
	}}

	before, err := tc.loop.scheduleDecisionDigest(brief, orders)
	if err != nil {
		t.Fatalf("initial digest: %v", err)
	}
	if err := tc.loop.events.Emit(event.LoopEventScheduleCompleted, ScheduleCompletedPayload{SessionID: "schedule-1"}); err != nil {
		t.Fatalf("emit schedule completion: %v", err)
	}

	brief.GeneratedAt = time.Unix(20, 0)
	brief.ActiveSummary = mise.ActiveSummary{Total: 0}
	brief.Resources.Active = 0
	brief.Resources.Available = 3
	brief.RecentHistory = nil
	brief.Backlog[0], brief.Backlog[1] = brief.Backlog[1], brief.Backlog[0]
	brief.Routing.AvailableRuntimes[0], brief.Routing.AvailableRuntimes[1] = brief.Routing.AvailableRuntimes[1], brief.Routing.AvailableRuntimes[0]
	brief.TaskTypes[0], brief.TaskTypes[1] = brief.TaskTypes[1], brief.TaskTypes[0]
	orders.Orders[0], orders.Orders[2] = orders.Orders[2], orders.Orders[0]
	orders.Orders[1].Stages[0].Status = StageStatusActive

	after, err := tc.loop.scheduleDecisionDigest(brief, orders)
	if err != nil {
		t.Fatalf("digest after schedule-only changes: %v", err)
	}
	if after != before {
		t.Fatalf("schedule-only lifecycle or input ordering changed digest:\n before %s\n after  %s", before, after)
	}
}

func TestScheduleDecisionDigestChangesForEveryAdmittedInput(t *testing.T) {
	logger, _ := newTestLogger()
	tc := newTestLoop(t, logger)
	baseBrief := mise.Brief{
		Backlog:   []adapter.BacklogItem{{ID: "1", Title: "first", Status: adapter.BacklogStatusOpen}},
		Resources: mise.ResourceSnapshot{MaxConcurrency: 2},
		Routing:   mise.RoutingSnapshot{Defaults: mise.RoutingPolicy{Provider: "codex", Model: "gpt-5"}},
		TaskTypes: []mise.TaskTypeSummary{{Key: "execute", Schedule: "ready work"}},
		Warnings:  []string{"one warning"},
	}
	baseOrders := OrdersFile{Orders: []Order{testOrder("work-1", "execute", "execute", "codex", "gpt-5")}}
	base, err := tc.loop.scheduleDecisionDigest(baseBrief, baseOrders)
	if err != nil {
		t.Fatalf("base digest: %v", err)
	}

	assertChanged := func(name string, brief mise.Brief, orders OrdersFile) {
		t.Helper()
		t.Run(name, func(t *testing.T) {
			got, err := tc.loop.scheduleDecisionDigest(brief, orders)
			if err != nil {
				t.Fatalf("digest: %v", err)
			}
			if got == base {
				t.Fatalf("%s did not change decision digest", name)
			}
		})
	}

	changedBacklog := baseBrief
	changedBacklog.Backlog = []adapter.BacklogItem{{ID: "1", Title: "changed", Status: adapter.BacklogStatusOpen}}
	assertChanged("provider backlog", changedBacklog, baseOrders)

	changedOrder := baseOrders
	changedOrder.Orders = []Order{testOrder("work-2", "execute", "execute", "codex", "gpt-5")}
	assertChanged("active non-schedule order", baseBrief, changedOrder)

	changedRouting := baseBrief
	changedRouting.Routing.Defaults.Model = "gpt-6"
	assertChanged("routing", changedRouting, baseOrders)

	changedTaskTypes := baseBrief
	changedTaskTypes.TaskTypes = []mise.TaskTypeSummary{{Key: "execute", Schedule: "new scheduling rule"}}
	assertChanged("task types", changedTaskTypes, baseOrders)

	if err := tc.loop.events.Emit("ci.failed", map[string]string{"branch": "main", "reason": "lint error"}); err != nil {
		t.Fatalf("emit external event: %v", err)
	}
	assertChanged("external event", baseBrief, baseOrders)
}

func TestEmptyScheduleMemoSuppressesUnchangedCyclesAndAdmitsBacklogChange(t *testing.T) {
	logger, handler := newTestLogger()
	brief := mise.Brief{
		Backlog:   []adapter.BacklogItem{{ID: "1", Title: "first", Status: adapter.BacklogStatusOpen}},
		Resources: mise.ResourceSnapshot{MaxConcurrency: 1},
	}
	tc := newTestLoop(t, logger, func(opts *testLoopOpts) { opts.brief = &brief })
	if err := writeOrdersAtomic(tc.ordersPath, bootstrapScheduleOrder(tc.loop.config)); err != nil {
		t.Fatalf("seed schedule order: %v", err)
	}
	if err := os.WriteFile(tc.loop.deps.OrdersNextFile, []byte(`{"orders":[]}`), 0o644); err != nil {
		t.Fatalf("write empty proposal: %v", err)
	}

	if _, _, err := tc.loop.prepareOrdersForCycle(brief, nil, true); err != nil {
		t.Fatalf("promote empty decision: %v", err)
	}
	if _, exists, err := tc.loop.readScheduleEmptyMemo(); err != nil || !exists {
		t.Fatalf("empty memo exists=%v err=%v", exists, err)
	}
	if err := tc.loop.writeOrdersState(OrdersFile{}); err != nil {
		t.Fatalf("simulate schedule completion: %v", err)
	}
	if err := tc.loop.events.Emit(event.LoopEventScheduleCompleted, ScheduleCompletedPayload{SessionID: "schedule-1"}); err != nil {
		t.Fatalf("emit schedule completion: %v", err)
	}

	for cycle := 0; cycle < 3; cycle++ {
		_, shouldContinue, err := tc.loop.prepareOrdersForCycle(brief, nil, true)
		if err != nil {
			t.Fatalf("unchanged cycle %d: %v", cycle, err)
		}
		if shouldContinue {
			t.Fatalf("unchanged cycle %d continued, want idle", cycle)
		}
	}
	if got := handler.countMessage("orders empty, bootstrapping schedule"); got != 0 {
		t.Fatalf("unchanged decision spawned %d schedules, want 0", got)
	}

	changed := brief
	changed.Backlog = []adapter.BacklogItem{
		{ID: "1", Title: "first", Status: adapter.BacklogStatusOpen},
		{ID: "2", Title: "new work", Status: adapter.BacklogStatusOpen},
	}
	_, shouldContinue, err := tc.loop.prepareOrdersForCycle(changed, nil, true)
	if err != nil {
		t.Fatalf("changed backlog cycle: %v", err)
	}
	if !shouldContinue {
		t.Fatal("changed backlog stayed idle, want one schedule order")
	}
	orders, err := readOrders(tc.ordersPath)
	if err != nil {
		t.Fatalf("read changed orders: %v", err)
	}
	if len(orders.Orders) != 1 || !isScheduleOrder(orders.Orders[0]) {
		t.Fatalf("changed backlog orders = %#v, want one schedule order", orders.Orders)
	}
	if got := handler.countMessage("orders empty, bootstrapping schedule"); got != 1 {
		t.Fatalf("changed decision spawned %d schedules, want exactly 1", got)
	}
}

func TestRestartDefersScheduleUntilMemoCanBeComparedWithProviderState(t *testing.T) {
	logger, _ := newTestLogger()
	brief := mise.Brief{
		Backlog:   []adapter.BacklogItem{{ID: "1", Title: "first", Status: adapter.BacklogStatusOpen}},
		Resources: mise.ResourceSnapshot{MaxConcurrency: 1},
	}
	tc := newTestLoop(t, logger, func(opts *testLoopOpts) { opts.brief = &brief })
	if err := tc.loop.writeScheduleEmptyMemo(brief, OrdersFile{}); err != nil {
		t.Fatalf("write memo: %v", err)
	}
	if err := writeOrdersAtomic(tc.ordersPath, OrdersFile{}); err != nil {
		t.Fatalf("write empty orders: %v", err)
	}

	restarted := New(tc.projectDir, "noodle", tc.loop.config, Dependencies{
		Runtimes:   tc.loop.deps.Runtimes,
		Worktree:   tc.worktree,
		Adapter:    tc.adapter,
		Mise:       tc.mise,
		Monitor:    fakeMonitor{},
		Registry:   tc.loop.registry,
		Now:        time.Now,
		OrdersFile: tc.ordersPath,
	})
	if err := restarted.loadOrdersState(); err != nil {
		t.Fatalf("load orders: %v", err)
	}
	if err := restarted.reconcile(context.Background()); err != nil {
		t.Fatalf("reconcile restart: %v", err)
	}
	orders, err := readOrders(tc.ordersPath)
	if err != nil {
		t.Fatalf("read orders after reconcile: %v", err)
	}
	if hasScheduleOrder(orders) {
		t.Fatalf("restart injected schedule before provider comparison: %#v", orders.Orders)
	}

	_, shouldContinue, err := restarted.prepareOrdersForCycle(brief, nil, true)
	if err != nil {
		t.Fatalf("prepare after restart: %v", err)
	}
	if shouldContinue {
		t.Fatal("unchanged restart decision continued, want idle")
	}
	if _, err := os.Stat(filepath.Join(tc.runtimeDir, "schedule-empty-decision.json")); err != nil {
		t.Fatalf("memo not preserved across restart: %v", err)
	}
}

func TestChefSteerBypassesMatchingEmptyDecisionMemo(t *testing.T) {
	logger, _ := newTestLogger()
	brief := mise.Brief{
		Backlog:   []adapter.BacklogItem{{ID: "1", Title: "first", Status: adapter.BacklogStatusOpen}},
		Resources: mise.ResourceSnapshot{MaxConcurrency: 1},
	}
	tc := newTestLoop(t, logger, func(opts *testLoopOpts) { opts.brief = &brief })
	if err := tc.loop.writeScheduleEmptyMemo(brief, OrdersFile{}); err != nil {
		t.Fatalf("write memo: %v", err)
	}
	if err := tc.loop.writeOrdersState(OrdersFile{}); err != nil {
		t.Fatalf("write empty orders: %v", err)
	}

	if err := tc.loop.steer(ScheduleTaskKey(), "prioritize the outage"); err != nil {
		t.Fatalf("chef steer: %v", err)
	}
	orders, err := tc.loop.currentOrders()
	if err != nil {
		t.Fatalf("read orders: %v", err)
	}
	if len(orders.Orders) != 1 || !isScheduleOrder(orders.Orders[0]) {
		t.Fatalf("chef steer orders = %#v, want one schedule order", orders.Orders)
	}
	if orders.Orders[0].Rationale != "Chef steer: prioritize the outage" {
		t.Fatalf("chef steer rationale = %q", orders.Orders[0].Rationale)
	}
	if _, exists, err := tc.loop.readScheduleEmptyMemo(); err != nil || exists {
		t.Fatalf("chef steer left empty memo exists=%v err=%v", exists, err)
	}
}
