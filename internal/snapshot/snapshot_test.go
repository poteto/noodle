package snapshot

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/poteto/noodle/event"
)

func TestDeriveHealth(t *testing.T) {
	cases := []struct {
		name       string
		status     string
		explicit   string
		contextPct float64
		idle       int64
		threshold  int64
		want       string
	}{
		{name: "explicit wins", status: "running", explicit: "red", want: "red"},
		{name: "failed is red", status: "failed", want: "red"},
		{name: "stuck is red", status: "stuck", want: "red"},
		{name: "high context is yellow", status: "running", contextPct: 81, want: "yellow"},
		{name: "idle over half threshold is yellow", status: "running", idle: 70, threshold: 120, want: "yellow"},
		{name: "idle over threshold is red", status: "running", idle: 121, threshold: 120, want: "red"},
		{name: "healthy running is green", status: "running", idle: 10, threshold: 120, want: "green"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := deriveHealth(tc.status, tc.explicit, tc.contextPct, tc.idle, tc.threshold)
			if got != tc.want {
				t.Fatalf("deriveHealth() = %q, want %q", got, tc.want)
			}
		})
	}
}

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
	if lines[0].Category != TraceFilterThink {
		t.Fatalf("category = %q, want %q", lines[0].Category, TraceFilterThink)
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
			payload:   map[string]any{"tool": "Skill", "summary": "schedule"},
			wantLabel: "Skill",
			wantBody:  "schedule",
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

func TestReadQueueEvents(t *testing.T) {
	dir := t.TempDir()
	ndjson := `{"at":"2026-02-24T10:00:00Z","type":"queue_drop","target":"item-1","skill":"old-skill","reason":"skill old-skill no longer registered"}
{"at":"2026-02-24T10:01:00Z","type":"registry_rebuild","added":["execute","verify"],"removed":["old-skill"]}
{"at":"2026-02-24T10:02:00Z","type":"bootstrap"}
`
	if err := os.WriteFile(filepath.Join(dir, "queue-events.ndjson"), []byte(ndjson), 0o644); err != nil {
		t.Fatalf("write queue-events: %v", err)
	}

	events := readQueueEvents(dir)
	if len(events) != 3 {
		t.Fatalf("event count = %d, want 3", len(events))
	}

	// queue_drop
	if events[0].Category != "queue_drop" {
		t.Errorf("event[0] category = %q, want queue_drop", events[0].Category)
	}
	if events[0].Label != "Dropped" {
		t.Errorf("event[0] label = %q, want Dropped", events[0].Label)
	}
	if events[0].Body != "Dropped item item-1: skill old-skill no longer registered" {
		t.Errorf("event[0] body = %q", events[0].Body)
	}

	// registry_rebuild
	if events[1].Category != "registry_rebuild" {
		t.Errorf("event[1] category = %q, want registry_rebuild", events[1].Category)
	}
	if events[1].Label != "Rebuild" {
		t.Errorf("event[1] label = %q, want Rebuild", events[1].Label)
	}
	wantRebuild := "Registry rebuilt — added: [execute verify], removed: [old-skill]"
	if events[1].Body != wantRebuild {
		t.Errorf("event[1] body = %q, want %q", events[1].Body, wantRebuild)
	}

	// bootstrap
	if events[2].Category != "bootstrap" {
		t.Errorf("event[2] category = %q, want bootstrap", events[2].Category)
	}
	if events[2].Label != "Bootstrap" {
		t.Errorf("event[2] label = %q, want Bootstrap", events[2].Label)
	}
	if events[2].Body != "Creating schedule skill from workflow analysis" {
		t.Errorf("event[2] body = %q", events[2].Body)
	}

	// All should have SessionID "loop"
	for i, ev := range events {
		if ev.SessionID != "loop" {
			t.Errorf("event[%d] session = %q, want loop", i, ev.SessionID)
		}
	}
}

func TestReadQueueEventsSkipsMalformed(t *testing.T) {
	dir := t.TempDir()
	ndjson := `not valid json
{"at":"2026-02-24T10:00:00Z","type":"queue_drop","target":"item-1","skill":"gone","reason":"skill gone no longer registered"}
{broken
{"at":"2026-02-24T10:01:00Z","type":"bootstrap"}
`
	if err := os.WriteFile(filepath.Join(dir, "queue-events.ndjson"), []byte(ndjson), 0o644); err != nil {
		t.Fatalf("write queue-events: %v", err)
	}

	events := readQueueEvents(dir)
	if len(events) != 2 {
		t.Fatalf("event count = %d, want 2 (malformed lines skipped)", len(events))
	}
	if events[0].Category != "queue_drop" {
		t.Errorf("event[0] category = %q, want queue_drop", events[0].Category)
	}
	if events[1].Category != "bootstrap" {
		t.Errorf("event[1] category = %q, want bootstrap", events[1].Category)
	}
}

func TestReadQueueEventsMissingFile(t *testing.T) {
	events := readQueueEvents(t.TempDir())
	if len(events) != 0 {
		t.Fatalf("event count = %d, want 0 for missing file", len(events))
	}
}

func TestReadQueueEventsDropWithoutReason(t *testing.T) {
	dir := t.TempDir()
	ndjson := `{"at":"2026-02-24T10:00:00Z","type":"queue_drop","target":"item-2","skill":"old-skill"}
`
	if err := os.WriteFile(filepath.Join(dir, "queue-events.ndjson"), []byte(ndjson), 0o644); err != nil {
		t.Fatalf("write queue-events: %v", err)
	}

	events := readQueueEvents(dir)
	if len(events) != 1 {
		t.Fatalf("event count = %d, want 1", len(events))
	}
	want := "Dropped item item-2: skill old-skill no longer exists"
	if events[0].Body != want {
		t.Errorf("body = %q, want %q", events[0].Body, want)
	}
}

func TestReadQueueEventsUnknownTypeSkipped(t *testing.T) {
	dir := t.TempDir()
	ndjson := `{"at":"2026-02-24T10:00:00Z","type":"unknown_event","target":"x"}
{"at":"2026-02-24T10:01:00Z","type":"bootstrap"}
`
	if err := os.WriteFile(filepath.Join(dir, "queue-events.ndjson"), []byte(ndjson), 0o644); err != nil {
		t.Fatalf("write queue-events: %v", err)
	}

	events := readQueueEvents(dir)
	if len(events) != 1 {
		t.Fatalf("event count = %d, want 1 (unknown type skipped)", len(events))
	}
	if events[0].Category != "bootstrap" {
		t.Errorf("event[0] category = %q, want bootstrap", events[0].Category)
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
