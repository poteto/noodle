package parse

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/poteto/noodle/internal/stringx"
)

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
		if text == "" || isInterruptNotice(text) {
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
