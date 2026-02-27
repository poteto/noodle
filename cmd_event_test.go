package main

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestEventEmit_WritesNDJSON(t *testing.T) {
	runtimeDir := t.TempDir()

	err := runEventEmit(runtimeDir, "ci.failed", `{"repo":"noodle"}`)
	if err != nil {
		t.Fatalf("runEventEmit: %v", err)
	}

	events := readEventFile(t, filepath.Join(runtimeDir, "loop-events.ndjson"))
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	if events[0].Type != "ci.failed" {
		t.Errorf("type = %q, want %q", events[0].Type, "ci.failed")
	}
	if events[0].Seq != 1 {
		t.Errorf("seq = %d, want 1", events[0].Seq)
	}

	var payload map[string]string
	if err := json.Unmarshal(events[0].Payload, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload["repo"] != "noodle" {
		t.Errorf("payload.repo = %q, want %q", payload["repo"], "noodle")
	}
}

func TestEventEmit_NoPayload(t *testing.T) {
	runtimeDir := t.TempDir()

	err := runEventEmit(runtimeDir, "deploy.completed", "")
	if err != nil {
		t.Fatalf("runEventEmit: %v", err)
	}

	events := readEventFile(t, filepath.Join(runtimeDir, "loop-events.ndjson"))
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	if events[0].Type != "deploy.completed" {
		t.Errorf("type = %q, want %q", events[0].Type, "deploy.completed")
	}
	if events[0].Payload != nil {
		t.Errorf("payload = %s, want nil", string(events[0].Payload))
	}
}

func TestEventEmit_MonotonicSequences(t *testing.T) {
	runtimeDir := t.TempDir()

	for i := 0; i < 5; i++ {
		if err := runEventEmit(runtimeDir, "test.event", ""); err != nil {
			t.Fatalf("emit %d: %v", i, err)
		}
	}

	events := readEventFile(t, filepath.Join(runtimeDir, "loop-events.ndjson"))
	if len(events) != 5 {
		t.Fatalf("got %d events, want 5", len(events))
	}
	for i, ev := range events {
		want := uint64(i + 1)
		if ev.Seq != want {
			t.Errorf("event[%d].Seq = %d, want %d", i, ev.Seq, want)
		}
	}
}

func TestEventEmit_InvalidPayload(t *testing.T) {
	runtimeDir := t.TempDir()

	err := runEventEmit(runtimeDir, "ci.failed", "not json")
	if err == nil {
		t.Fatal("expected error for invalid JSON payload")
	}
}

func TestEventEmit_EmptyTypeRejected(t *testing.T) {
	cmd := newEventCmd(&App{})
	emitCmd, _, err := cmd.Find([]string{"emit"})
	if err != nil {
		t.Fatalf("find emit subcommand: %v", err)
	}
	emitCmd.SilenceUsage = true
	emitCmd.SilenceErrors = true
	if err := emitCmd.Args(emitCmd, []string{""}); err == nil {
		t.Error("expected error for empty type argument")
	}
	if err := emitCmd.Args(emitCmd, nil); err == nil {
		t.Error("expected error for missing type argument")
	}
}

type testEvent struct {
	Seq     uint64          `json:"seq"`
	Type    string          `json:"type"`
	At      time.Time       `json:"at"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

func readEventFile(t *testing.T, path string) []testEvent {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()

	var events []testEvent
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var ev testEvent
		if err := json.Unmarshal(scanner.Bytes(), &ev); err != nil {
			t.Fatalf("unmarshal: %v (line: %s)", err, scanner.Text())
		}
		events = append(events, ev)
	}
	return events
}
