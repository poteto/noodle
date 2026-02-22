package model

import (
	"encoding/json"
	"testing"
)

func TestCookJSONRoundTrip(t *testing.T) {
	parent := AgentID("cook-parent")
	input := Cook{
		ID:     AgentID("cook-42"),
		Status: CookStatusRunning,
		Parent: &parent,
		Policy: ModelPolicy{
			Provider:       ProviderClaude,
			Model:          "claude-sonnet-4-6",
			ReasoningLevel: "medium",
		},
	}

	encoded, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal cook: %v", err)
	}

	var got Cook
	if err := json.Unmarshal(encoded, &got); err != nil {
		t.Fatalf("unmarshal cook: %v", err)
	}

	if got.ID != input.ID {
		t.Fatalf("id mismatch: got %q want %q", got.ID, input.ID)
	}
	if got.Status != input.Status {
		t.Fatalf("status mismatch: got %q want %q", got.Status, input.Status)
	}
	if got.Parent == nil || *got.Parent != parent {
		t.Fatalf("parent mismatch: got %#v want %q", got.Parent, parent)
	}
	if got.Policy != input.Policy {
		t.Fatalf("policy mismatch: got %#v want %#v", got.Policy, input.Policy)
	}
}

func TestCookStatusValues(t *testing.T) {
	want := []CookStatus{
		CookStatusSpawning,
		CookStatusRunning,
		CookStatusCompleted,
		CookStatusFailed,
		CookStatusKilled,
	}
	if len(want) != 5 {
		t.Fatalf("unexpected status count: got %d", len(want))
	}
}
