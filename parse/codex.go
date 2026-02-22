package parse

import (
	"encoding/json"
	"strings"
	"time"
)

type CodexAdapter struct{}

type codexEnvelope struct {
	Type      string          `json:"type"`
	Timestamp json.RawMessage `json:"timestamp"`
	Injected  json.RawMessage `json:"_ts"`
	Payload   json.RawMessage `json:"payload"`
	Item      json.RawMessage `json:"item"`
}

type codexResponseItem struct {
	Type      string          `json:"type"`
	Name      string          `json:"name"`
	Arguments string          `json:"arguments"`
	Input     json.RawMessage `json:"input"`
	Role      string          `json:"role"`
	Message   string          `json:"message"`
	Content   []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Output string `json:"output"`
}

type codexEventMessage struct {
	Type      string  `json:"type"`
	Message   string  `json:"message"`
	Error     string  `json:"error"`
	Cost      float64 `json:"cost"`
	TokensIn  int     `json:"tokens_in"`
	TokensOut int     `json:"tokens_out"`
}

type codexItem struct {
	Type     string `json:"type"`
	Command  string `json:"command"`
	Status   string `json:"status"`
	ExitCode *int   `json:"exit_code"`
}

func (CodexAdapter) Parse(line []byte) ([]CanonicalEvent, error) {
	var envelope codexEnvelope
	if err := json.Unmarshal(line, &envelope); err != nil {
		return nil, err
	}

	ts := parseTimestamp(parseTimestampField(envelope.Injected), parseTimestampField(envelope.Timestamp))
	switch envelope.Type {
	case "session_meta", "thread.started":
		return []CanonicalEvent{{
			Type:      EventInit,
			Message:   "codex session started",
			Timestamp: ts,
		}}, nil
	case "response_item":
		return parseCodexResponseItem(envelope.Payload, ts)
	case "event_msg":
		return parseCodexEventMsg(envelope.Payload, ts)
	case "item.completed":
		return parseCodexItemCompleted(envelope.Item, ts)
	case "compacted":
		return []CanonicalEvent{{
			Type:      EventAction,
			Message:   "text:context compacted",
			Timestamp: ts,
		}}, nil
	default:
		return nil, nil
	}
}

func parseCodexResponseItem(payload json.RawMessage, ts time.Time) ([]CanonicalEvent, error) {
	var item codexResponseItem
	if err := json.Unmarshal(payload, &item); err != nil {
		return nil, err
	}

	switch item.Type {
	case "function_call", "custom_tool_call":
		return []CanonicalEvent{{
			Type:      EventAction,
			Message:   summarizeCodexFunctionCall(item.Name, item.Arguments, item.Input),
			Timestamp: ts,
		}}, nil
	case "message":
		if !strings.EqualFold(item.Role, "assistant") {
			return nil, nil
		}
		text := strings.TrimSpace(item.Message)
		if text == "" {
			for _, part := range item.Content {
				if part.Type == "output_text" {
					text = strings.TrimSpace(part.Text)
					if text != "" {
						break
					}
				}
			}
		}
		if text == "" {
			return nil, nil
		}
		return []CanonicalEvent{{
			Type:      EventAction,
			Message:   "text:" + text,
			Timestamp: ts,
		}}, nil
	case "function_call_output", "custom_tool_call_output":
		output := strings.TrimSpace(item.Output)
		if output == "" || !looksLikeErrorText(output) {
			return nil, nil
		}
		return []CanonicalEvent{{
			Type:      EventError,
			Message:   output,
			Timestamp: ts,
		}}, nil
	default:
		return nil, nil
	}
}

func parseCodexEventMsg(payload json.RawMessage, ts time.Time) ([]CanonicalEvent, error) {
	var event codexEventMessage
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, err
	}

	switch event.Type {
	case "task_complete":
		return []CanonicalEvent{{
			Type:      EventComplete,
			Message:   firstNonEmpty(event.Message, "task complete"),
			Timestamp: ts,
			CostUSD:   event.Cost,
			TokensIn:  event.TokensIn,
			TokensOut: event.TokensOut,
		}}, nil
	case "error":
		return []CanonicalEvent{{
			Type:      EventError,
			Message:   firstNonEmpty(event.Error, event.Message, "codex error"),
			Timestamp: ts,
		}}, nil
	default:
		return nil, nil
	}
}

func parseCodexItemCompleted(raw json.RawMessage, ts time.Time) ([]CanonicalEvent, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var item codexItem
	if err := json.Unmarshal(raw, &item); err != nil {
		return nil, err
	}

	if item.Type != "command_execution" {
		return nil, nil
	}

	message := strings.TrimSpace(item.Command)
	if message == "" {
		message = "command execution"
	}

	if item.ExitCode != nil && *item.ExitCode != 0 {
		return []CanonicalEvent{{
			Type:      EventError,
			Message:   "command failed: " + message,
			Timestamp: ts,
		}}, nil
	}

	return []CanonicalEvent{{
		Type:      EventAction,
		Message:   "$ " + message,
		Timestamp: ts,
	}}, nil
}

func summarizeCodexFunctionCall(name, arguments string, input json.RawMessage) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "tool"
	}

	raw := []byte(arguments)
	if len(raw) == 0 {
		raw = input
	}

	var args map[string]json.RawMessage
	if err := json.Unmarshal(raw, &args); err != nil {
		return name
	}

	if commandRaw, ok := args["command"]; ok {
		var command string
		if err := json.Unmarshal(commandRaw, &command); err == nil && strings.TrimSpace(command) != "" {
			return "$ " + command
		}
	}

	if pathRaw, ok := args["file_path"]; ok {
		var filePath string
		if err := json.Unmarshal(pathRaw, &filePath); err == nil && strings.TrimSpace(filePath) != "" {
			return name + " " + filePath
		}
	}

	return name
}
