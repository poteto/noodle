package dispatcher

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/poteto/noodle/event"
	"github.com/poteto/noodle/parse"
)

func TestTmuxSessionClosesEventsAfterDone(t *testing.T) {
	run := func(
		_ context.Context,
		_ string,
		_ []string,
		name string,
		args ...string,
	) ([]byte, error) {
		if name == "tmux" && len(args) >= 1 && args[0] == "has-session" {
			return nil, errors.New("session missing")
		}
		return nil, nil
	}

	session := newTmuxSession(
		"session-a",
		"noodle-session-a",
		".",
		nil,
		"does-not-exist.ndjson",
		"",
		nil,
		nil,
		run,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	session.start(ctx)

	select {
	case <-session.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("session did not signal done")
	}

	select {
	case _, ok := <-session.Events():
		if ok {
			t.Fatal("events channel should be closed after done")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("events channel did not close")
	}
}

func TestTmuxSessionWritesEventLogFromCanonical(t *testing.T) {
	runtimeDir := filepath.Join(t.TempDir(), ".noodle")
	writer, err := event.NewEventWriter(runtimeDir, "session-a")
	if err != nil {
		t.Fatalf("new event writer: %v", err)
	}

	session := newTmuxSession(
		"session-a",
		"noodle-session-a",
		".",
		nil,
		"",
		"",
		writer,
		nil,
		nil,
	)

	session.consumeCanonical(parse.CanonicalEvent{
		Type:      parse.EventAction,
		Message:   "apply patch",
		Timestamp: time.Date(2026, 2, 22, 20, 0, 0, 0, time.UTC),
	})
	session.consumeCanonical(parse.CanonicalEvent{
		Type:      parse.EventResult,
		CostUSD:   0.12,
		TokensIn:  100,
		TokensOut: 50,
		Timestamp: time.Date(2026, 2, 22, 20, 0, 1, 0, time.UTC),
	})

	reader := event.NewEventReader(runtimeDir)
	records, err := reader.ReadSession("session-a", event.EventFilter{})
	if err != nil {
		t.Fatalf("read event log: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("event count = %d", len(records))
	}
	if records[0].Type != event.EventAction {
		t.Fatalf("first event type = %q", records[0].Type)
	}
	if records[1].Type != event.EventCost {
		t.Fatalf("second event type = %q", records[1].Type)
	}
}

func TestTmuxSessionLogsInjectedPromptOnInit(t *testing.T) {
	runtimeDir := filepath.Join(t.TempDir(), ".noodle")
	writer, err := event.NewEventWriter(runtimeDir, "session-a")
	if err != nil {
		t.Fatalf("new event writer: %v", err)
	}
	prompt := "Use Skill(prioritize) to refresh .noodle/queue.json from .noodle/mise.json."

	session := newTmuxSession(
		"session-a",
		"noodle-session-a",
		".",
		nil,
		"",
		prompt,
		writer,
		nil,
		nil,
	)

	ts := time.Date(2026, 2, 23, 0, 0, 0, 0, time.UTC)
	session.consumeCanonical(parse.CanonicalEvent{
		Type:      parse.EventInit,
		Message:   "session started",
		Timestamp: ts,
	})

	reader := event.NewEventReader(runtimeDir)
	records, err := reader.ReadSession("session-a", event.EventFilter{})
	if err != nil {
		t.Fatalf("read event log: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("event count = %d, want 2", len(records))
	}
	if records[0].Type != event.EventSpawned {
		t.Fatalf("first event type = %q", records[0].Type)
	}
	if records[1].Type != event.EventAction {
		t.Fatalf("second event type = %q", records[1].Type)
	}
	var payload struct {
		Tool    string `json:"tool"`
		Action  string `json:"action"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(records[1].Payload, &payload); err != nil {
		t.Fatalf("decode prompt payload: %v", err)
	}
	if payload.Tool != "prompt" {
		t.Fatalf("tool = %q, want prompt", payload.Tool)
	}
	if payload.Action != "prompt_injected" {
		t.Fatalf("action = %q, want prompt_injected", payload.Action)
	}
	if payload.Message != prompt {
		t.Fatalf("message = %q, want %q", payload.Message, prompt)
	}
}

func TestTerminalStatusWithoutCompletionIsFailed(t *testing.T) {
	session := newTmuxSession(
		"session-a",
		"noodle-session-a",
		".",
		nil,
		filepath.Join(t.TempDir(), "canonical.ndjson"),
		"",
		nil,
		nil,
		nil,
	)
	if got := session.terminalStatus(); got != "failed" {
		t.Fatalf("terminal status = %q, want failed", got)
	}
}

func TestTerminalStatusWithCompleteEventIsCompleted(t *testing.T) {
	dir := t.TempDir()
	canonicalPath := filepath.Join(dir, "canonical.ndjson")
	line := `{"type":"complete","message":"done","timestamp":"2026-02-23T01:00:00Z"}`
	if err := os.WriteFile(canonicalPath, []byte(line+"\n"), 0o644); err != nil {
		t.Fatalf("write canonical: %v", err)
	}

	session := newTmuxSession(
		"session-a",
		"noodle-session-a",
		".",
		nil,
		canonicalPath,
		"",
		nil,
		nil,
		nil,
	)
	if got := session.terminalStatus(); got != "completed" {
		t.Fatalf("terminal status = %q, want completed", got)
	}
}

func TestTmuxSessionEmitsDroppedEventSummary(t *testing.T) {
	runtimeDir := filepath.Join(t.TempDir(), ".noodle")
	writer, err := event.NewEventWriter(runtimeDir, "session-a")
	if err != nil {
		t.Fatalf("new event writer: %v", err)
	}
	session := newTmuxSession(
		"session-a",
		"noodle-session-a",
		".",
		nil,
		"",
		"",
		writer,
		nil,
		nil,
	)

	for i := 0; i < cap(session.events); i++ {
		session.events <- SessionEvent{
			Type:      "action",
			Message:   "seed",
			Timestamp: time.Date(2026, 2, 23, 2, 0, i, 0, time.UTC),
		}
	}
	session.publish(SessionEvent{
		Type:      "action",
		Message:   "newest",
		Timestamp: time.Date(2026, 2, 23, 2, 1, 0, 0, time.UTC),
	})

	drained := make([]SessionEvent, 0, cap(session.events)+2)
	done := make(chan struct{})
	go func() {
		for event := range session.Events() {
			drained = append(drained, event)
		}
		close(done)
	}()

	session.markDone("completed")
	go session.closeEventsWhenDone()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("events channel did not close")
	}

	foundWarning := false
	for _, event := range drained {
		if event.Type == "warning" && strings.Contains(event.Message, "dropped") {
			foundWarning = true
			break
		}
	}
	if !foundWarning {
		t.Fatalf("expected dropped-events warning, got %#v", drained)
	}

	reader := event.NewEventReader(runtimeDir)
	records, err := reader.ReadSession("session-a", event.EventFilter{})
	if err != nil {
		t.Fatalf("read event log: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("event log count = %d, want 1", len(records))
	}
	var payload struct {
		Action        string `json:"action"`
		DroppedEvents int64  `json:"dropped_events"`
	}
	if err := json.Unmarshal(records[0].Payload, &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload.Action != "events_dropped" {
		t.Fatalf("action = %q, want events_dropped", payload.Action)
	}
	if payload.DroppedEvents <= 0 {
		t.Fatalf("dropped_events = %d, want > 0", payload.DroppedEvents)
	}
}

func TestTmuxSessionEmitsDroppedSummaryWhenBufferFullAtShutdown(t *testing.T) {
	session := newTmuxSession(
		"session-a",
		"noodle-session-a",
		".",
		nil,
		"",
		"",
		nil,
		nil,
		nil,
	)

	for i := 0; i < cap(session.events); i++ {
		session.events <- SessionEvent{
			Type:      "action",
			Message:   "seed",
			Timestamp: time.Date(2026, 2, 23, 3, 0, i, 0, time.UTC),
		}
	}
	session.publish(SessionEvent{
		Type:      "action",
		Message:   "newest",
		Timestamp: time.Date(2026, 2, 23, 3, 1, 0, 0, time.UTC),
	})

	session.markDone("completed")
	session.closeEventsWhenDone()

	foundWarning := false
	for event := range session.Events() {
		if event.Type == "warning" && strings.Contains(event.Message, "dropped") {
			foundWarning = true
			break
		}
	}
	if !foundWarning {
		t.Fatal("expected dropped-events warning when buffer is full on shutdown")
	}
}
