package parse

import (
	"encoding/json"
	"strings"

	"github.com/poteto/noodle/internal/stringx"
)

type ClaudeAdapter struct{}

type claudeLine struct {
	Type      string          `json:"type"`
	SubType   string          `json:"subType"`
	Subtype   string          `json:"subtype"`
	Timestamp json.RawMessage `json:"timestamp"`
	Injected  json.RawMessage `json:"_ts"`
	IsError   json.RawMessage `json:"is_error"`
	IsErrorV2 json.RawMessage `json:"isError"`
	Error     json.RawMessage `json:"error"`
	Result    json.RawMessage `json:"result"`
	Usage     json.RawMessage `json:"usage"`
	Message   json.RawMessage `json:"message"`
	Event     json.RawMessage `json:"event"`
}

type claudeStreamEvent struct {
	Type  string         `json:"type"`
	Index int            `json:"index"`
	Delta claudeSSEDelta `json:"delta"`
}

type claudeSSEDelta struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type claudeMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

type claudeContent struct {
	Type    string          `json:"type"`
	Name    string          `json:"name"`
	Text    string          `json:"text"`
	IsError bool            `json:"is_error"`
	Input   json.RawMessage `json:"input"`
}

func (ClaudeAdapter) Parse(line []byte) ([]CanonicalEvent, error) {
	var payload claudeLine
	if err := json.Unmarshal(line, &payload); err != nil {
		return nil, err
	}

	ts := parseTimestamp(parseTimestampField(payload.Injected), parseTimestampField(payload.Timestamp))
	switch payload.Type {
	case "system":
		subType := stringx.Normalize(payload.SubType)
		if subType == "" {
			subType = stringx.Normalize(payload.Subtype)
		}
		switch subType {
		case "init":
			return []CanonicalEvent{{
				Type:      EventInit,
				Message:   "session started",
				Timestamp: ts,
			}}, nil
		case "api_error":
			message := stringx.FirstNonEmpty(
				extractFlexibleMessage(payload.Error),
				extractFlexibleMessage(payload.Result),
				"provider API error",
			)
			return []CanonicalEvent{{
				Type:      EventError,
				Message:   message,
				Timestamp: ts,
			}}, nil
		default:
			return nil, nil
		}
	case "assistant":
		msg, ok := decodeClaudeMessage(payload.Message)
		if !ok {
			return nil, nil
		}

		contents := decodeClaudeContent(msg.Content)
		events := make([]CanonicalEvent, 0, len(contents))
		for _, content := range contents {
			switch content.Type {
			case "tool_use":
				events = append(events, CanonicalEvent{
					Type:      EventAction,
					Message:   summarizeClaudeToolUse(content.Name, content.Input),
					Timestamp: ts,
				})
			case "text":
				text := strings.TrimSpace(content.Text)
				if text == "" {
					continue
				}
				events = append(events, CanonicalEvent{
					Type:      EventAction,
					Message:   "text:" + text,
					Timestamp: ts,
				})
			}
		}
		return events, nil
	case "result":
		event := CanonicalEvent{
			Type:      EventResult,
			Message:   "turn complete",
			Timestamp: ts,
		}

		event.CostUSD = extractClaudeCost(line)
		event.TokensIn, event.TokensOut = extractClaudeUsage(payload.Usage)

		subType := stringx.Normalize(payload.SubType)
		if subType == "" {
			subType = stringx.Normalize(payload.Subtype)
		}

		isError := parseFlexibleBool(payload.IsError) || parseFlexibleBool(payload.IsErrorV2)
		if subType == "error" || isError {
			event.Type = EventError
			event.Message = stringx.FirstNonEmpty(
				extractFlexibleMessage(payload.Error),
				extractFlexibleMessage(payload.Result),
				"turn failed",
			)
		}

		return []CanonicalEvent{event}, nil
	case "stream_event":
		return parseClaudeStreamEvent(payload.Event, ts)
	case "rate_limit_event":
		// Informational transport event. Route as Claude so it doesn't surface
		// as a canonical routing error, but don't emit canonical activity.
		return nil, nil
	case "user":
		msg, ok := decodeClaudeMessage(payload.Message)
		if !ok {
			return nil, nil
		}

		contents := decodeClaudeContent(msg.Content)
		events := make([]CanonicalEvent, 0, len(contents))
		for _, content := range contents {
			switch content.Type {
			case "text":
				text := strings.TrimSpace(content.Text)
				if text != "" && !isInterruptNotice(text) {
					events = append(events, CanonicalEvent{
						Type:      EventAction,
						Message:   "user:" + text,
						Timestamp: ts,
					})
				}
			case "tool_result":
				text := strings.TrimSpace(content.Text)
				if text == "" {
					continue
				}
				if content.IsError || looksLikeErrorText(text) {
					events = append(events, CanonicalEvent{
						Type:      EventError,
						Message:   strings.TrimSpace(content.Name + ": " + text),
						Timestamp: ts,
					})
				}
			}
		}
		return events, nil
	default:
		return nil, nil
	}
}
