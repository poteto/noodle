package loop

import (
	"strings"
	"testing"

	"github.com/poteto/noodle/config"
)

func TestHasNonScheduleItems(t *testing.T) {
	testCases := []struct {
		name  string
		queue Queue
		want  bool
	}{
		{
			name:  "empty queue",
			queue: Queue{},
			want:  false,
		},
		{
			name: "only schedule item",
			queue: Queue{
				Items: []QueueItem{
					{ID: "schedule"},
				},
			},
			want: false,
		},
		{
			name: "one non-schedule item",
			queue: Queue{
				Items: []QueueItem{
					{ID: "42"},
				},
			},
			want: true,
		},
		{
			name: "mixed schedule and non-schedule items",
			queue: Queue{
				Items: []QueueItem{
					{ID: "schedule"},
					{ID: "42"},
				},
			},
			want: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got := hasNonScheduleItems(testCase.queue)
			if got != testCase.want {
				t.Fatalf("hasNonScheduleItems() = %t, want %t", got, testCase.want)
			}
		})
	}
}

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

func TestFilterStaleScheduleItems(t *testing.T) {
	testCases := []struct {
		name    string
		queue   Queue
		wantIDs []string
	}{
		{
			name:    "empty queue",
			queue:   Queue{},
			wantIDs: nil,
		},
		{
			name: "only schedule item",
			queue: Queue{
				Items: []QueueItem{
					{ID: "schedule"},
				},
			},
			wantIDs: []string{},
		},
		{
			name: "non-schedule items unchanged",
			queue: Queue{
				Items: []QueueItem{
					{ID: "42"},
					{ID: "43"},
				},
			},
			wantIDs: []string{"42", "43"},
		},
		{
			name: "mixed items remove schedule",
			queue: Queue{
				Items: []QueueItem{
					{ID: "42"},
					{ID: "schedule"},
					{ID: "43"},
				},
			},
			wantIDs: []string{"42", "43"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			filtered := filterStaleScheduleItems(testCase.queue)
			if len(filtered.Items) != len(testCase.wantIDs) {
				t.Fatalf("len(items) = %d, want %d", len(filtered.Items), len(testCase.wantIDs))
			}
			for index, expectedID := range testCase.wantIDs {
				if got := filtered.Items[index].ID; got != expectedID {
					t.Fatalf("items[%d].ID = %q, want %q", index, got, expectedID)
				}
			}
		})
	}
}

func TestBuildSchedulePromptIncludesOrdersSchema(t *testing.T) {
	item := QueueItem{
		ID:    "schedule",
		Skill: "schedule",
	}
	taskTypes := buildOrderTaskTypesPrompt([]TaskType{
		{Key: "execute", Schedule: "When ready"},
	})
	prompt := buildSchedulePrompt("schedule", taskTypes, item, "")

	if !strings.Contains(prompt, "orders-next.json") {
		t.Fatal("prompt should reference orders-next.json")
	}
	if !strings.Contains(prompt, "on_failure") {
		t.Fatal("prompt should mention on_failure stages")
	}
	if !strings.Contains(prompt, "execute: When ready") {
		t.Fatal("prompt should include task types")
	}
}

func TestScheduleOrderWithChefGuidance(t *testing.T) {
	cfg := config.DefaultConfig()
	order := scheduleOrder(cfg, "focus on auth refactor")
	if order.Rationale != "Chef steer: focus on auth refactor" {
		t.Fatalf("rationale = %q, want Chef steer prefix", order.Rationale)
	}
}
