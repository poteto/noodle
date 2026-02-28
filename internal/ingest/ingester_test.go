package ingest

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestIngesterAssignsMonotonicIDs(t *testing.T) {
	ingester := NewIngester()
	now := time.Date(2026, 2, 28, 15, 0, 0, 0, time.UTC)

	first, err := ingester.Ingest(InputEnvelope{
		Source:     string(SourceControl),
		RawPayload: json.RawMessage(`{"type":"control_received","command_id":"cmd-1"}`),
		ReceivedAt: now,
	})
	if err != nil {
		t.Fatalf("ingest first event: %v", err)
	}
	second, err := ingester.Ingest(InputEnvelope{
		Source:     string(SourceControl),
		RawPayload: json.RawMessage(`{"type":"mode_changed","command_id":"cmd-2"}`),
		ReceivedAt: now.Add(time.Second),
	})
	if err != nil {
		t.Fatalf("ingest second event: %v", err)
	}

	if first.ID != 1 {
		t.Fatalf("first event ID mismatch: got %d, want 1", first.ID)
	}
	if second.ID != 2 {
		t.Fatalf("second event ID mismatch: got %d, want 2", second.ID)
	}
	if !first.Applied || !second.Applied {
		t.Fatalf("expected both events to be applied: first=%t second=%t", first.Applied, second.Applied)
	}
}

func TestNextIDMonotonic(t *testing.T) {
	ingester := NewIngester()
	if got := ingester.NextID(); got != 1 {
		t.Fatalf("first NextID mismatch: got %d, want 1", got)
	}
	if got := ingester.NextID(); got != 2 {
		t.Fatalf("second NextID mismatch: got %d, want 2", got)
	}
	if got := ingester.NextID(); got != 3 {
		t.Fatalf("third NextID mismatch: got %d, want 3", got)
	}
}

func TestDuplicateDetectionSetsDedupReason(t *testing.T) {
	ingester := NewIngester()

	first, err := ingester.Ingest(InputEnvelope{
		Source:     string(SourceControl),
		RawPayload: json.RawMessage(`{"type":"control_received","command_id":"cmd-dup"}`),
	})
	if err != nil {
		t.Fatalf("ingest first duplicate candidate: %v", err)
	}
	second, err := ingester.Ingest(InputEnvelope{
		Source:     string(SourceControl),
		RawPayload: json.RawMessage(`{"type":"control_received","command_id":"cmd-dup"}`),
	})
	if err != nil {
		t.Fatalf("ingest second duplicate candidate: %v", err)
	}

	if !first.Applied {
		t.Fatal("first event should be applied")
	}
	if second.Applied {
		t.Fatal("duplicate event should not be applied")
	}
	if second.DedupReason == "" {
		t.Fatal("duplicate event should include dedup reason")
	}
	if !strings.Contains(second.DedupReason, "event 1") {
		t.Fatalf("dedup reason should reference first event ID, got %q", second.DedupReason)
	}
}

func TestDifferentKeysProduceDistinctEvents(t *testing.T) {
	ingester := NewIngester()

	first, err := ingester.Ingest(InputEnvelope{
		Source:     string(SourceControl),
		RawPayload: json.RawMessage(`{"type":"control_received","command_id":"cmd-a"}`),
	})
	if err != nil {
		t.Fatalf("ingest first event: %v", err)
	}
	second, err := ingester.Ingest(InputEnvelope{
		Source:     string(SourceControl),
		RawPayload: json.RawMessage(`{"type":"control_received","command_id":"cmd-b"}`),
	})
	if err != nil {
		t.Fatalf("ingest second event: %v", err)
	}

	if first.ID == second.ID {
		t.Fatalf("event IDs should differ for distinct keys: %d vs %d", first.ID, second.ID)
	}
	if !first.Applied || !second.Applied {
		t.Fatalf("both events should be applied: first=%t second=%t", first.Applied, second.Applied)
	}
	if first.DedupReason != "" || second.DedupReason != "" {
		t.Fatalf("dedup reason should be empty for distinct keys: first=%q second=%q", first.DedupReason, second.DedupReason)
	}
}

func TestPerSourceIsolationForSameKey(t *testing.T) {
	ingester := NewIngester()

	control, err := ingester.Ingest(InputEnvelope{
		Source:     string(SourceControl),
		RawPayload: json.RawMessage(`{"type":"control_received","command_id":"shared-key"}`),
	})
	if err != nil {
		t.Fatalf("ingest control event: %v", err)
	}
	internal, err := ingester.Ingest(InputEnvelope{
		Source:     string(SourceInternal),
		RawPayload: json.RawMessage(`{"type":"merge_completed","idempotency_key":"shared-key"}`),
	})
	if err != nil {
		t.Fatalf("ingest internal event: %v", err)
	}

	if !control.Applied || !internal.Applied {
		t.Fatalf("expected both events to apply despite matching key: control=%t internal=%t", control.Applied, internal.Applied)
	}
	if internal.DedupReason != "" {
		t.Fatalf("per-source isolation violated, got dedup reason %q", internal.DedupReason)
	}
}

func TestConcurrentIngestGetsUniqueIDs(t *testing.T) {
	ingester := NewIngester()
	const total = 200

	type result struct {
		id  EventID
		err error
	}

	results := make(chan result, total)
	var wg sync.WaitGroup
	wg.Add(total)
	for i := 0; i < total; i++ {
		i := i
		go func() {
			defer wg.Done()
			event, err := ingester.Ingest(InputEnvelope{
				Source: string(SourceControl),
				RawPayload: json.RawMessage(
					fmt.Sprintf(`{"type":"control_received","command_id":"cmd-%d"}`, i),
				),
			})
			results <- result{id: event.ID, err: err}
		}()
	}

	wg.Wait()
	close(results)

	seen := make(map[EventID]struct{}, total)
	for result := range results {
		if result.err != nil {
			t.Fatalf("concurrent ingest failed: %v", result.err)
		}
		if _, exists := seen[result.id]; exists {
			t.Fatalf("duplicate ID observed during concurrent ingest: %d", result.id)
		}
		seen[result.id] = struct{}{}
	}

	if len(seen) != total {
		t.Fatalf("unique ID count mismatch: got %d, want %d", len(seen), total)
	}

	stats := ingester.Stats()
	if stats.TotalEvents != total || stats.AppliedEvents != total || stats.DedupedEvents != 0 {
		t.Fatalf("stats mismatch after concurrent ingest: %+v", stats)
	}
}

func TestStatsTrackingAccuracy(t *testing.T) {
	ingester := NewIngester()

	events := []InputEnvelope{
		{
			Source:     string(SourceControl),
			RawPayload: json.RawMessage(`{"type":"control_received","command_id":"cmd-1"}`),
		},
		{
			Source:     string(SourceControl),
			RawPayload: json.RawMessage(`{"type":"control_received","command_id":"cmd-1"}`),
		},
		{
			Source:     string(SourceControl),
			RawPayload: json.RawMessage(`{"type":"control_received","command_id":"cmd-2"}`),
		},
	}
	for _, envelope := range events {
		if _, err := ingester.Ingest(envelope); err != nil {
			t.Fatalf("ingest event: %v", err)
		}
	}

	stats := ingester.Stats()
	if stats.TotalEvents != 3 {
		t.Fatalf("total events mismatch: got %d, want 3", stats.TotalEvents)
	}
	if stats.DedupedEvents != 1 {
		t.Fatalf("deduped events mismatch: got %d, want 1", stats.DedupedEvents)
	}
	if stats.AppliedEvents != 2 {
		t.Fatalf("applied events mismatch: got %d, want 2", stats.AppliedEvents)
	}
}

func TestInputEnvelopeNormalization(t *testing.T) {
	ingester := NewIngester()
	at := time.Date(2026, 2, 28, 16, 0, 0, 0, time.UTC)

	event, err := ingester.Ingest(InputEnvelope{
		Source: "  CONTROL  ",
		RawPayload: json.RawMessage(`{
			"event_type": "control_received",
			"command_id": "cmd-normalized"
		}`),
		ReceivedAt: at,
	})
	if err != nil {
		t.Fatalf("ingest normalized envelope: %v", err)
	}

	if event.Source != string(SourceControl) {
		t.Fatalf("source normalization mismatch: got %q, want %q", event.Source, SourceControl)
	}
	if event.Type != string(EventControlReceived) {
		t.Fatalf("event type normalization mismatch: got %q", event.Type)
	}
	if !event.Timestamp.Equal(at) {
		t.Fatalf("timestamp normalization mismatch: got %s, want %s", event.Timestamp, at)
	}
	var payloadMap map[string]any
	if err := json.Unmarshal(event.Payload, &payloadMap); err != nil {
		t.Fatalf("decode normalized payload: %v", err)
	}
	if payloadMap["command_id"] != "cmd-normalized" {
		t.Fatalf("normalized payload missing command_id: %+v", payloadMap)
	}
}

func TestStateEventJSONRoundTripAllEventTypes(t *testing.T) {
	eventTypes := []EventType{
		EventControlReceived,
		EventSchedulePromoted,
		EventDispatchRequested,
		EventDispatchCompleted,
		EventStageCompleted,
		EventStageFailed,
		EventOrderCompleted,
		EventOrderFailed,
		EventModeChanged,
		EventSessionAdopted,
		EventMergeCompleted,
		EventMergeFailed,
	}

	for idx, eventType := range eventTypes {
		event := StateEvent{
			ID:             EventID(idx + 1),
			Source:         string(SourceInternal),
			Type:           string(eventType),
			Timestamp:      time.Date(2026, 2, 28, 17, 0, 0, idx, time.UTC),
			Payload:        json.RawMessage(`{"ok":true}`),
			IdempotencyKey: fmt.Sprintf("key-%d", idx),
			DedupReason:    "",
			Applied:        true,
		}

		data, err := json.Marshal(event)
		if err != nil {
			t.Fatalf("marshal event type %q: %v", eventType, err)
		}
		if !strings.Contains(string(data), `"idempotency_key"`) {
			t.Fatalf("serialized event missing snake_case idempotency key field: %s", data)
		}
		if strings.Contains(string(data), `"idempotencyKey"`) {
			t.Fatalf("serialized event contains camelCase field name: %s", data)
		}

		var decoded StateEvent
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("unmarshal event type %q: %v", eventType, err)
		}
		if !reflect.DeepEqual(event, decoded) {
			t.Fatalf("event round-trip mismatch for %q:\noriginal=%+v\ndecoded=%+v", eventType, event, decoded)
		}
	}
}

func TestEdgeCaseEmptyPayload(t *testing.T) {
	ingester := NewIngester()
	_, err := ingester.Ingest(InputEnvelope{
		Source:     string(SourceControl),
		RawPayload: nil,
	})
	if err == nil {
		t.Fatal("empty payload should fail ingestion")
	}
	if !strings.Contains(err.Error(), "payload empty") {
		t.Fatalf("unexpected empty payload error: %v", err)
	}
}

func TestEdgeCaseMissingIdempotencyKey(t *testing.T) {
	ingester := NewIngester()
	_, err := ingester.Ingest(InputEnvelope{
		Source:     string(SourceControl),
		RawPayload: json.RawMessage(`{"type":"control_received"}`),
	})
	if err == nil {
		t.Fatal("missing idempotency key should fail ingestion")
	}
	if !strings.Contains(err.Error(), "idempotency key unresolved") {
		t.Fatalf("unexpected missing-key error: %v", err)
	}
}

func TestEdgeCaseVeryLongIdempotencyKey(t *testing.T) {
	ingester := NewIngester()
	longKey := strings.Repeat("k", 8192)

	first, err := ingester.Ingest(InputEnvelope{
		Source: string(SourceControl),
		RawPayload: json.RawMessage(
			fmt.Sprintf(`{"type":"control_received","command_id":"%s"}`, longKey),
		),
	})
	if err != nil {
		t.Fatalf("ingest first long-key event: %v", err)
	}
	second, err := ingester.Ingest(InputEnvelope{
		Source: string(SourceControl),
		RawPayload: json.RawMessage(
			fmt.Sprintf(`{"type":"control_received","command_id":"%s"}`, longKey),
		),
	})
	if err != nil {
		t.Fatalf("ingest second long-key event: %v", err)
	}

	if !first.Applied {
		t.Fatal("first long-key event should apply")
	}
	if second.Applied {
		t.Fatal("duplicate long-key event should not apply")
	}
	if second.DedupReason == "" {
		t.Fatal("duplicate long-key event should include dedup reason")
	}
}

func TestAppliedEventIndexMarkAppliedAndDuplicateLookup(t *testing.T) {
	index := NewAppliedEventIndex()

	if _, exists := index.IsDuplicate("k"); exists {
		t.Fatal("key should not exist before mark")
	}

	index.MarkApplied("k", 7)
	if id, exists := index.IsDuplicate("k"); !exists || id != 7 {
		t.Fatalf("unexpected duplicate lookup result after first mark: exists=%t id=%d", exists, id)
	}

	index.MarkApplied("k", 9)
	if id, exists := index.IsDuplicate("k"); !exists || id != 7 {
		t.Fatalf("first-applied ID should be retained: exists=%t id=%d", exists, id)
	}
}
