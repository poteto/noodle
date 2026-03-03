package parse

import "strings"

// ActionSummary is the structured interpretation of a canonical action message.
type ActionSummary struct {
	Tool    string
	Summary string
}

// ParseActionMessage extracts a tool/summary pair from a canonical action message.
// Conventions:
//   - "$ cmd" => Bash
//   - "text:..." => Think
//   - "user:..." => Prompt
//   - "Tool detail" => Tool + detail
func ParseActionMessage(message string) ActionSummary {
	message = strings.TrimSpace(message)
	if message == "" {
		return ActionSummary{}
	}
	if strings.HasPrefix(message, "$ ") {
		return ActionSummary{
			Tool:    "Bash",
			Summary: strings.TrimPrefix(message, "$ "),
		}
	}
	if strings.HasPrefix(message, "text:") {
		return ActionSummary{
			Tool:    "Think",
			Summary: strings.TrimPrefix(message, "text:"),
		}
	}
	if strings.HasPrefix(message, "user:") {
		return ActionSummary{
			Tool:    "Prompt",
			Summary: strings.TrimPrefix(message, "user:"),
		}
	}
	idx := strings.IndexByte(message, ' ')
	if idx == -1 {
		return ActionSummary{Tool: message}
	}
	return ActionSummary{
		Tool:    message[:idx],
		Summary: message[idx+1:],
	}
}
