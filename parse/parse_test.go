package parse

import (
	"encoding/json"
	"testing"
)

func TestCanonicalEventJSONRoundTrip(t *testing.T) {
	original := CanonicalEvent{
		Provider:  "claude",
		Type:      EventResult,
		Message:   "turn complete",
		CostUSD:   1.25,
		TokensIn:  120,
		TokensOut: 48,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal canonical event: %v", err)
	}

	var decoded CanonicalEvent
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal canonical event: %v", err)
	}

	if decoded.Provider != original.Provider {
		t.Fatalf("provider mismatch: got %q want %q", decoded.Provider, original.Provider)
	}
	if decoded.Type != original.Type {
		t.Fatalf("type mismatch: got %q want %q", decoded.Type, original.Type)
	}
	if decoded.Message != original.Message {
		t.Fatalf("message mismatch: got %q want %q", decoded.Message, original.Message)
	}
	if decoded.CostUSD != original.CostUSD {
		t.Fatalf("cost mismatch: got %f want %f", decoded.CostUSD, original.CostUSD)
	}
	if decoded.TokensIn != original.TokensIn {
		t.Fatalf("tokens in mismatch: got %d want %d", decoded.TokensIn, original.TokensIn)
	}
	if decoded.TokensOut != original.TokensOut {
		t.Fatalf("tokens out mismatch: got %d want %d", decoded.TokensOut, original.TokensOut)
	}
}

func TestDetectProvider(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected string
	}{
		{
			name:     "claude_default",
			line:     `{"type":"assistant"}`,
			expected: "claude",
		},
		{
			name:     "codex_response_item",
			line:     `{"type":"response_item","payload":{}}`,
			expected: "codex",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := DetectProvider([]byte(test.line))
			if err != nil {
				t.Fatalf("detect provider: %v", err)
			}
			if got != test.expected {
				t.Fatalf("provider mismatch: got %q want %q", got, test.expected)
			}
		})
	}
}

func TestClaudeAdapterParsesToolUseAndResult(t *testing.T) {
	adapter := ClaudeAdapter{}

	actionLine := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","name":"Bash","input":{"command":"go test ./..."}}]},"_ts":"2026-02-22T16:40:00Z"}`
	events, err := adapter.Parse([]byte(actionLine))
	if err != nil {
		t.Fatalf("parse assistant line: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("event count: got %d want 1", len(events))
	}
	if events[0].Type != EventAction {
		t.Fatalf("event type: got %q want %q", events[0].Type, EventAction)
	}
	if events[0].Message != "$ go test ./..." {
		t.Fatalf("event message: got %q", events[0].Message)
	}

	resultLine := `{"type":"result","total_cost_usd":2.5,"usage":{"input_tokens":100,"output_tokens":60},"_ts":"2026-02-22T16:41:00Z"}`
	events, err = adapter.Parse([]byte(resultLine))
	if err != nil {
		t.Fatalf("parse result line: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("event count: got %d want 1", len(events))
	}
	if events[0].Type != EventResult {
		t.Fatalf("event type: got %q want %q", events[0].Type, EventResult)
	}
	if events[0].CostUSD != 2.5 {
		t.Fatalf("cost mismatch: got %f want 2.5", events[0].CostUSD)
	}
	if events[0].TokensIn != 100 || events[0].TokensOut != 60 {
		t.Fatalf("token mismatch: got in=%d out=%d", events[0].TokensIn, events[0].TokensOut)
	}
}

func TestCodexAdapterParsesFunctionCallAndComplete(t *testing.T) {
	adapter := CodexAdapter{}

	callLine := `{"type":"response_item","timestamp":"2026-02-22T16:42:00Z","payload":{"type":"function_call","name":"shell","arguments":"{\"command\":\"npm test\"}"}}`
	events, err := adapter.Parse([]byte(callLine))
	if err != nil {
		t.Fatalf("parse response_item line: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("event count: got %d want 1", len(events))
	}
	if events[0].Type != EventAction {
		t.Fatalf("event type: got %q want %q", events[0].Type, EventAction)
	}
	if events[0].Message != "$ npm test" {
		t.Fatalf("event message: got %q want %q", events[0].Message, "$ npm test")
	}

	completeLine := `{"type":"event_msg","timestamp":"2026-02-22T16:43:00Z","payload":{"type":"task_complete","message":"done","cost":1.1,"tokens_in":200,"tokens_out":40}}`
	events, err = adapter.Parse([]byte(completeLine))
	if err != nil {
		t.Fatalf("parse event_msg line: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("event count: got %d want 1", len(events))
	}
	if events[0].Type != EventComplete {
		t.Fatalf("event type: got %q want %q", events[0].Type, EventComplete)
	}
	if events[0].CostUSD != 1.1 {
		t.Fatalf("cost mismatch: got %f want 1.1", events[0].CostUSD)
	}
	if events[0].TokensIn != 200 || events[0].TokensOut != 40 {
		t.Fatalf("token mismatch: got in=%d out=%d", events[0].TokensIn, events[0].TokensOut)
	}
}

func TestCodexAdapterParsesAgentMessagesFromEventAndItem(t *testing.T) {
	adapter := CodexAdapter{}

	eventLine := `{"type":"event_msg","timestamp":"2026-02-22T16:44:00Z","payload":{"type":"agent_message","message":"Investigating parser now."}}`
	events, err := adapter.Parse([]byte(eventLine))
	if err != nil {
		t.Fatalf("parse event_msg agent_message line: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("event count: got %d want 1", len(events))
	}
	if events[0].Type != EventAction {
		t.Fatalf("event type: got %q want %q", events[0].Type, EventAction)
	}
	if events[0].Message != "text:Investigating parser now." {
		t.Fatalf("event message: got %q want %q", events[0].Message, "text:Investigating parser now.")
	}

	itemLine := `{"type":"item.completed","_ts":"2026-02-22T16:44:01Z","item":{"type":"agent_message","text":"Parser fallback works."}}`
	events, err = adapter.Parse([]byte(itemLine))
	if err != nil {
		t.Fatalf("parse item.completed agent_message line: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("event count: got %d want 1", len(events))
	}
	if events[0].Type != EventAction {
		t.Fatalf("event type: got %q want %q", events[0].Type, EventAction)
	}
	if events[0].Message != "text:Parser fallback works." {
		t.Fatalf("event message: got %q want %q", events[0].Message, "text:Parser fallback works.")
	}
}

func TestCodexAdapterParsesItemStartedCommandExecution(t *testing.T) {
	adapter := CodexAdapter{}

	line := `{"type":"item.started","timestamp":"2026-02-22T16:45:00Z","item":{"type":"command_execution","command":"/bin/zsh -lc \"go test ./...\"","status":"in_progress"}}`
	events, err := adapter.Parse([]byte(line))
	if err != nil {
		t.Fatalf("parse item.started command_execution line: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("event count: got %d want 1", len(events))
	}
	if events[0].Type != EventAction {
		t.Fatalf("event type: got %q want %q", events[0].Type, EventAction)
	}
	if events[0].Message != "$ go test ./..." {
		t.Fatalf("event message: got %q want %q", events[0].Message, "$ go test ./...")
	}
}
