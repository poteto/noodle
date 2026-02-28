package parse

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

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
		subType := strings.ToLower(strings.TrimSpace(payload.SubType))
		if subType == "" {
			subType = strings.ToLower(strings.TrimSpace(payload.Subtype))
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

		subType := strings.ToLower(strings.TrimSpace(payload.SubType))
		if subType == "" {
			subType = strings.ToLower(strings.TrimSpace(payload.Subtype))
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
				if text != "" {
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

func decodeClaudeMessage(raw json.RawMessage) (claudeMessage, bool) {
	if len(raw) == 0 {
		return claudeMessage{}, false
	}

	var message claudeMessage
	if err := json.Unmarshal(raw, &message); err != nil {
		return claudeMessage{}, false
	}
	return message, true
}

func decodeClaudeContent(raw json.RawMessage) []claudeContent {
	if len(raw) == 0 {
		return nil
	}

	var list []claudeContent
	if err := json.Unmarshal(raw, &list); err == nil {
		return list
	}

	// Some lines use a single object instead of an array.
	var one claudeContent
	if err := json.Unmarshal(raw, &one); err == nil && one.Type != "" {
		return []claudeContent{one}
	}

	// Some lines use plain text content; treat it as assistant text.
	var text string
	if err := json.Unmarshal(raw, &text); err == nil && strings.TrimSpace(text) != "" {
		return []claudeContent{{
			Type: "text",
			Text: text,
		}}
	}

	return nil
}

func summarizeClaudeToolUse(name string, input json.RawMessage) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "tool"
	}

	if command := extractStringField(input, "command"); command != "" {
		return "$ " + command
	}
	if filePath := extractStringField(input, "file_path"); filePath != "" {
		return name + " " + filePath
	}
	if pattern := extractStringField(input, "pattern"); pattern != "" {
		return name + " " + pattern
	}
	if skill := extractStringField(input, "skill"); skill != "" {
		return name + " " + skill
	}
	return name
}

func extractClaudeCost(line []byte) float64 {
	var snake struct {
		TotalCostUSD float64 `json:"total_cost_usd"`
	}
	if err := json.Unmarshal(line, &snake); err == nil && snake.TotalCostUSD > 0 {
		return snake.TotalCostUSD
	}

	var camel struct {
		TotalCostUSD float64 `json:"totalCostUsd"`
	}
	if err := json.Unmarshal(line, &camel); err == nil && camel.TotalCostUSD > 0 {
		return camel.TotalCostUSD
	}

	return 0
}

func extractClaudeUsage(raw json.RawMessage) (int, int) {
	if len(raw) == 0 {
		return 0, 0
	}

	var usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	}
	if err := json.Unmarshal(raw, &usage); err != nil {
		return 0, 0
	}
	return usage.InputTokens, usage.OutputTokens
}

func parseFlexibleBool(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return false
	}

	var asBool bool
	if err := json.Unmarshal(raw, &asBool); err == nil {
		return asBool
	}

	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		asString = strings.ToLower(strings.TrimSpace(asString))
		return asString == "true" || asString == "1" || asString == "yes"
	}

	return false
}

func extractFlexibleMessage(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		return strings.TrimSpace(asString)
	}

	var asObject map[string]json.RawMessage
	if err := json.Unmarshal(raw, &asObject); err != nil {
		return ""
	}

	for _, field := range []string{"message", "error", "type"} {
		if value, ok := asObject[field]; ok {
			text := extractFlexibleMessage(value)
			if text != "" {
				return text
			}
		}
	}

	return strings.TrimSpace(fmt.Sprint(asObject))
}

func looksLikeErrorText(text string) bool {
	lower := strings.ToLower(text)
	return strings.Contains(lower, "error") ||
		strings.Contains(lower, "failed") ||
		strings.Contains(lower, "exit code")
}

func parseClaudeStreamEvent(raw json.RawMessage, ts time.Time) ([]CanonicalEvent, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var sse claudeStreamEvent
	if err := json.Unmarshal(raw, &sse); err != nil {
		return nil, nil
	}
	if sse.Type != "content_block_delta" || sse.Delta.Type != "text_delta" {
		return nil, nil
	}
	if sse.Delta.Text == "" {
		return nil, nil
	}
	return []CanonicalEvent{{
		Type:      EventDelta,
		Message:   sse.Delta.Text,
		Timestamp: ts,
	}}, nil
}
