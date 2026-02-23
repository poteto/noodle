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

func TestFormatActionToolTypes(t *testing.T) {
	cases := []struct {
		name      string
		payload   map[string]any
		wantLabel string
		wantBody  string
	}{
		{
			name:      "skill tool",
			payload:   map[string]any{"tool": "Skill", "summary": "prioritize"},
			wantLabel: "Skill",
			wantBody:  "prioritize",
		},
		{
			name:      "task tool",
			payload:   map[string]any{"tool": "Task", "summary": "spawn explorer"},
			wantLabel: "Task",
			wantBody:  "spawn explorer",
		},
		{
			name:      "read tool",
			payload:   map[string]any{"tool": "Read", "summary": "/path/to/file.go"},
			wantLabel: "Read",
			wantBody:  "/path/to/file.go",
		},
		{
			name:      "bash tool",
			payload:   map[string]any{"tool": "Bash", "summary": "go test ./..."},
			wantLabel: "Bash",
			wantBody:  "go test ./...",
		},
		{
			name:      "think",
			payload:   map[string]any{"tool": "Think", "summary": "analyzing the code"},
			wantLabel: "Think",
			wantBody:  "analyzing the code",
		},
		{
			name:      "prompt",
			payload:   map[string]any{"tool": "Prompt", "summary": "Work on item 15"},
			wantLabel: "Prompt",
			wantBody:  "Work on item 15",
		},
		{
			name:      "legacy message-only payload",
			payload:   map[string]any{"message": "Read /path/to/file"},
			wantLabel: "Think",
			wantBody:  "Read /path/to/file",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw, _ := json.Marshal(tc.payload)
			label, body, _ := formatAction(raw)
			if label != tc.wantLabel {
				t.Errorf("label = %q, want %q", label, tc.wantLabel)
			}
			if body != tc.wantBody {
				t.Errorf("body = %q, want %q", body, tc.wantBody)
			}
		})
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

	qr, err := readQueue(path)
	if err != nil {
		t.Fatalf("read queue: %v", err)
	}
	if len(qr.Items) != 1 {
		t.Fatalf("item count = %d", len(qr.Items))
	}
	if qr.Items[0].TaskKey != "verify" {
		t.Fatalf("task key = %q", qr.Items[0].TaskKey)
	}
	if qr.Items[0].Rationale != "post-execute gate" {
		t.Fatalf("rationale = %q", qr.Items[0].Rationale)
	}
}
