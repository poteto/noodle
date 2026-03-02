package parse

import "encoding/json"

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
			Type:      EventResult,
			Message:   "turn complete",
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
