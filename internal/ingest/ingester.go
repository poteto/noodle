package ingest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/poteto/noodle/internal/stringx"
)

// Ingester is the single ingestion arbiter for external inputs.
type Ingester struct {
	nextID atomic.Uint64

	mu      sync.Mutex
	indexes map[string]*AppliedEventIndex
	stats   IngestStats
}

// NewIngester creates a new ingestion arbiter.
func NewIngester() *Ingester {
	return &Ingester{
		indexes: make(map[string]*AppliedEventIndex),
	}
}

// NextID returns the next monotonic event ID.
func (i *Ingester) NextID() EventID {
	return EventID(i.nextID.Add(1))
}

// Ingest normalizes an input envelope, checks dedup, assigns event ID, and
// returns a canonical event envelope.
func (i *Ingester) Ingest(envelope InputEnvelope) (StateEvent, error) {
	normalized, err := normalizeEnvelope(envelope)
	if err != nil {
		return StateEvent{}, err
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	id := i.NextID()
	event := StateEvent{
		ID:             id,
		Source:         normalized.source,
		Type:           normalized.eventType,
		Timestamp:      normalized.timestamp,
		Payload:        normalized.payload,
		IdempotencyKey: normalized.idempotencyKey,
	}

	i.stats.TotalEvents++
	sourceIndex := i.appliedIndex(normalized.source)
	if existingID, duplicate := sourceIndex.IsDuplicate(normalized.idempotencyKey); duplicate {
		event.DedupReason = fmt.Sprintf(
			"duplicate key already applied by event %d",
			existingID,
		)
		i.stats.DedupedEvents++
		return event, nil
	}

	sourceIndex.MarkApplied(normalized.idempotencyKey, id)
	event.Applied = true
	i.stats.AppliedEvents++
	return event, nil
}

// Stats returns ingestion counters.
func (i *Ingester) Stats() IngestStats {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.stats
}

func (i *Ingester) appliedIndex(source string) *AppliedEventIndex {
	index, exists := i.indexes[source]
	if !exists {
		index = NewAppliedEventIndex()
		i.indexes[source] = index
	}
	return index
}

type normalizedEnvelope struct {
	source         string
	eventType      string
	timestamp      time.Time
	payload        json.RawMessage
	idempotencyKey string
}

func normalizeEnvelope(envelope InputEnvelope) (normalizedEnvelope, error) {
	source := stringx.Normalize(envelope.Source)
	if source == "" {
		return normalizedEnvelope{}, fmt.Errorf("ingestion source unavailable")
	}
	if !supportedSource(source) {
		return normalizedEnvelope{}, fmt.Errorf("ingestion source not recognized: %s", source)
	}

	payload, fields, err := compactAndDecodePayload(envelope.RawPayload)
	if err != nil {
		return normalizedEnvelope{}, err
	}

	eventType, err := extractEventType(fields)
	if err != nil {
		return normalizedEnvelope{}, err
	}

	idempotencyKey, err := extractIdempotencyKey(source, fields)
	if err != nil {
		return normalizedEnvelope{}, err
	}

	ts := envelope.ReceivedAt
	if ts.IsZero() {
		ts = time.Now().UTC()
	}

	return normalizedEnvelope{
		source:         source,
		eventType:      eventType,
		timestamp:      ts,
		payload:        payload,
		idempotencyKey: idempotencyKey,
	}, nil
}

func supportedSource(source string) bool {
	switch EventSource(source) {
	case SourceControl, SourceScheduler, SourceRuntime, SourceInternal:
		return true
	default:
		return false
	}
}

func compactAndDecodePayload(raw json.RawMessage) (json.RawMessage, map[string]json.RawMessage, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil, nil, fmt.Errorf("ingestion payload empty")
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &fields); err != nil {
		return nil, nil, fmt.Errorf("ingestion payload unreadable: %w", err)
	}

	var compacted bytes.Buffer
	if err := json.Compact(&compacted, trimmed); err != nil {
		return nil, nil, fmt.Errorf("ingestion payload unreadable: %w", err)
	}

	return json.RawMessage(append([]byte(nil), compacted.Bytes()...)), fields, nil
}

func extractEventType(fields map[string]json.RawMessage) (string, error) {
	if eventType, ok := readStringField(fields, "type"); ok {
		if IsKnownEventType(EventType(eventType)) {
			return eventType, nil
		}
		return "", fmt.Errorf("event type not recognized: %s", eventType)
	}
	if eventType, ok := readStringField(fields, "event_type"); ok {
		if IsKnownEventType(EventType(eventType)) {
			return eventType, nil
		}
		return "", fmt.Errorf("event type not recognized: %s", eventType)
	}
	return "", fmt.Errorf("event type unavailable in payload")
}

func extractIdempotencyKey(source string, fields map[string]json.RawMessage) (string, error) {
	switch EventSource(source) {
	case SourceControl:
		commandID, ok := readStringField(fields, "command_id")
		if !ok {
			return "", fmt.Errorf("idempotency key unresolved for source %s", source)
		}
		return commandID, nil
	case SourceScheduler:
		generationID, ok := readStringField(fields, "scheduler_generation_id")
		if !ok {
			return "", fmt.Errorf("idempotency key unresolved for source %s", source)
		}
		orderID, ok := readStringField(fields, "order_id")
		if !ok {
			return "", fmt.Errorf("idempotency key unresolved for source %s", source)
		}
		return generationID + ":" + orderID, nil
	case SourceRuntime:
		attemptID, ok := readStringField(fields, "attempt_id")
		if !ok {
			return "", fmt.Errorf("idempotency key unresolved for source %s", source)
		}
		terminalStatus, ok := readStringField(fields, "terminal_status")
		if !ok {
			return "", fmt.Errorf("idempotency key unresolved for source %s", source)
		}
		return attemptID + ":" + terminalStatus, nil
	case SourceInternal:
		idempotencyKey, ok := readStringField(fields, "idempotency_key")
		if !ok {
			return "", fmt.Errorf("idempotency key unresolved for source %s", source)
		}
		return idempotencyKey, nil
	default:
		return "", fmt.Errorf("idempotency key unresolved for source %s", source)
	}
}

func readStringField(fields map[string]json.RawMessage, key string) (string, bool) {
	raw, ok := fields[key]
	if !ok {
		return "", false
	}

	var stringValue string
	if err := json.Unmarshal(raw, &stringValue); err == nil {
		stringValue = strings.TrimSpace(stringValue)
		if stringValue == "" {
			return "", false
		}
		return stringValue, true
	}

	var numberValue int64
	if err := json.Unmarshal(raw, &numberValue); err == nil {
		return strconv.FormatInt(numberValue, 10), true
	}

	return "", false
}
