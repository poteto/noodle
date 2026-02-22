package stamp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/poteto/noodle/parse"
)

func TestProcessLineInjectsTimestampAndParsesEvent(t *testing.T) {
	processor := NewProcessor()
	processor.Now = func() time.Time {
		return time.Date(2026, 2, 22, 17, 0, 0, 0, time.UTC)
	}

	line := []byte(`{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","name":"Bash","input":{"command":"go test ./..."}}]}}`)
	stamped, events, err := processor.ProcessLine(line)
	if err != nil {
		t.Fatalf("process line: %v", err)
	}

	var payload map[string]json.RawMessage
	if err := json.Unmarshal(stamped, &payload); err != nil {
		t.Fatalf("parse stamped line: %v", err)
	}
	if _, ok := payload["_ts"]; !ok {
		t.Fatalf("expected _ts field in stamped line: %s", string(stamped))
	}

	if len(events) != 1 {
		t.Fatalf("event count mismatch: got %d want 1", len(events))
	}
	if events[0].Provider != "claude" {
		t.Fatalf("provider mismatch: got %q want claude", events[0].Provider)
	}
	if events[0].Type != parse.EventAction {
		t.Fatalf("type mismatch: got %q want %q", events[0].Type, parse.EventAction)
	}
}

func TestProcessWritesStampedAndSidecarEvents(t *testing.T) {
	processor := NewProcessor()
	processor.Now = func() time.Time {
		return time.Date(2026, 2, 22, 17, 10, 0, 0, time.UTC)
	}

	input := strings.NewReader(strings.Join([]string{
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"working"}]}}`,
		`{"type":"event_msg","payload":{"type":"task_complete","message":"done"}}`,
		"",
	}, "\n"))

	var stamped bytes.Buffer
	var sidecar bytes.Buffer
	err := processor.Process(context.Background(), input, &stamped, &sidecar)
	if err != nil {
		t.Fatalf("process stream: %v", err)
	}

	stampedLines := strings.Split(strings.TrimSpace(stamped.String()), "\n")
	if len(stampedLines) != 2 {
		t.Fatalf("stamped line count mismatch: got %d want 2", len(stampedLines))
	}
	for _, line := range stampedLines {
		if !strings.Contains(line, `"_ts"`) {
			t.Fatalf("missing _ts in line: %s", line)
		}
	}

	eventLines := strings.Split(strings.TrimSpace(sidecar.String()), "\n")
	if len(eventLines) == 0 {
		t.Fatal("expected sidecar events")
	}

	var foundComplete bool
	for _, line := range eventLines {
		var event parse.CanonicalEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatalf("parse sidecar event: %v", err)
		}
		if event.Type == parse.EventComplete {
			foundComplete = true
		}
	}
	if !foundComplete {
		t.Fatal("expected complete event in sidecar output")
	}
}

func TestProcessRejectsInvalidJSONLine(t *testing.T) {
	processor := NewProcessor()
	var stamped bytes.Buffer

	err := processor.Process(context.Background(), strings.NewReader("not-json\n"), &stamped, nil)
	if err == nil {
		t.Fatal("expected invalid JSON error")
	}
	if !strings.Contains(err.Error(), "parse JSON object") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProcessFailsWhenLineExceedsScannerLimit(t *testing.T) {
	processor := NewProcessor()
	processor.MaxLineBytes = 64

	oversizedText := strings.Repeat("x", 120)
	input := fmt.Sprintf("{\"type\":\"assistant\",\"message\":\"%s\"}\n", oversizedText)

	var stamped bytes.Buffer
	err := processor.Process(context.Background(), strings.NewReader(input), &stamped, nil)
	if err == nil {
		t.Fatal("expected scanner token too long error")
	}
	if !strings.Contains(err.Error(), "token too long") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProcessDefaultLimitAcceptsLineLargerThanOld16MiBCap(t *testing.T) {
	processor := NewProcessor()
	largeText := strings.Repeat("x", 17*1024*1024)
	input := fmt.Sprintf("{\"type\":\"assistant\",\"message\":\"%s\"}\n", largeText)

	var stamped bytes.Buffer
	err := processor.Process(context.Background(), strings.NewReader(input), &stamped, nil)
	if err != nil {
		t.Fatalf("process large line failed: %v", err)
	}
	if stamped.Len() == 0 {
		t.Fatal("expected stamped output for large line")
	}
}
