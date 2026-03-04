package dispatcher

import (
	"encoding/json"
	"time"

	"github.com/poteto/noodle/event"
	"github.com/poteto/noodle/parse"
)

func nowUTC() time.Time {
	return time.Now().UTC()
}

const sessionHeartbeatTTLSeconds = 30

func eventFromCanonical(sessionID string, canonical parse.CanonicalEvent) (event.Event, bool) {
	timestamp := canonical.Timestamp
	if timestamp.IsZero() {
		timestamp = nowUTC()
	}

	makePayload := func(values map[string]any) json.RawMessage {
		payload, err := json.Marshal(values)
		if err != nil {
			return nil
		}
		return payload
	}

	switch canonical.Type {
	case parse.EventInit:
		return event.Event{
			Type:      event.EventSpawned,
			Payload:   makePayload(map[string]any{"message": canonical.Message, "provider": canonical.Provider}),
			Timestamp: timestamp,
			SessionID: sessionID,
		}, true
	case parse.EventAction:
		action := parse.ParseActionMessage(canonical.Message)
		payload := map[string]any{"message": canonical.Message}
		if action.Tool != "" {
			payload["tool"] = action.Tool
		}
		if action.Summary != "" {
			payload["summary"] = action.Summary
		}
		return event.Event{
			Type:      event.EventAction,
			Payload:   makePayload(payload),
			Timestamp: timestamp,
			SessionID: sessionID,
		}, true
	case parse.EventResult:
		return event.Event{
			Type: event.EventCost,
			Payload: makePayload(map[string]any{
				"cost_usd":   canonical.CostUSD,
				"tokens_in":  canonical.TokensIn,
				"tokens_out": canonical.TokensOut,
			}),
			Timestamp: timestamp,
			SessionID: sessionID,
		}, true
	case parse.EventError:
		return event.Event{
			Type:      event.EventAction,
			Payload:   makePayload(map[string]any{"message": canonical.Message, "tool": "Error", "summary": canonical.Message}),
			Timestamp: timestamp,
			SessionID: sessionID,
		}, true
	case parse.EventComplete:
		return event.Event{
			Type:      event.EventExited,
			Payload:   makePayload(map[string]any{"message": canonical.Message, "outcome": "completed"}),
			Timestamp: timestamp,
			SessionID: sessionID,
		}, true
	default:
		return event.Event{}, false
	}
}

// FormatEventLine converts a canonical event into an event.Event for sink broadcasting.
func FormatEventLine(sessionID string, ce parse.CanonicalEvent) (event.Event, bool) {
	return eventFromCanonical(sessionID, ce)
}
