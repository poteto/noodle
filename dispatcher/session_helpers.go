package dispatcher

import (
	"bufio"
	"encoding/json"
	"os"
	"strings"
	"time"

	"github.com/poteto/noodle/event"
	"github.com/poteto/noodle/parse"
)

func nowUTC() time.Time {
	return time.Now().UTC()
}

const sessionHeartbeatTTLSeconds = 30

func readCanonicalEvents(path string) ([]parse.CanonicalEvent, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, scannerInitialBuffer), scannerMaxBuffer)
	events := make([]parse.CanonicalEvent, 0, 32)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var event parse.CanonicalEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}
		events = append(events, event)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return events, nil
}

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
		tool, summary := parseActionMessage(canonical.Message)
		payload := map[string]any{"message": canonical.Message}
		if tool != "" {
			payload["tool"] = tool
		}
		if summary != "" {
			payload["summary"] = summary
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

// parseActionMessage extracts a tool name and summary from a canonical action message.
// Conventions: "$ cmd" → bash, "text:..." → think, "user:..." → prompt, "Tool detail" → tool.
func parseActionMessage(message string) (tool string, summary string) {
	message = strings.TrimSpace(message)
	if message == "" {
		return "", ""
	}
	if strings.HasPrefix(message, "$ ") {
		return "Bash", strings.TrimPrefix(message, "$ ")
	}
	if strings.HasPrefix(message, "text:") {
		return "Think", strings.TrimPrefix(message, "text:")
	}
	if strings.HasPrefix(message, "user:") {
		return "Prompt", strings.TrimPrefix(message, "user:")
	}
	// "ToolName detail" or just "ToolName"
	idx := strings.IndexByte(message, ' ')
	if idx == -1 {
		return message, ""
	}
	return message[:idx], message[idx+1:]
}
