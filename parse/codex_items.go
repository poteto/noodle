package parse

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/poteto/noodle/internal/stringx"
)

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
