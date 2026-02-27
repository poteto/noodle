package orderx

import (
	"encoding/json"
	"testing"
	"time"
)

func TestQueuexStageJSONRoundTrip(t *testing.T) {
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
	if decoded.Provider != original.Provider {
		t.Errorf("Provider = %q, want %q", decoded.Provider, original.Provider)
	}
	if decoded.Status != original.Status {
		t.Errorf("Status = %q, want %q", decoded.Status, original.Status)
	}
	if string(decoded.Extra["priority"]) != `42` {
		t.Errorf("Extra[priority] = %s, want 42", decoded.Extra["priority"])
	}
}

func TestQueuexOrderJSONRoundTrip(t *testing.T) {
	original := Order{
		ID:        "order-1",
		Title:     "Implement auth",
		Plan:      []string{"step 1", "step 2"},
		Rationale: "needed for security",
		Stages: []Stage{
			{Provider: "claude", Model: "claude-opus-4-6", Status: StageStatusPending},
			{Provider: "claude", Model: "claude-opus-4-6", Status: StageStatusActive},
		},
		Status: OrderStatusActive,
		OnFailure: []Stage{
			{Prompt: "rollback changes", Provider: "claude", Model: "claude-opus-4-6", Status: StageStatusPending},
		},
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
	if len(decoded.Stages) != 2 {
		t.Fatalf("Stages len = %d, want 2", len(decoded.Stages))
	}
	if decoded.Status != OrderStatusActive {
		t.Errorf("Status = %q, want %q", decoded.Status, OrderStatusActive)
	}
	if len(decoded.OnFailure) != 1 {
		t.Fatalf("OnFailure len = %d, want 1", len(decoded.OnFailure))
	}
}

func TestQueuexOrdersFileJSONRoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	original := OrdersFile{
		GeneratedAt: now,
		Orders: []Order{
			{
				ID:     "1",
				Status: OrderStatusActive,
				Stages: []Stage{
					{Provider: "claude", Model: "claude-opus-4-6", Status: StageStatusPending},
					{Provider: "claude", Model: "claude-opus-4-6", Status: StageStatusCompleted},
				},
			},
			{
				ID:     "2",
				Status: OrderStatusFailing,
				Stages: []Stage{
					{Provider: "claude", Model: "claude-opus-4-6", Status: StageStatusFailed},
				},
				OnFailure: []Stage{
					{Provider: "claude", Model: "claude-opus-4-6", Status: StageStatusActive},
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
}

func TestQueuexStageExtraRoundTrip(t *testing.T) {
	complexJSON := json.RawMessage(`{"nested":{"key":"value"},"arr":[1,2,3],"null_val":null}`)
	stage := Stage{
		Provider: "claude",
		Model:    "claude-opus-4-6",
		Status:   StageStatusPending,
		Extra: map[string]json.RawMessage{
			"complex": complexJSON,
			"number":  json.RawMessage(`99.5`),
			"bool":    json.RawMessage(`true`),
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

func TestQueuexOrderOnFailureNilAndEmpty(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		order := Order{
			ID:        "1",
			Status:    OrderStatusActive,
			Stages:    []Stage{{Provider: "claude", Model: "claude-opus-4-6", Status: StageStatusPending}},
			OnFailure: nil,
		}
		data, err := json.Marshal(order)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		var decoded Order
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if decoded.OnFailure != nil {
			t.Errorf("OnFailure = %v, want nil", decoded.OnFailure)
		}
	})

	t.Run("empty", func(t *testing.T) {
		order := Order{
			ID:        "2",
			Status:    OrderStatusActive,
			Stages:    []Stage{{Provider: "claude", Model: "claude-opus-4-6", Status: StageStatusPending}},
			OnFailure: []Stage{},
		}
		data, err := json.Marshal(order)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		var decoded Order
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if decoded.OnFailure != nil {
			t.Errorf("OnFailure = %v, want nil (empty omitted)", decoded.OnFailure)
		}
	})
}

func TestQueuexStageStatusConstants(t *testing.T) {
	tests := []struct {
		got  StageStatus
		want StageStatus
	}{
		{StageStatusPending, "pending"},
		{StageStatusActive, "active"},
		{StageStatusCompleted, "completed"},
		{StageStatusFailed, "failed"},
		{StageStatusCancelled, "cancelled"},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("got %q, want %q", tt.got, tt.want)
		}
	}
}

func TestQueuexOrderStatusConstants(t *testing.T) {
	tests := []struct {
		got  OrderStatus
		want OrderStatus
	}{
		{OrderStatusActive, "active"},
		{OrderStatusCompleted, "completed"},
		{OrderStatusFailed, "failed"},
		{OrderStatusFailing, "failing"},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("got %q, want %q", tt.got, tt.want)
		}
	}
}

func TestQueuexValidateOrderStatus(t *testing.T) {
	valid := []OrderStatus{OrderStatusActive, OrderStatusCompleted, OrderStatusFailed, OrderStatusFailing}
	for _, s := range valid {
		if err := ValidateOrderStatus(s); err != nil {
			t.Errorf("ValidateOrderStatus(%q) = %v, want nil", s, err)
		}
	}

	if err := ValidateOrderStatus(""); err == nil {
		t.Error("ValidateOrderStatus(\"\") = nil, want error")
	}

	if err := ValidateOrderStatus("bogus"); err == nil {
		t.Error("ValidateOrderStatus(\"bogus\") = nil, want error")
	}
}

func TestQueuexValidateStageStatus(t *testing.T) {
	valid := []StageStatus{StageStatusPending, StageStatusActive, StageStatusCompleted, StageStatusFailed, StageStatusCancelled}
	for _, s := range valid {
		if err := ValidateStageStatus(s); err != nil {
			t.Errorf("ValidateStageStatus(%q) = %v, want nil", s, err)
		}
	}

	if err := ValidateStageStatus(""); err == nil {
		t.Error("ValidateStageStatus(\"\") = nil, want error")
	}

	if err := ValidateStageStatus("bogus"); err == nil {
		t.Error("ValidateStageStatus(\"bogus\") = nil, want error")
	}
}
