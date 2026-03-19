package loop

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
)

func TestShouldTriggerRuntimeCycle(t *testing.T) {
	t.Parallel()

	cases := []struct {
		path string
		want bool
	}{
		{path: "/tmp/orders.json", want: true},
		{path: "/tmp/orders-next.json", want: true},
		{path: "/tmp/control.ndjson", want: true},
		{path: "/tmp/status.json", want: false},
		{path: "/tmp/sessions/abc/events.ndjson", want: false},
	}

	for _, tc := range cases {
		if got := shouldTriggerRuntimeCycle(tc.path); got != tc.want {
			t.Fatalf("shouldTriggerRuntimeCycle(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

func TestForwardRuntimeWatcherCoalescesRelevantEvents(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	events := make(chan fsnotify.Event, 4)
	watchErrors := make(chan error, 1)
	cycleTrigger := make(chan struct{}, 1)
	forwardedErrors := make(chan error, 1)

	l := &Loop{}
	go l.forwardRuntimeWatcher(ctx, events, watchErrors, cycleTrigger, forwardedErrors)

	events <- fsnotify.Event{Name: "/tmp/status.json"}
	select {
	case <-cycleTrigger:
		t.Fatal("unexpected cycle trigger for status.json")
	case <-time.After(50 * time.Millisecond):
	}

	events <- fsnotify.Event{Name: "/tmp/orders.json"}
	waitForTriggerBuffer(t, cycleTrigger, 1)

	events <- fsnotify.Event{Name: "/tmp/control.ndjson"}
	time.Sleep(20 * time.Millisecond)
	if got := len(cycleTrigger); got != 1 {
		t.Fatalf("cycle trigger buffer len = %d, want 1", got)
	}

	select {
	case err := <-forwardedErrors:
		t.Fatalf("unexpected forwarded watcher error: %v", err)
	default:
	}
}

func TestForwardRuntimeWatcherDoesNotBlockWhenCycleTriggerFull(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	events := make(chan fsnotify.Event)
	watchErrors := make(chan error, 1)
	cycleTrigger := make(chan struct{}, 1)
	forwardedErrors := make(chan error, 1)

	l := &Loop{}
	go l.forwardRuntimeWatcher(ctx, events, watchErrors, cycleTrigger, forwardedErrors)

	cycleTrigger <- struct{}{}

	const burst = 500
	sent := make(chan struct{})
	go func() {
		defer close(sent)
		for i := 0; i < burst; i++ {
			events <- fsnotify.Event{Name: "/tmp/orders-next.json"}
		}
	}()

	select {
	case <-sent:
	case <-time.After(2 * time.Second):
		t.Fatal("watcher event burst blocked while cycle trigger buffer was full")
	}

	if got := len(cycleTrigger); got != 1 {
		t.Fatalf("cycle trigger buffer len = %d, want 1", got)
	}

	select {
	case err := <-forwardedErrors:
		t.Fatalf("unexpected forwarded watcher error: %v", err)
	default:
	}
}

func TestForwardRuntimeWatcherForwardsErrors(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	events := make(chan fsnotify.Event)
	watchErrors := make(chan error, 1)
	cycleTrigger := make(chan struct{}, 1)
	forwardedErrors := make(chan error, 1)

	l := &Loop{}
	go l.forwardRuntimeWatcher(ctx, events, watchErrors, cycleTrigger, forwardedErrors)

	watchErrors <- errors.New("queue or buffer overflow")

	select {
	case err := <-forwardedErrors:
		if err == nil {
			t.Fatal("expected forwarded watcher error")
		}
		if !strings.Contains(err.Error(), "watch runtime directory: queue or buffer overflow") {
			t.Fatalf("forwarded watcher error = %q", err)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for forwarded watcher error")
	}
}

func waitForTriggerBuffer(t *testing.T, ch chan struct{}, want int) {
	t.Helper()

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if len(ch) == want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("trigger buffer len = %d, want %d", len(ch), want)
}
