package loop

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/poteto/noodle/config"
	loopruntime "github.com/poteto/noodle/runtime"
)

func TestHasNonScheduleOrders(t *testing.T) {
	testCases := []struct {
		name   string
		orders OrdersFile
		want   bool
	}{
		{
			name:   "empty orders",
			orders: OrdersFile{},
			want:   false,
		},
		{
			name: "only schedule order",
			orders: OrdersFile{
				Orders: []Order{
					{ID: "schedule", Status: OrderStatusActive, Stages: []Stage{{TaskKey: "schedule", Status: StageStatusPending}}},
				},
			},
			want: false,
		},
		{
			name: "one non-schedule order",
			orders: OrdersFile{
				Orders: []Order{
					{ID: "42", Status: OrderStatusActive, Stages: []Stage{{TaskKey: "execute", Status: StageStatusPending}}},
				},
			},
			want: true,
		},
		{
			name: "mixed schedule and non-schedule orders",
			orders: OrdersFile{
				Orders: []Order{
					{ID: "schedule", Status: OrderStatusActive, Stages: []Stage{{TaskKey: "schedule", Status: StageStatusPending}}},
					{ID: "42", Status: OrderStatusActive, Stages: []Stage{{TaskKey: "execute", Status: StageStatusPending}}},
				},
			},
			want: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got := hasNonScheduleOrders(testCase.orders)
			if got != testCase.want {
				t.Fatalf("hasNonScheduleOrders() = %t, want %t", got, testCase.want)
			}
		})
	}
}

func TestBootstrapScheduleOrderCreatesSingleStageOrder(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Routing.Defaults.Provider = "claude"
	cfg.Routing.Defaults.Model = "claude-opus-4-6"

	of := bootstrapScheduleOrder(cfg)
	if len(of.Orders) != 1 {
		t.Fatalf("orders count = %d, want 1", len(of.Orders))
	}
	order := of.Orders[0]
	if order.ID != "schedule" {
		t.Fatalf("order ID = %q, want %q", order.ID, "schedule")
	}
	if order.Status != OrderStatusActive {
		t.Fatalf("order status = %q, want %q", order.Status, OrderStatusActive)
	}
	if len(order.Stages) != 1 {
		t.Fatalf("stages count = %d, want 1", len(order.Stages))
	}
	stage := order.Stages[0]
	if stage.TaskKey != "schedule" {
		t.Fatalf("stage task_key = %q, want %q", stage.TaskKey, "schedule")
	}
	if stage.Status != StageStatusPending {
		t.Fatalf("stage status = %q, want %q", stage.Status, StageStatusPending)
	}
	if stage.Provider != "claude" {
		t.Fatalf("stage provider = %q, want %q", stage.Provider, "claude")
	}
	if stage.Model != "claude-opus-4-6" {
		t.Fatalf("stage model = %q, want %q", stage.Model, "claude-opus-4-6")
	}
}

func TestIsScheduleOrder(t *testing.T) {
	if !isScheduleOrder(Order{ID: "schedule"}) {
		t.Fatal("expected schedule order to be recognized")
	}
	if !isScheduleOrder(Order{ID: "Schedule"}) {
		t.Fatal("expected case-insensitive match")
	}
	if isScheduleOrder(Order{ID: "42"}) {
		t.Fatal("expected non-schedule order to not match")
	}
}

func TestBuildSchedulePromptIncludesOrdersSchema(t *testing.T) {
	order := Order{
		ID:     "schedule",
		Status: OrderStatusActive,
		Stages: []Stage{{TaskKey: "schedule", Skill: "schedule", Status: StageStatusPending}},
	}
	taskTypes := buildOrderTaskTypesPrompt([]TaskType{
		{Key: "execute", Schedule: "When ready"},
	})
	prompt := buildSchedulePrompt("schedule", taskTypes, order, "", "/tmp/test/.noodle", "")

	if !strings.Contains(prompt, "/tmp/test/.noodle/orders-next.json") {
		t.Fatal("prompt should reference absolute path to orders-next.json")
	}
	if !strings.Contains(prompt, "/tmp/test/.noodle/mise.json") {
		t.Fatal("prompt should reference absolute path to mise.json")
	}
	if strings.Contains(prompt, "recovery pipeline") {
		t.Fatal("prompt should not mention recovery pipeline stages")
	}
	if !strings.Contains(prompt, "execute: When ready") {
		t.Fatal("prompt should include task types")
	}
}

func TestBuildSchedulePromptIncludesPromotionError(t *testing.T) {
	order := Order{
		ID:     "schedule",
		Status: OrderStatusActive,
		Stages: []Stage{{TaskKey: "schedule", Skill: "schedule", Status: StageStatusPending}},
	}
	taskTypes := buildOrderTaskTypesPrompt(nil)
	prompt := buildSchedulePrompt("schedule", taskTypes, order, "", "/tmp/test/.noodle", "unknown field on_failure")

	if !strings.Contains(prompt, "PREVIOUS ORDERS REJECTED") {
		t.Fatal("prompt should include rejection header when promotion error is set")
	}
	if !strings.Contains(prompt, "unknown field on_failure") {
		t.Fatal("prompt should include the specific validation error")
	}

	// No error — should not include rejection header.
	promptClean := buildSchedulePrompt("schedule", taskTypes, order, "", "/tmp/test/.noodle", "")
	if strings.Contains(promptClean, "PREVIOUS ORDERS REJECTED") {
		t.Fatal("prompt should not include rejection header when no promotion error")
	}
}

func TestScheduleOrderWithChefGuidance(t *testing.T) {
	cfg := config.DefaultConfig()
	order := scheduleOrder(cfg, "focus on auth refactor")
	if order.Rationale != "Chef steer: focus on auth refactor" {
		t.Fatalf("rationale = %q, want Chef steer prefix", order.Rationale)
	}
}

func TestSpawnSchedulePersistsActiveStageStatus(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	ordersPath := filepath.Join(runtimeDir, "orders.json")

	cfg := config.DefaultConfig()
	order := scheduleOrder(cfg, "")
	if err := writeOrdersAtomic(ordersPath, OrdersFile{Orders: []Order{order}}); err != nil {
		t.Fatalf("write orders: %v", err)
	}

	rt := newMockRuntime()
	l := New(projectDir, "noodle", cfg, Dependencies{
		Runtimes:   map[string]loopruntime.Runtime{"process": rt},
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		OrdersFile: ordersPath,
	})

	if err := l.spawnSchedule(context.Background(), order, 0, ""); err != nil {
		t.Fatalf("spawnSchedule: %v", err)
	}
	if len(rt.calls) != 1 {
		t.Fatalf("dispatch calls = %d, want 1", len(rt.calls))
	}
	if _, ok := l.cooks.activeCooksByOrder[scheduleOrderID]; !ok {
		t.Fatal("expected schedule order in activeCooksByOrder")
	}

	updated, err := readOrders(ordersPath)
	if err != nil {
		t.Fatalf("read orders: %v", err)
	}
	if len(updated.Orders) != 1 || len(updated.Orders[0].Stages) != 1 {
		t.Fatalf("unexpected orders shape: %+v", updated.Orders)
	}
	if updated.Orders[0].Stages[0].Status != StageStatusActive {
		t.Fatalf("schedule stage status = %q, want %q", updated.Orders[0].Stages[0].Status, StageStatusActive)
	}
}

func TestSpawnScheduleRetryableDispatchFailureResetsPending(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	ordersPath := filepath.Join(runtimeDir, "orders.json")

	cfg := config.DefaultConfig()
	cfg.Runtime.Default = "sprites"
	order := scheduleOrder(cfg, "")
	if err := writeOrdersAtomic(ordersPath, OrdersFile{Orders: []Order{order}}); err != nil {
		t.Fatalf("write orders: %v", err)
	}

	spritesRT := newMockRuntime()
	spritesRT.dispatchErr = errors.New("sprites runtime temporarily unavailable")

	l := New(projectDir, "noodle", cfg, Dependencies{
		Runtimes:   map[string]loopruntime.Runtime{"sprites": spritesRT},
		Worktree:   &fakeWorktree{},
		Adapter:    &fakeAdapterRunner{},
		Mise:       &fakeMise{},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		OrdersFile: ordersPath,
	})

	if err := l.spawnSchedule(context.Background(), order, 0, ""); err != nil {
		t.Fatalf("spawnSchedule should treat retryable dispatch failures as recoverable: %v", err)
	}
	if _, ok := l.cooks.activeCooksByOrder[scheduleOrderID]; ok {
		t.Fatal("schedule cook should not be active after retryable dispatch failure")
	}

	updated, err := readOrders(ordersPath)
	if err != nil {
		t.Fatalf("read orders: %v", err)
	}
	if len(updated.Orders) != 1 || len(updated.Orders[0].Stages) != 1 {
		t.Fatalf("unexpected orders shape: %+v", updated.Orders)
	}
	if updated.Orders[0].Stages[0].Status != StageStatusPending {
		t.Fatalf("schedule stage status = %q, want %q", updated.Orders[0].Stages[0].Status, StageStatusPending)
	}
}
