package tui

import (
	"encoding/json"
	"os"
	"path/filepath"
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

func TestReadQueuePreservesTaskMetadata(t *testing.T) {
	path := filepath.Join(t.TempDir(), "queue.json")
	payload := `{
  "items": [
    {
      "id": "verify-1",
      "task_key": "verify",
      "title": "Run CI checks",
      "provider": "claude",
      "model": "claude-sonnet-4-6",
      "skill": "verify",
      "rationale": "post-execute gate"
    }
  ]
}`
	if err := os.WriteFile(path, []byte(payload), 0o644); err != nil {
		t.Fatalf("write queue: %v", err)
	}

	items, err := readQueue(path)
	if err != nil {
		t.Fatalf("read queue: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("item count = %d", len(items))
	}
	if items[0].TaskKey != "verify" {
		t.Fatalf("task key = %q", items[0].TaskKey)
	}
	if items[0].Rationale != "post-execute gate" {
		t.Fatalf("rationale = %q", items[0].Rationale)
	}
}
