package mise

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/event"
)

func TestBuilderBuildWritesMiseJSON(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, ".noodle"), 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}

	tickets := []event.Ticket{{
		Target:     "42",
		TargetType: event.TicketTargetBacklogItem,
		CookID:     "cook-a",
		Status:     event.TicketStatusActive,
		ClaimedAt:  time.Date(2026, 2, 22, 16, 0, 0, 0, time.UTC),
	}}
	ticketBytes, _ := json.Marshal(tickets)
	if err := os.WriteFile(filepath.Join(projectDir, ".noodle", "tickets.json"), ticketBytes, 0o644); err != nil {
		t.Fatalf("write tickets: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.Adapters = map[string]config.AdapterConfig{
		"backlog": {
			Scripts: map[string]string{
				"sync": "printf '%s\\n' '{\"id\":\"42\",\"title\":\"Fix login\",\"status\":\"open\"}' '{\"id\":\"43\",\"title\":\"Done item\",\"status\":\"done\"}'",
			},
		},
	}

	builder := NewBuilder(projectDir, cfg)
	builder.now = func() time.Time { return time.Date(2026, 2, 22, 16, 1, 0, 0, time.UTC) }

	brief, warnings, _, err := builder.Build(context.Background(), ActiveSummary{
		Total:     1,
		ByTaskKey: map[string]int{"execute": 1},
		ByStatus:  map[string]int{"active": 1},
		ByRuntime: map[string]int{"process": 1},
	}, []HistoryItem{{
		SessionID:   "cook-a",
		OrderID:     "42",
		TaskKey:     "execute",
		Status:      "completed",
		DurationS:   80,
		CompletedAt: time.Date(2026, 2, 22, 16, 0, 0, 0, time.UTC),
	}})
	if err != nil {
		t.Fatalf("build mise: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %#v", warnings)
	}
	if len(brief.Backlog) != 1 {
		t.Fatalf("backlog count = %d", len(brief.Backlog))
	}
	if brief.Backlog[0].ID != "42" {
		t.Fatalf("backlog id = %q", brief.Backlog[0].ID)
	}
	if brief.ActiveSummary.Total != 1 {
		t.Fatalf("active summary total = %d", brief.ActiveSummary.Total)
	}
	if len(brief.Tickets) != 1 {
		t.Fatalf("tickets count = %d", len(brief.Tickets))
	}

	misePath := filepath.Join(projectDir, ".noodle", "mise.json")
	if _, err := os.Stat(misePath); err != nil {
		t.Fatalf("mise.json not written: %v", err)
	}

	// Second build with identical inputs should skip the write.
	firstInfo, _ := os.Stat(misePath)
	firstMod := firstInfo.ModTime()

	// Advance the clock so GeneratedAt differs, but content is the same.
	builder.now = func() time.Time { return time.Date(2026, 2, 22, 16, 2, 0, 0, time.UTC) }
	_, _, _, err = builder.Build(context.Background(), ActiveSummary{
		Total:     1,
		ByTaskKey: map[string]int{"execute": 1},
		ByStatus:  map[string]int{"active": 1},
		ByRuntime: map[string]int{"process": 1},
	}, []HistoryItem{{
		SessionID:   "cook-a",
		OrderID:     "42",
		TaskKey:     "execute",
		Status:      "completed",
		DurationS:   80,
		CompletedAt: time.Date(2026, 2, 22, 16, 0, 0, 0, time.UTC),
	}})
	if err != nil {
		t.Fatalf("second build: %v", err)
	}

	secondInfo, _ := os.Stat(misePath)
	if secondInfo.ModTime() != firstMod {
		t.Fatal("mise.json rewritten despite no content change")
	}
}

func TestBuilderMissingSyncScriptWarnsAndContinues(t *testing.T) {
	projectDir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Adapters = map[string]config.AdapterConfig{
		"backlog": {
			Scripts: map[string]string{
				"sync": ".noodle/adapters/backlog-sync",
			},
		},
	}

	builder := NewBuilder(projectDir, cfg)
	brief, warnings, _, err := builder.Build(context.Background(), ActiveSummary{}, nil)
	if err != nil {
		t.Fatalf("build mise: %v", err)
	}
	if len(brief.Backlog) != 0 {
		t.Fatalf("expected empty backlog, got %d items", len(brief.Backlog))
	}
	if len(warnings) == 0 {
		t.Fatal("expected warning for missing sync script")
	}
	if !strings.Contains(strings.ToLower(strings.Join(warnings, "\n")), "missing") {
		t.Fatalf("unexpected warnings: %#v", warnings)
	}
}

func TestReadRecentEventsWatermark(t *testing.T) {
	dir := t.TempDir()
	eventsPath := filepath.Join(dir, "loop-events.ndjson")

	// Write events: seq 1 (stage.completed), seq 2 (schedule.completed watermark),
	// seq 3 (stage.completed), seq 4 (order.completed).
	lines := []string{
		`{"seq":1,"type":"stage.completed","at":"2026-02-22T16:00:00Z","payload":{"order_id":"a","task_key":"execute"}}`,
		`{"seq":2,"type":"schedule.completed","at":"2026-02-22T16:01:00Z","payload":{"session_id":"s1"}}`,
		`{"seq":3,"type":"stage.completed","at":"2026-02-22T16:02:00Z","payload":{"order_id":"b","task_key":"review"}}`,
		`{"seq":4,"type":"order.completed","at":"2026-02-22T16:03:00Z","payload":{"order_id":"b"}}`,
	}
	if err := os.WriteFile(eventsPath, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("write events: %v", err)
	}

	events := readRecentEvents(dir)

	// Watermark is seq 2 (schedule.completed). Events 3 and 4 should be returned.
	if len(events) != 2 {
		t.Fatalf("got %d events, want 2", len(events))
	}
	if events[0].Seq != 3 {
		t.Fatalf("first event seq = %d, want 3", events[0].Seq)
	}
	if events[0].Type != "stage.completed" {
		t.Fatalf("first event type = %q, want stage.completed", events[0].Type)
	}
	if events[1].Seq != 4 {
		t.Fatalf("second event seq = %d, want 4", events[1].Seq)
	}
}

func TestReadRecentEventsEmptyFile(t *testing.T) {
	dir := t.TempDir()
	eventsPath := filepath.Join(dir, "loop-events.ndjson")
	if err := os.WriteFile(eventsPath, []byte{}, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	events := readRecentEvents(dir)
	if len(events) != 0 {
		t.Fatalf("got %d events, want 0", len(events))
	}
}

func TestReadRecentEventsMissingFile(t *testing.T) {
	events := readRecentEvents(t.TempDir())
	if len(events) != 0 {
		t.Fatalf("got %d events, want 0", len(events))
	}
}

func TestReadRecentEventsMalformedLines(t *testing.T) {
	dir := t.TempDir()
	eventsPath := filepath.Join(dir, "loop-events.ndjson")
	lines := []string{
		`not valid json`,
		`{"seq":1,"type":"stage.completed","at":"2026-02-22T16:00:00Z","payload":{"order_id":"a","task_key":"execute"}}`,
		`{broken`,
		`{"seq":2,"type":"order.completed","at":"2026-02-22T16:01:00Z","payload":{"order_id":"a"}}`,
	}
	if err := os.WriteFile(eventsPath, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	events := readRecentEvents(dir)
	// No watermark, so both valid events are returned. Malformed lines skipped.
	if len(events) != 2 {
		t.Fatalf("got %d events, want 2", len(events))
	}
}

func TestReadRecentEventsCap(t *testing.T) {
	dir := t.TempDir()
	eventsPath := filepath.Join(dir, "loop-events.ndjson")

	var lines []string
	for i := 1; i <= 60; i++ {
		lines = append(lines, fmt.Sprintf(`{"seq":%d,"type":"stage.completed","at":"2026-02-22T16:00:00Z","payload":{"order_id":"o%d","task_key":"execute"}}`, i, i))
	}
	if err := os.WriteFile(eventsPath, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	events := readRecentEvents(dir)
	if len(events) != maxRecentEvents {
		t.Fatalf("got %d events, want %d (capped)", len(events), maxRecentEvents)
	}
	// Should be the most recent 50.
	if events[0].Seq != 11 {
		t.Fatalf("first event seq = %d, want 11", events[0].Seq)
	}
}

func TestReadRecentEventsSummary(t *testing.T) {
	dir := t.TempDir()
	eventsPath := filepath.Join(dir, "loop-events.ndjson")
	lines := []string{
		`{"seq":1,"type":"stage.completed","at":"2026-02-22T16:00:00Z","payload":{"order_id":"42","task_key":"execute"}}`,
		`{"seq":2,"type":"order.failed","at":"2026-02-22T16:01:00Z","payload":{"order_id":"43","reason":"quality rejected"}}`,
	}
	if err := os.WriteFile(eventsPath, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	events := readRecentEvents(dir)
	if len(events) != 2 {
		t.Fatalf("got %d events, want 2", len(events))
	}
	if events[0].Summary != "execute stage completed for 42" {
		t.Fatalf("summary[0] = %q", events[0].Summary)
	}
	if events[1].Summary != "order 43 failed: quality rejected" {
		t.Fatalf("summary[1] = %q", events[1].Summary)
	}
}

func TestReadRecentEventsSummaryIncludesOwnerForAgentMistake(t *testing.T) {
	dir := t.TempDir()
	eventsPath := filepath.Join(dir, "loop-events.ndjson")
	lines := []string{
		`{"seq":1,"type":"promotion.failed","at":"2026-02-22T16:00:00Z","payload":{"reason":"invalid orders-next","failure":{"class":"agent_mistake","recoverability":"recoverable","owner":"scheduler_agent","scope":"system","cycle_class":"degrade-continue"}}}`,
		`{"seq":2,"type":"order.failed","at":"2026-02-22T16:01:00Z","payload":{"order_id":"43","reason":"changes requested","failure":{"class":"agent_mistake","recoverability":"recoverable","owner":"cook_agent","scope":"order","cycle_class":"order-hard","order_class":"stage-terminal"}}}`,
	}
	if err := os.WriteFile(eventsPath, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	events := readRecentEvents(dir)
	if len(events) != 2 {
		t.Fatalf("got %d events, want 2", len(events))
	}
	if events[0].Summary != "[scheduler_agent] orders-next rejected: invalid orders-next" {
		t.Fatalf("summary[0] = %q", events[0].Summary)
	}
	if events[1].Summary != "[cook_agent] order 43 failed: changes requested" {
		t.Fatalf("summary[1] = %q", events[1].Summary)
	}
}

func TestReadRecentEventsSummaryDoesNotPrefixBackendOwner(t *testing.T) {
	dir := t.TempDir()
	eventsPath := filepath.Join(dir, "loop-events.ndjson")
	lines := []string{
		`{"seq":1,"type":"order.failed","at":"2026-02-22T16:00:00Z","payload":{"order_id":"43","reason":"merge failed","failure":{"class":"backend_recoverable","recoverability":"recoverable","owner":"backend","scope":"order","cycle_class":"order-hard","order_class":"stage-terminal"}}}`,
	}
	if err := os.WriteFile(eventsPath, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	events := readRecentEvents(dir)
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	if events[0].Summary != "order 43 failed: merge failed" {
		t.Fatalf("summary[0] = %q", events[0].Summary)
	}
}
