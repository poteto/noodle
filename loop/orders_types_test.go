package loop

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/poteto/noodle/internal/orderx"
)

func TestStageJSONRoundTrip(t *testing.T) {
	original := Stage{
		TaskKey:  "execute",
		Prompt:   "implement feature X",
		Skill:    "execute",
		Provider: "claude",
		Model:    "claude-opus-4-6",
		Runtime:  "sprites",
		Status:   StageStatusPending,
		Extra: map[string]json.RawMessage{
			"priority": json.RawMessage(`42`),
			"tags":     json.RawMessage(`["urgent","backend"]`),
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded Stage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.TaskKey != original.TaskKey {
		t.Errorf("TaskKey = %q, want %q", decoded.TaskKey, original.TaskKey)
	}
	if decoded.Prompt != original.Prompt {
		t.Errorf("Prompt = %q, want %q", decoded.Prompt, original.Prompt)
	}
	if decoded.Skill != original.Skill {
		t.Errorf("Skill = %q, want %q", decoded.Skill, original.Skill)
	}
	if decoded.Provider != original.Provider {
		t.Errorf("Provider = %q, want %q", decoded.Provider, original.Provider)
	}
	if decoded.Model != original.Model {
		t.Errorf("Model = %q, want %q", decoded.Model, original.Model)
	}
	if decoded.Runtime != original.Runtime {
		t.Errorf("Runtime = %q, want %q", decoded.Runtime, original.Runtime)
	}
	if decoded.Status != original.Status {
		t.Errorf("Status = %q, want %q", decoded.Status, original.Status)
	}
	if string(decoded.Extra["priority"]) != string(original.Extra["priority"]) {
		t.Errorf("Extra[priority] = %s, want %s", decoded.Extra["priority"], original.Extra["priority"])
	}
	if string(decoded.Extra["tags"]) != string(original.Extra["tags"]) {
		t.Errorf("Extra[tags] = %s, want %s", decoded.Extra["tags"], original.Extra["tags"])
	}
}

func TestOrderJSONRoundTrip(t *testing.T) {
	original := Order{
		ID:        "order-1",
		Title:     "Implement auth",
		Plan:      []string{"step 1", "step 2"},
		Rationale: "needed for security",
		Stages: []Stage{
			{TaskKey: "execute", Provider: "claude", Model: "claude-opus-4-6", Status: StageStatusPending},
			{TaskKey: "execute", Provider: "claude", Model: "claude-opus-4-6", Status: StageStatusActive},
		},
		Status: OrderStatusActive,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded Order
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.ID != original.ID {
		t.Errorf("ID = %q, want %q", decoded.ID, original.ID)
	}
	if decoded.Title != original.Title {
		t.Errorf("Title = %q, want %q", decoded.Title, original.Title)
	}
	if len(decoded.Plan) != len(original.Plan) {
		t.Fatalf("Plan len = %d, want %d", len(decoded.Plan), len(original.Plan))
	}
	if decoded.Rationale != original.Rationale {
		t.Errorf("Rationale = %q, want %q", decoded.Rationale, original.Rationale)
	}
	if len(decoded.Stages) != 2 {
		t.Fatalf("Stages len = %d, want 2", len(decoded.Stages))
	}
	if decoded.Status != OrderStatusActive {
		t.Errorf("Status = %q, want %q", decoded.Status, OrderStatusActive)
	}
}

func TestOrdersFileJSONRoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	original := OrdersFile{
		GeneratedAt: now,
		Orders: []Order{
			{
				ID:     "1",
				Title:  "first order",
				Status: OrderStatusActive,
				Stages: []Stage{
					{Provider: "claude", Model: "claude-opus-4-6", Status: StageStatusPending},
					{Provider: "claude", Model: "claude-opus-4-6", Status: StageStatusCompleted},
				},
			},
			{
				ID:     "2",
				Title:  "second order",
				Status: OrderStatusFailed,
				Stages: []Stage{
					{Provider: "claude", Model: "claude-opus-4-6", Status: StageStatusFailed},
				},
			},
		},
		ActionNeeded: []string{"review order 1"},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded OrdersFile
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if !decoded.GeneratedAt.Equal(original.GeneratedAt) {
		t.Errorf("GeneratedAt = %v, want %v", decoded.GeneratedAt, original.GeneratedAt)
	}
	if len(decoded.Orders) != 2 {
		t.Fatalf("Orders len = %d, want 2", len(decoded.Orders))
	}
	if len(decoded.ActionNeeded) != 1 {
		t.Fatalf("ActionNeeded len = %d, want 1", len(decoded.ActionNeeded))
	}
}

func TestOrdersFileMixedStageStatuses(t *testing.T) {
	of := OrdersFile{
		Orders: []Order{
			{
				ID:     "mixed",
				Status: OrderStatusActive,
				Stages: []Stage{
					{Provider: "claude", Model: "claude-opus-4-6", Status: StageStatusCompleted},
					{Provider: "claude", Model: "claude-opus-4-6", Status: StageStatusActive},
					{Provider: "claude", Model: "claude-opus-4-6", Status: StageStatusPending},
					{Provider: "claude", Model: "claude-opus-4-6", Status: StageStatusFailed},
					{Provider: "claude", Model: "claude-opus-4-6", Status: StageStatusCancelled},
				},
			},
		},
	}

	data, err := json.Marshal(of)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded OrdersFile
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	statuses := []orderx.StageStatus{
		StageStatusCompleted, StageStatusActive, StageStatusPending,
		StageStatusFailed, StageStatusCancelled,
	}
	for i, want := range statuses {
		if got := decoded.Orders[0].Stages[i].Status; got != want {
			t.Errorf("stage[%d].Status = %q, want %q", i, got, want)
		}
	}
}

func TestStageExtraRoundTrip(t *testing.T) {
	complexJSON := json.RawMessage(`{"nested":{"key":"value"},"arr":[1,2,3],"null_val":null}`)
	stage := Stage{
		Provider: "claude",
		Model:    "claude-opus-4-6",
		Status:   StageStatusPending,
		Extra: map[string]json.RawMessage{
			"complex": complexJSON,
			"number":  json.RawMessage(`99.5`),
			"bool":    json.RawMessage(`true`),
			"string":  json.RawMessage(`"hello"`),
		},
	}

	data, err := json.Marshal(stage)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded Stage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	for key, original := range stage.Extra {
		got, ok := decoded.Extra[key]
		if !ok {
			t.Errorf("Extra[%q] missing after round-trip", key)
			continue
		}
		if string(got) != string(original) {
			t.Errorf("Extra[%q] = %s, want %s", key, got, original)
		}
	}
}

func TestStageStatusConstants(t *testing.T) {
	tests := []struct {
		name string
		got  orderx.StageStatus
		want orderx.StageStatus
	}{
		{"pending", StageStatusPending, "pending"},
		{"active", StageStatusActive, "active"},
		{"completed", StageStatusCompleted, "completed"},
		{"failed", StageStatusFailed, "failed"},
		{"cancelled", StageStatusCancelled, "cancelled"},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("StageStatus%s = %q, want %q", tt.name, tt.got, tt.want)
		}
	}
}

func TestOrderStatusConstants(t *testing.T) {
	tests := []struct {
		name string
		got  orderx.OrderStatus
		want orderx.OrderStatus
	}{
		{"active", OrderStatusActive, "active"},
		{"completed", OrderStatusCompleted, "completed"},
		{"failed", OrderStatusFailed, "failed"},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("OrderStatus%s = %q, want %q", tt.name, tt.got, tt.want)
		}
	}
}

func TestValidateOrderStatus(t *testing.T) {
	valid := []orderx.OrderStatus{OrderStatusActive, OrderStatusCompleted, OrderStatusFailed}
	for _, s := range valid {
		if err := orderx.ValidateOrderStatus(s); err != nil {
			t.Errorf("ValidateOrderStatus(%q) = %v, want nil", s, err)
		}
	}

	if err := orderx.ValidateOrderStatus(""); err == nil {
		t.Error("ValidateOrderStatus(\"\") = nil, want error")
	}

	if err := orderx.ValidateOrderStatus("bogus"); err == nil {
		t.Error("ValidateOrderStatus(\"bogus\") = nil, want error")
	}
}

func TestValidateStageStatus(t *testing.T) {
	valid := []orderx.StageStatus{StageStatusPending, StageStatusActive, StageStatusCompleted, StageStatusFailed, StageStatusCancelled}
	for _, s := range valid {
		if err := orderx.ValidateStageStatus(s); err != nil {
			t.Errorf("ValidateStageStatus(%q) = %v, want nil", s, err)
		}
	}

	if err := orderx.ValidateStageStatus(""); err == nil {
		t.Error("ValidateStageStatus(\"\") = nil, want error")
	}

	if err := orderx.ValidateStageStatus("bogus"); err == nil {
		t.Error("ValidateStageStatus(\"bogus\") = nil, want error")
	}
}
