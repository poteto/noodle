package snapshot

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/poteto/noodle/event"
	"github.com/poteto/noodle/loop"
	"github.com/poteto/noodle/mise"
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

func TestReadLoopEvents(t *testing.T) {
	dir := t.TempDir()
	ndjson := `{"seq":1,"type":"order.dropped","at":"2026-02-24T10:00:00Z","payload":{"order_id":"item-1","reason":"skill old-skill no longer registered"}}
{"seq":2,"type":"registry.rebuilt","at":"2026-02-24T10:01:00Z","payload":{"added":["execute","verify"],"removed":["old-skill"]}}
{"seq":3,"type":"bootstrap.completed","at":"2026-02-24T10:02:00Z"}
{"seq":4,"type":"bootstrap.exhausted","at":"2026-02-24T10:03:00Z","payload":{"reason":"bootstrap exhausted after 3 attempts"}}
{"seq":5,"type":"sync.degraded","at":"2026-02-24T10:04:00Z","payload":{"reason":"sync script failed"}}
`
	if err := os.WriteFile(filepath.Join(dir, "loop-events.ndjson"), []byte(ndjson), 0o644); err != nil {
		t.Fatalf("write loop-events: %v", err)
	}

	events := readLoopEvents(dir)
	if len(events) != 5 {
		t.Fatalf("event count = %d, want 5", len(events))
	}

	// order.dropped
	if events[0].Category != "order_drop" {
		t.Errorf("event[0] category = %q, want order_drop", events[0].Category)
	}
	if events[0].Label != "Dropped" {
		t.Errorf("event[0] label = %q, want Dropped", events[0].Label)
	}
	if events[0].Body != "Dropped order item-1: skill old-skill no longer registered" {
		t.Errorf("event[0] body = %q", events[0].Body)
	}

	// registry.rebuilt
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

	// bootstrap.completed
	if events[2].Category != "bootstrap" {
		t.Errorf("event[2] category = %q, want bootstrap", events[2].Category)
	}
	if events[2].Label != "Bootstrap" {
		t.Errorf("event[2] label = %q, want Bootstrap", events[2].Label)
	}
	if events[2].Body != "Bootstrap completed" {
		t.Errorf("event[2] body = %q", events[2].Body)
	}

	// bootstrap.exhausted
	if events[3].Category != "bootstrap" {
		t.Errorf("event[3] category = %q, want bootstrap", events[3].Category)
	}
	if events[3].Label != "Bootstrap" {
		t.Errorf("event[3] label = %q, want Bootstrap", events[3].Label)
	}
	if events[3].Body != "bootstrap exhausted after 3 attempts" {
		t.Errorf("event[3] body = %q", events[3].Body)
	}

	// sync.degraded
	if events[4].Category != "sync_degraded" {
		t.Errorf("event[4] category = %q, want sync_degraded", events[4].Category)
	}
	if events[4].Label != "Sync" {
		t.Errorf("event[4] label = %q, want Sync", events[4].Label)
	}
	if events[4].Body != "sync script failed" {
		t.Errorf("event[4] body = %q", events[4].Body)
	}

	// All should have SessionID "loop"
	for i, ev := range events {
		if ev.SessionID != "loop" {
			t.Errorf("event[%d] session = %q, want loop", i, ev.SessionID)
		}
	}
}

func TestReadLoopEventsSkipsMalformed(t *testing.T) {
	dir := t.TempDir()
	ndjson := `not valid json
{"seq":1,"type":"order.dropped","at":"2026-02-24T10:00:00Z","payload":{"order_id":"item-1","reason":"skill gone no longer registered"}}
{broken
{"seq":2,"type":"bootstrap.completed","at":"2026-02-24T10:01:00Z"}
`
	if err := os.WriteFile(filepath.Join(dir, "loop-events.ndjson"), []byte(ndjson), 0o644); err != nil {
		t.Fatalf("write loop-events: %v", err)
	}

	events := readLoopEvents(dir)
	if len(events) != 2 {
		t.Fatalf("event count = %d, want 2 (malformed lines skipped)", len(events))
	}
	if events[0].Category != "order_drop" {
		t.Errorf("event[0] category = %q, want order_drop", events[0].Category)
	}
	if events[1].Category != "bootstrap" {
		t.Errorf("event[1] category = %q, want bootstrap", events[1].Category)
	}
}

func TestReadLoopEventsMissingFile(t *testing.T) {
	events := readLoopEvents(t.TempDir())
	if len(events) != 0 {
		t.Fatalf("event count = %d, want 0 for missing file", len(events))
	}
}

func TestReadLoopEventsDropWithoutReason(t *testing.T) {
	dir := t.TempDir()
	ndjson := `{"seq":1,"type":"order.dropped","at":"2026-02-24T10:00:00Z","payload":{"order_id":"item-2"}}
`
	if err := os.WriteFile(filepath.Join(dir, "loop-events.ndjson"), []byte(ndjson), 0o644); err != nil {
		t.Fatalf("write loop-events: %v", err)
	}

	events := readLoopEvents(dir)
	if len(events) != 1 {
		t.Fatalf("event count = %d, want 1", len(events))
	}
	want := "Dropped order item-2"
	if events[0].Body != want {
		t.Errorf("body = %q, want %q", events[0].Body, want)
	}
}

func TestReadLoopEventsUnknownTypeSkipped(t *testing.T) {
	dir := t.TempDir()
	ndjson := `{"seq":1,"type":"unknown.event","at":"2026-02-24T10:00:00Z"}
{"seq":2,"type":"bootstrap.completed","at":"2026-02-24T10:01:00Z"}
`
	if err := os.WriteFile(filepath.Join(dir, "loop-events.ndjson"), []byte(ndjson), 0o644); err != nil {
		t.Fatalf("write loop-events: %v", err)
	}

	events := readLoopEvents(dir)
	if len(events) != 1 {
		t.Fatalf("event count = %d, want 1 (unknown type skipped)", len(events))
	}
	if events[0].Category != "bootstrap" {
		t.Errorf("event[0] category = %q, want bootstrap", events[0].Category)
	}
}

func TestSnapshotSerializationIncludesOrders(t *testing.T) {
	snap := Snapshot{
		UpdatedAt: time.Date(2026, 2, 26, 12, 0, 0, 0, time.UTC),
		LoopState: LoopStateRunning,
		Orders: []Order{
			{
				ID:        "o-1",
				Title:     "Test order",
				Plan:      []string{"plan step"},
				Rationale: "test",
				Stages: []Stage{
					{
						TaskKey:  "execute",
						Prompt:   "do it",
						Skill:    "execute",
						Provider: "claude",
						Model:    "claude-sonnet-4-6",
						Runtime:  "claude-code",
						Status:   "active",
						Extra:    map[string]json.RawMessage{"key": json.RawMessage(`"val"`)},
					},
				},
				Status: "active",
			},
		},
		ActiveOrderIDs: []string{"o-1"},
	}

	data, err := json.Marshal(snap)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var parsed map[string]json.RawMessage
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Verify "orders" field exists (not "queue")
	if _, ok := parsed["orders"]; !ok {
		t.Fatal("serialized snapshot missing 'orders' field")
	}
	if _, ok := parsed["queue"]; ok {
		t.Fatal("serialized snapshot still has 'queue' field")
	}

	// Verify "active_order_ids" field exists (not "active_queue_ids")
	if _, ok := parsed["active_order_ids"]; !ok {
		t.Fatal("serialized snapshot missing 'active_order_ids' field")
	}
	if _, ok := parsed["active_queue_ids"]; ok {
		t.Fatal("serialized snapshot still has 'active_queue_ids' field")
	}

	// Roundtrip: unmarshal back to Snapshot
	var roundtrip Snapshot
	if err := json.Unmarshal(data, &roundtrip); err != nil {
		t.Fatalf("roundtrip unmarshal: %v", err)
	}
	if len(roundtrip.Orders) != 1 {
		t.Fatalf("roundtrip order count = %d", len(roundtrip.Orders))
	}
	order := roundtrip.Orders[0]
	if order.ID != "o-1" {
		t.Errorf("roundtrip order id = %q", order.ID)
	}
	if len(order.Stages) != 1 {
		t.Fatalf("roundtrip stage count = %d", len(order.Stages))
	}
	if order.Stages[0].Extra == nil {
		t.Fatal("roundtrip stage extra is nil")
	}
	if string(order.Stages[0].Extra["key"]) != `"val"` {
		t.Errorf("roundtrip stage extra[key] = %s", order.Stages[0].Extra["key"])
	}
	if len(roundtrip.ActiveOrderIDs) != 1 || roundtrip.ActiveOrderIDs[0] != "o-1" {
		t.Errorf("roundtrip active_order_ids = %v", roundtrip.ActiveOrderIDs)
	}
}

func TestLoopEventsMultipleDrops(t *testing.T) {
	dir := t.TempDir()
	ndjson := `{"seq":1,"type":"order.dropped","at":"2026-02-24T10:00:00Z","payload":{"order_id":"order-1","reason":"no stages resolve"}}
{"seq":2,"type":"order.dropped","at":"2026-02-24T10:01:00Z","payload":{"order_id":"order-2","reason":"skill removed"}}
`
	if err := os.WriteFile(filepath.Join(dir, "loop-events.ndjson"), []byte(ndjson), 0o644); err != nil {
		t.Fatalf("write loop-events: %v", err)
	}

	events := readLoopEvents(dir)
	if len(events) != 2 {
		t.Fatalf("event count = %d, want 2", len(events))
	}

	if events[0].Category != "order_drop" {
		t.Errorf("event[0] category = %q, want order_drop", events[0].Category)
	}
	if events[0].Body != "Dropped order order-1: no stages resolve" {
		t.Errorf("event[0] body = %q", events[0].Body)
	}
	if events[1].Category != "order_drop" {
		t.Errorf("event[1] category = %q, want order_drop", events[1].Category)
	}
	if events[1].Body != "Dropped order order-2: skill removed" {
		t.Errorf("event[1] body = %q", events[1].Body)
	}
}

func TestSnapshotPendingReviewIncludesReason(t *testing.T) {
	snap := Snapshot{
		PendingReviews: []loop.PendingReviewItem{
			{
				OrderID: "o-1",
				TaskKey: "execute",
				Reason:  "max retries exceeded",
			},
		},
		PendingReviewCount: 1,
	}

	data, err := json.Marshal(snap)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var parsed struct {
		PendingReviews []struct {
			OrderID string `json:"order_id"`
			Reason  string `json:"reason"`
		} `json:"pending_reviews"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(parsed.PendingReviews) != 1 {
		t.Fatalf("pending_reviews count = %d", len(parsed.PendingReviews))
	}
	if parsed.PendingReviews[0].Reason != "max retries exceeded" {
		t.Errorf("reason = %q, want %q", parsed.PendingReviews[0].Reason, "max retries exceeded")
	}
}

func TestLoadSnapshotFromLoopState(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 2, 26, 12, 0, 0, 0, time.UTC)

	state := loop.LoopState{
		Status: "running",
		ActiveCooks: []loop.CookSummary{
			{
				SessionID:   "cook-a",
				OrderID:     "order-1",
				TaskKey:     "execute",
				Runtime:     "claude-code",
				Provider:    "claude",
				Model:       "claude-sonnet-4-6",
				DisplayName: "Cook Alpha",
				Status:      "running",
			},
		},
		RecentHistory: []mise.HistoryItem{
			{
				SessionID:   "cook-b",
				Status:      "completed",
				TaskKey:     "plan",
				CompletedAt: now.Add(-5 * time.Minute),
			},
		},
		Orders: []loop.Order{
			{
				ID:    "order-1",
				Title: "Test order",
				Stages: []loop.Stage{
					{TaskKey: "execute", Status: "active"},
				},
				Status: "active",
			},
		},
		ActiveOrderIDs:     []string{"order-1"},
		ActionNeeded:       []string{"check order-1"},
		TotalCostUSD:       1.23,
		MaxCooks:           4,
		Autonomy:           "auto",
		PendingReviews:     nil,
		PendingReviewCount: 0,
	}

	snap, err := LoadSnapshot(dir, now, state)
	if err != nil {
		t.Fatalf("LoadSnapshot: %v", err)
	}

	if snap.LoopState != "running" {
		t.Errorf("loop_state = %q, want running", snap.LoopState)
	}
	if len(snap.Active) != 1 {
		t.Fatalf("active count = %d, want 1", len(snap.Active))
	}
	if snap.Active[0].ID != "cook-a" {
		t.Errorf("active[0] id = %q", snap.Active[0].ID)
	}
	if snap.Active[0].DisplayName != "Cook Alpha" {
		t.Errorf("active[0] display_name = %q", snap.Active[0].DisplayName)
	}
	if len(snap.Recent) != 1 {
		t.Fatalf("recent count = %d, want 1", len(snap.Recent))
	}
	if snap.Recent[0].ID != "cook-b" {
		t.Errorf("recent[0] id = %q", snap.Recent[0].ID)
	}
	if len(snap.Orders) != 1 {
		t.Fatalf("order count = %d, want 1", len(snap.Orders))
	}
	if snap.Orders[0].Stages[0].SessionID != "cook-a" {
		t.Errorf("stage session_id = %q, want cook-a", snap.Orders[0].Stages[0].SessionID)
	}
	if snap.TotalCostUSD != 1.23 {
		t.Errorf("total_cost = %f, want 1.23", snap.TotalCostUSD)
	}
	if snap.MaxCooks != 4 {
		t.Errorf("max_cooks = %d, want 4", snap.MaxCooks)
	}
	if snap.Autonomy != "auto" {
		t.Errorf("autonomy = %q, want auto", snap.Autonomy)
	}
	if len(snap.ActiveOrderIDs) != 1 || snap.ActiveOrderIDs[0] != "order-1" {
		t.Errorf("active_order_ids = %v", snap.ActiveOrderIDs)
	}
	if len(snap.ActionNeeded) != 1 || snap.ActionNeeded[0] != "check order-1" {
		t.Errorf("action_needed = %v", snap.ActionNeeded)
	}
}

func TestLoadSnapshotNilSlicesMarshalAsEmptyArrays(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 2, 26, 12, 0, 0, 0, time.UTC)

	// All slice fields nil — simulates idle loop with no orders/cooks/reviews.
	state := loop.LoopState{Status: "idle"}

	snap, err := LoadSnapshot(dir, now, state)
	if err != nil {
		t.Fatalf("LoadSnapshot: %v", err)
	}

	data, err := json.Marshal(snap)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	jsonStr := string(data)

	// None of the array fields should be null — they must be [].
	for _, field := range []string{
		"sessions", "active", "recent", "orders",
		"active_order_ids", "action_needed", "feed_events", "pending_reviews",
	} {
		needle := `"` + field + `":null`
		if strings.Contains(jsonStr, needle) {
			t.Errorf("%s is null in JSON, want []", field)
		}
	}
}
