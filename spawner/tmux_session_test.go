package spawner

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/poteto/noodle/event"
	"github.com/poteto/noodle/parse"
)

func TestTmuxSessionClosesEventsAfterDone(t *testing.T) {
	run := func(
		_ context.Context,
		_ string,
		_ []string,
		name string,
		args ...string,
	) ([]byte, error) {
		if name == "tmux" && len(args) >= 1 && args[0] == "has-session" {
			return nil, errors.New("session missing")
		}
		return nil, nil
	}

	session := newTmuxSession(
		"session-a",
		"noodle-session-a",
		".",
		nil,
		"does-not-exist.ndjson",
		nil,
		nil,
		run,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	session.start(ctx)

	select {
	case <-session.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("session did not signal done")
	}

	select {
	case _, ok := <-session.Events():
		if ok {
			t.Fatal("events channel should be closed after done")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("events channel did not close")
	}
}

func TestTmuxSessionWritesEventLogFromCanonical(t *testing.T) {
	runtimeDir := filepath.Join(t.TempDir(), ".noodle")
	writer, err := event.NewEventWriter(runtimeDir, "session-a")
	if err != nil {
		t.Fatalf("new event writer: %v", err)
	}

	session := newTmuxSession(
		"session-a",
		"noodle-session-a",
		".",
		nil,
		"",
		writer,
		nil,
		nil,
	)

	session.consumeCanonical(parse.CanonicalEvent{
		Type:      parse.EventAction,
		Message:   "apply patch",
		Timestamp: time.Date(2026, 2, 22, 20, 0, 0, 0, time.UTC),
	})
	session.consumeCanonical(parse.CanonicalEvent{
		Type:      parse.EventResult,
		CostUSD:   0.12,
		TokensIn:  100,
		TokensOut: 50,
		Timestamp: time.Date(2026, 2, 22, 20, 0, 1, 0, time.UTC),
	})

	reader := event.NewEventReader(runtimeDir)
	records, err := reader.ReadSession("session-a", event.EventFilter{})
	if err != nil {
		t.Fatalf("read event log: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("event count = %d", len(records))
	}
	if records[0].Type != event.EventAction {
		t.Fatalf("first event type = %q", records[0].Type)
	}
	if records[1].Type != event.EventCost {
		t.Fatalf("second event type = %q", records[1].Type)
	}
}
