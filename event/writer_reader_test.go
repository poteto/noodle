package event

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"
)

func TestEventWriterReaderRoundTripAndFilter(t *testing.T) {
	runtimeDir := filepath.Join(t.TempDir(), ".noodle")
	writer, err := NewEventWriter(runtimeDir, "cook-a")
	if err != nil {
		t.Fatalf("new event writer: %v", err)
	}

	actionTime := time.Date(2026, 2, 22, 19, 0, 0, 0, time.UTC)
	costTime := actionTime.Add(2 * time.Minute)
	actionPayload := mustRawJSON(t, `{"tool":"Edit","summary":"touch files"}`)
	costPayload := mustRawJSON(t, `{"cost_usd":1.2,"tokens_in":120}`)

	if err := writer.Append(context.Background(), Event{
		Type:      EventAction,
		Payload:   actionPayload,
		Timestamp: actionTime,
	}); err != nil {
		t.Fatalf("append action: %v", err)
	}
	if err := writer.Append(context.Background(), Event{
		Type:      EventCost,
		Payload:   costPayload,
		Timestamp: costTime,
	}); err != nil {
		t.Fatalf("append cost: %v", err)
	}

	reader := NewEventReader(runtimeDir)
	allEvents, err := reader.ReadSession("cook-a", EventFilter{})
	if err != nil {
		t.Fatalf("read all events: %v", err)
	}
	if len(allEvents) != 2 {
		t.Fatalf("event count = %d, want 2", len(allEvents))
	}
	if allEvents[0].Type != EventAction {
		t.Fatalf("first event type = %q, want %q", allEvents[0].Type, EventAction)
	}
	if allEvents[1].Type != EventCost {
		t.Fatalf("second event type = %q, want %q", allEvents[1].Type, EventCost)
	}

	filter := EventFilter{
		Types: map[EventType]struct{}{
			EventCost: {},
		},
		Since: &costTime,
	}
	filtered, err := reader.ReadSession("cook-a", filter)
	if err != nil {
		t.Fatalf("read filtered events: %v", err)
	}
	if len(filtered) != 1 {
		t.Fatalf("filtered event count = %d, want 1", len(filtered))
	}
	if filtered[0].Type != EventCost {
		t.Fatalf("filtered event type = %q, want %q", filtered[0].Type, EventCost)
	}
}

func TestEventReaderMissingSessionLog(t *testing.T) {
	reader := NewEventReader(filepath.Join(t.TempDir(), ".noodle"))
	events, err := reader.ReadSession("cook-missing", EventFilter{})
	if err != nil {
		t.Fatalf("read missing session log: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("events from missing session = %d, want 0", len(events))
	}
}

func TestEventWriterRejectsWrongSessionID(t *testing.T) {
	writer, err := NewEventWriter(filepath.Join(t.TempDir(), ".noodle"), "cook-a")
	if err != nil {
		t.Fatalf("new event writer: %v", err)
	}
	err = writer.Append(context.Background(), Event{
		Type:      EventAction,
		SessionID: "cook-b",
	})
	if err == nil {
		t.Fatal("expected session mismatch error")
	}
}

func TestEventReaderTailSession(t *testing.T) {
	runtimeDir := filepath.Join(t.TempDir(), ".noodle")
	writer, err := NewEventWriter(runtimeDir, "cook-tail")
	if err != nil {
		t.Fatalf("new event writer: %v", err)
	}

	base := time.Date(2026, 2, 22, 22, 10, 0, 0, time.UTC)
	for i := 0; i < 4; i++ {
		eventType := EventAction
		if i%2 == 1 {
			eventType = EventCost
		}
		if err := writer.Append(context.Background(), Event{
			Type:      eventType,
			Payload:   mustRawJSON(t, `{"index":1}`),
			Timestamp: base.Add(time.Duration(i) * time.Minute),
		}); err != nil {
			t.Fatalf("append event %d: %v", i, err)
		}
	}

	reader := NewEventReader(runtimeDir)
	filter := EventFilter{
		Types: map[EventType]struct{}{
			EventCost: {},
		},
	}
	events, err := reader.TailSession("cook-tail", 1, filter)
	if err != nil {
		t.Fatalf("tail session: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("tail event count = %d, want 1", len(events))
	}
	if events[0].Type != EventCost {
		t.Fatalf("tail event type = %q, want %q", events[0].Type, EventCost)
	}
}

func mustRawJSON(t *testing.T, value string) json.RawMessage {
	t.Helper()
	var raw json.RawMessage
	if err := json.Unmarshal([]byte(value), &raw); err != nil {
		t.Fatalf("parse raw JSON: %v", err)
	}
	return raw
}
