package snapshot

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/poteto/noodle/event"
	"github.com/poteto/noodle/internal/stringx"
)

// ReadSessionEvents reads canonical events for a single session on demand.
func ReadSessionEvents(runtimeDir, sessionID string) ([]EventLine, error) {
	reader := event.NewEventReader(runtimeDir)
	events, err := reader.ReadSession(sessionID, event.EventFilter{})
	if err != nil {
		if os.IsNotExist(err) {
			return []EventLine{}, nil
		}
		return nil, err
	}
	return mapEventLines(events), nil
}

func mapEventLines(events []event.Event) []EventLine {
	lines := make([]EventLine, 0, len(events))
	for _, ev := range events {
		if line, ok := FormatSingleEvent(ev); ok {
			lines = append(lines, line)
		}
	}
	return lines
}

// FormatSingleEvent converts a single event.Event into an EventLine for display.
func FormatSingleEvent(ev event.Event) (EventLine, bool) {
	line := EventLine{
		At:       ev.Timestamp,
		Label:    "Event",
		Body:     "",
		Category: TraceFilterAll,
	}
	switch ev.Type {
	case event.EventCost:
		line.Label = "Cost"
		line.Category = TraceFilterAll
		line.Body = formatCost(ev.Payload)
	case event.EventTicketClaim, event.EventTicketProgress, event.EventTicketDone, event.EventTicketBlocked, event.EventTicketRelease:
		line.Label = "Ticket"
		line.Category = TraceFilterTicket
		line.Body = formatTicket(ev.Payload, string(ev.Type))
	case event.EventAction:
		label, body, category := formatAction(ev.Payload)
		line.Label = label
		line.Body = body
		line.Category = category
	case event.EventStateChange:
		line.Label = "State"
		line.Category = TraceFilterAll
		line.Body = formatStateChange(ev.Payload)
	default:
		line.Label = titleCase(strings.ReplaceAll(string(ev.Type), "_", " "))
		line.Body = summarizePayload(ev.Payload)
		line.Category = TraceFilterAll
	}
	if line.Body == "" {
		line.Body = "(no details)"
	}
	return line, true
}

func formatCost(payload json.RawMessage) string {
	var body struct {
		CostUSD   float64 `json:"cost_usd"`
		TokensIn  int     `json:"tokens_in"`
		TokensOut int     `json:"tokens_out"`
	}
	if err := json.Unmarshal(payload, &body); err != nil {
		return summarizePayload(payload)
	}
	if body.TokensIn == 0 && body.TokensOut == 0 {
		return fmt.Sprintf("$%.2f", body.CostUSD)
	}
	return fmt.Sprintf("$%.2f | %s in / %s out", body.CostUSD, shortInt(body.TokensIn), shortInt(body.TokensOut))
}

func formatTicket(payload json.RawMessage, fallback string) string {
	var body struct {
		Target  string `json:"target"`
		Summary string `json:"summary"`
		Outcome string `json:"outcome"`
		Reason  string `json:"reason"`
	}
	if err := json.Unmarshal(payload, &body); err != nil {
		return fallback
	}
	if body.Summary != "" {
		return body.Summary
	}
	if body.Outcome != "" {
		return body.Outcome
	}
	if body.Reason != "" {
		return body.Reason
	}
	if body.Target != "" {
		return body.Target
	}
	return fallback
}

func formatAction(payload json.RawMessage) (label string, body string, category TraceFilter) {
	var action struct {
		Tool    string `json:"tool"`
		Action  string `json:"action"`
		Summary string `json:"summary"`
		Message string `json:"message"`
		Command string `json:"command"`
		Path    string `json:"path"`
	}
	_ = json.Unmarshal(payload, &action)

	tool := stringx.Normalize(stringx.NonEmpty(action.Tool, action.Action))
	switch tool {
	case "user":
		label = "User"
		category = TraceFilterAll
	case "read":
		label = "Read"
		category = TraceFilterTools
	case "edit", "write":
		label = "Edit"
		category = TraceFilterTools
	case "bash", "shell":
		label = "Bash"
		category = TraceFilterTools
	case "glob":
		label = "Glob"
		category = TraceFilterTools
	case "grep":
		label = "Grep"
		category = TraceFilterTools
	case "skill":
		label = "Skill"
		category = TraceFilterTools
	case "task":
		label = "Task"
		category = TraceFilterTools
	case "prompt":
		label = "Prompt"
		category = TraceFilterThink
	case "think":
		label = "Think"
		category = TraceFilterThink
	default:
		label = "Think"
		category = TraceFilterThink
	}

	body = stringx.FirstNonEmpty(
		strings.TrimSpace(action.Summary),
		strings.TrimSpace(action.Message),
		strings.TrimSpace(action.Command),
		strings.TrimSpace(action.Path),
	)
	if body == "" {
		body = summarizePayload(payload)
	}
	return label, body, category
}

func formatStateChange(payload json.RawMessage) string {
	var body struct {
		ToStatus string `json:"to_status"`
		Reason   string `json:"reason"`
	}
	if err := json.Unmarshal(payload, &body); err != nil {
		return summarizePayload(payload)
	}
	status := strings.TrimSpace(body.ToStatus)
	reason := strings.TrimSpace(body.Reason)
	if status != "" && reason != "" {
		return status + ": " + reason
	}
	if reason != "" {
		return reason
	}
	if status != "" {
		return status
	}
	return summarizePayload(payload)
}

func summarizePayload(payload json.RawMessage) string {
	value := strings.TrimSpace(string(payload))
	if value == "" || value == "null" {
		return ""
	}
	return stringx.MiddleTruncate(value, 80)
}

func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func shortInt(v int) string {
	if v >= 1000 {
		return fmt.Sprintf("%.1fk", float64(v)/1000.0)
	}
	return fmt.Sprintf("%d", v)
}
