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
	tests := []struct {
		name string
		got  CookStatus
		want string
	}{
		{name: "spawning", got: CookStatusSpawning, want: "spawning"},
		{name: "running", got: CookStatusRunning, want: "running"},
		{name: "completed", got: CookStatusCompleted, want: "completed"},
		{name: "failed", got: CookStatusFailed, want: "failed"},
		{name: "killed", got: CookStatusKilled, want: "killed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.got) != tt.want {
				t.Fatalf("status value = %q, want %q", tt.got, tt.want)
			}
		})
	}
}
