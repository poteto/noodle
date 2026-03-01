package parse

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/poteto/noodle/internal/stringx"
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
	Reason    string  `json:"reason"`
	NumTurns  int     `json:"num_turns"`
	Cost      float64 `json:"cost"`
	TokensIn  int     `json:"tokens_in"`
	TokensOut int     `json:"tokens_out"`
}

type codexTurnContext struct {
	TurnID string `json:"turn_id"`
	Model  string `json:"model"`
}

type codexItem struct {
	Type             string `json:"type"`
	Command          string `json:"command"`
	Status           string `json:"status"`
	Text             string `json:"text"`
	Tool             string `json:"tool"`
	AggregatedOutput string `json:"aggregated_output"`
	ExitCode         *int   `json:"exit_code"`
	AgentsStates     map[string]struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	} `json:"agents_states"`
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
	case "turn.started":
		return []CanonicalEvent{{
			Type:      EventAction,
			Message:   "text:turn started",
			Timestamp: ts,
		}}, nil
	case "turn_context":
		return parseCodexTurnContext(envelope.Payload, ts)
	case "response_item":
		return parseCodexResponseItem(envelope.Payload, ts)
	case "event_msg":
		return parseCodexEventMsg(envelope.Payload, ts)
	case "item.completed", "item.started":
		return parseCodexItem(envelope.Type, envelope.Item, ts)
	case "turn.completed":
		return []CanonicalEvent{{
			Type:      EventComplete,
			Message:   "turn completed",
			Timestamp: ts,
		}}, nil
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

func parseCodexTurnContext(payload json.RawMessage, ts time.Time) ([]CanonicalEvent, error) {
	var turn codexTurnContext
	if err := json.Unmarshal(payload, &turn); err != nil {
		return nil, err
	}
	turnID := strings.TrimSpace(turn.TurnID)
	model := strings.TrimSpace(turn.Model)
	if turnID == "" && model == "" {
		return nil, nil
	}
	if turnID != "" && model != "" {
		return []CanonicalEvent{{
			Type:      EventAction,
			Message:   "turn " + turnID + " (" + model + ")",
			Timestamp: ts,
		}}, nil
	}
	return []CanonicalEvent{{
		Type:      EventAction,
		Message:   "turn " + stringx.FirstNonEmpty(turnID, model),
		Timestamp: ts,
	}}, nil
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
		text := extractCodexMessageText(item.Content)
		if text == "" {
			text = strings.TrimSpace(item.Message)
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
		if output == "" {
			return nil, nil
		}
		if _, ok := parseCodexToolOutputKind(output); !ok && !looksLikeErrorText(output) {
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
	case "task_complete", "complete", "session.complete":
		return []CanonicalEvent{{
			Type:      EventComplete,
			Message:   stringx.FirstNonEmpty(event.Message, "task complete"),
			Timestamp: ts,
			CostUSD:   event.Cost,
			TokensIn:  event.TokensIn,
			TokensOut: event.TokensOut,
		}}, nil
	case "error":
		return []CanonicalEvent{{
			Type:      EventError,
			Message:   stringx.FirstNonEmpty(event.Error, event.Message, "codex error"),
			Timestamp: ts,
		}}, nil
	case "agent_message":
		text := strings.TrimSpace(event.Message)
		if text == "" {
			return nil, nil
		}
		return []CanonicalEvent{{
			Type:      EventAction,
			Message:   "text:" + text,
			Timestamp: ts,
		}}, nil
	case "user_message":
		text := strings.TrimSpace(event.Message)
		if text == "" {
			return nil, nil
		}
		return []CanonicalEvent{{
			Type:      EventAction,
			Message:   "user:" + text,
			Timestamp: ts,
		}}, nil
	case "task_started":
		return []CanonicalEvent{{
			Type:      EventAction,
			Message:   "text:task started",
			Timestamp: ts,
		}}, nil
	case "context_compacted":
		return []CanonicalEvent{{
			Type:      EventAction,
			Message:   "text:context compacted",
			Timestamp: ts,
		}}, nil
	case "thread_rolled_back":
		summary := "thread rolled back"
		if event.NumTurns > 0 {
			summary = fmt.Sprintf("thread rolled back (%d turns)", event.NumTurns)
		}
		return []CanonicalEvent{{
			Type:      EventAction,
			Message:   "text:" + summary,
			Timestamp: ts,
		}}, nil
	case "turn_aborted":
		reason := strings.TrimSpace(stringx.FirstNonEmpty(event.Reason, event.Message))
		if reason == "" {
			reason = "aborted"
		}
		return []CanonicalEvent{{
			Type:      EventAction,
			Message:   "text:turn " + reason,
			Timestamp: ts,
		}}, nil
	default:
		return nil, nil
	}
}

func parseCodexItem(envType string, raw json.RawMessage, ts time.Time) ([]CanonicalEvent, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var item codexItem
	if err := json.Unmarshal(raw, &item); err != nil {
		return nil, err
	}

	switch item.Type {
	case "command_execution":
		message := normalizeShellCommand(item.Command)
		if envType == "item.started" && message != "" {
			return []CanonicalEvent{{
				Type:      EventAction,
				Message:   "$ " + message,
				Timestamp: ts,
			}}, nil
		}
		if envType != "item.completed" {
			return nil, nil
		}
		if item.ExitCode == nil || *item.ExitCode == 0 {
			return nil, nil
		}
		if item.ExitCode != nil && *item.ExitCode != 0 {
			if message == "" {
				message = "command execution"
			}
			return []CanonicalEvent{{
				Type:      EventError,
				Message:   "command failed: " + message,
				Timestamp: ts,
			}}, nil
		}
		return nil, nil
	case "agent_message":
		text := strings.TrimSpace(stringx.FirstNonEmpty(item.Text, item.AggregatedOutput))
		if text == "" {
			return nil, nil
		}
		return []CanonicalEvent{{
			Type:      EventAction,
			Message:   "text:" + text,
			Timestamp: ts,
		}}, nil
	case "reasoning":
		return nil, nil
	case "collab_tool_call":
		if envType != "item.completed" {
			return nil, nil
		}
		status := stringx.Normalize(item.Status)
		if status != "failed" && status != "error" && status != "cancelled" {
			return nil, nil
		}
		errText := stringx.FirstNonEmpty(strings.TrimSpace(item.Text), strings.TrimSpace(item.AggregatedOutput))
		if errText == "" {
			for _, state := range item.AgentsStates {
				if msg := strings.TrimSpace(state.Message); msg != "" {
					errText = msg
					break
				}
			}
		}
		if errText == "" {
			tool := strings.TrimSpace(item.Tool)
			if tool == "" {
				tool = "collab tool"
			}
			errText = tool + " " + status
		}
		return []CanonicalEvent{{
			Type:      EventError,
			Message:   errText,
			Timestamp: ts,
		}}, nil
	default:
		return nil, nil
	}
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
			return "$ " + normalizeShellCommand(command)
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

func extractCodexMessageText(parts []struct {
	Type string `json:"type"`
	Text string `json:"text"`
}) string {
	collected := make([]string, 0, len(parts))
	for _, part := range parts {
		text := strings.TrimSpace(part.Text)
		if text == "" {
			continue
		}
		if strings.HasSuffix(part.Type, "_text") || part.Type == "text" {
			collected = append(collected, text)
		}
	}
	return strings.TrimSpace(strings.Join(collected, "\n"))
}

func normalizeShellCommand(command string) string {
	command = strings.TrimSpace(command)
	if command == "" {
		return ""
	}
	prefixes := []string{
		"/bin/zsh -lc ",
		"zsh -lc ",
		"/bin/bash -lc ",
		"bash -lc ",
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(command, prefix) {
			command = strings.TrimSpace(strings.TrimPrefix(command, prefix))
			command = strings.Trim(command, "\"'")
			break
		}
	}
	return command
}

func parseCodexToolOutputKind(output string) (string, bool) {
	lower := stringx.Normalize(output)
	switch {
	case strings.HasPrefix(lower, "exec_command failed:"):
		return "exec_command", true
	case strings.HasPrefix(lower, "write_stdin failed:"):
		return "write_stdin", true
	case strings.HasPrefix(lower, "apply_patch verification failed:"):
		return "apply_patch", true
	case strings.HasPrefix(lower, "collab spawn failed:"):
		return "collab_spawn", true
	default:
		return "", false
	}
}
