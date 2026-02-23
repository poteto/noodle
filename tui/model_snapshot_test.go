package tui

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/poteto/noodle/event"
)

func TestMapEventLinesPromptAction(t *testing.T) {
	payload, err := json.Marshal(map[string]any{
		"tool":    "prompt",
		"action":  "prompt_injected",
		"message": "line one\nline two",
	})
	if err != nil {
		t.Fatalf("encode payload: %v", err)
	}

	lines := mapEventLines([]event.Event{
		{
			Type:      event.EventAction,
			Payload:   payload,
			Timestamp: time.Date(2026, 2, 23, 0, 0, 0, 0, time.UTC),
			SessionID: "cook-a",
		},
	})

	if len(lines) != 1 {
		t.Fatalf("line count = %d, want 1", len(lines))
	}
	if lines[0].Label != "Prompt" {
		t.Fatalf("label = %q, want Prompt", lines[0].Label)
	}
	if lines[0].Category != traceFilterThink {
		t.Fatalf("category = %q, want %q", lines[0].Category, traceFilterThink)
	}
	if lines[0].Body != "line one\nline two" {
		t.Fatalf("body = %q", lines[0].Body)
	}
}
