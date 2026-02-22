package spawner

import (
	"context"
	"errors"
	"testing"
	"time"
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
