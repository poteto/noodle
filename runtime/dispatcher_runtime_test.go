package runtime

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/poteto/noodle/dispatcher"
)

type fakeSession struct {
	id   string
	done chan struct{}
}

func (s *fakeSession) ID() string                             { return s.id }
func (s *fakeSession) Status() string                         { return "running" }
func (s *fakeSession) Done() <-chan struct{}                  { return s.done }
func (s *fakeSession) TotalCost() float64                     { return 0 }
func (s *fakeSession) Terminate() error                       { return s.closeDone() }
func (s *fakeSession) ForceKill() error                       { return s.closeDone() }
func (s *fakeSession) Events() <-chan dispatcher.SessionEvent { return nil }
func (s *fakeSession) Controller() dispatcher.AgentController { return nil }

func (s *fakeSession) closeDone() error {
	select {
	case <-s.done:
	default:
		close(s.done)
	}
	return nil
}

type fakeDispatcher struct {
	nextID   int
	sessions []*fakeSession
}

func (d *fakeDispatcher) Dispatch(_ context.Context, _ dispatcher.DispatchRequest) (dispatcher.Session, error) {
	d.nextID++
	s := &fakeSession{id: fmt.Sprintf("sess-%d", d.nextID), done: make(chan struct{})}
	d.sessions = append(d.sessions, s)
	return s, nil
}

func TestDispatcherRuntimeConcurrencyLimit(t *testing.T) {
	fd := &fakeDispatcher{}
	rt := NewDispatcherRuntime("test", fd, t.TempDir())
	rt.SetMaxConcurrent(2)

	ctx := context.Background()

	// Dispatch two sessions — should succeed.
	h1, err := rt.Dispatch(ctx, DispatchRequest{})
	if err != nil {
		t.Fatalf("dispatch 1: %v", err)
	}
	h2, err := rt.Dispatch(ctx, DispatchRequest{})
	if err != nil {
		t.Fatalf("dispatch 2: %v", err)
	}

	// Third dispatch should fail with concurrency limit error.
	_, err = rt.Dispatch(ctx, DispatchRequest{})
	if err == nil {
		t.Fatal("expected concurrency limit error")
	}
	if !strings.Contains(err.Error(), "concurrency limit reached") {
		t.Fatalf("unexpected error: %v", err)
	}

	// Terminate first session and wait for watchDone goroutine to decrement.
	_ = h1.ForceKill()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		rt.mu.Lock()
		active := rt.active
		rt.mu.Unlock()
		if active < 2 {
			break
		}
		runtime.Gosched()
	}

	// Now dispatch should succeed again.
	h3, err := rt.Dispatch(ctx, DispatchRequest{})
	if err != nil {
		t.Fatalf("dispatch 3 after kill: %v", err)
	}

	// Cleanup.
	_ = h2.ForceKill()
	_ = h3.ForceKill()
}

func TestDispatcherRuntimeUnlimitedConcurrency(t *testing.T) {
	fd := &fakeDispatcher{}
	rt := NewDispatcherRuntime("test", fd, t.TempDir())
	// maxConcurrent = 0 means unlimited.

	ctx := context.Background()
	handles := make([]SessionHandle, 10)
	for i := range handles {
		var err error
		handles[i], err = rt.Dispatch(ctx, DispatchRequest{})
		if err != nil {
			t.Fatalf("dispatch %d: %v", i, err)
		}
	}

	for _, h := range handles {
		_ = h.ForceKill()
	}
}

func TestDispatcherRuntimeDecrementOnDispatchError(t *testing.T) {
	// A dispatcher that fails on the second call.
	callCount := 0
	fd := &fakeDispatcher{}
	origDispatch := fd
	failingDispatcher := &failOnNthDispatcher{delegate: origDispatch, failOn: 2}

	rt := NewDispatcherRuntime("test", failingDispatcher, t.TempDir())
	rt.SetMaxConcurrent(2)

	ctx := context.Background()

	// First dispatch succeeds.
	_, err := rt.Dispatch(ctx, DispatchRequest{})
	if err != nil {
		t.Fatalf("dispatch 1: %v", err)
	}
	_ = callCount // suppress unused

	// Second dispatch fails — active count should be decremented.
	_, err = rt.Dispatch(ctx, DispatchRequest{})
	if err == nil {
		t.Fatal("expected dispatch error")
	}

	// active should be 1 (not 2).
	rt.mu.Lock()
	active := rt.active
	rt.mu.Unlock()
	if active != 1 {
		t.Fatalf("active = %d, want 1", active)
	}
}

type failOnNthDispatcher struct {
	delegate dispatcher.Dispatcher
	failOn   int
	count    int
}

func (d *failOnNthDispatcher) Dispatch(ctx context.Context, req dispatcher.DispatchRequest) (dispatcher.Session, error) {
	d.count++
	if d.count == d.failOn {
		return nil, fmt.Errorf("injected dispatch failure")
	}
	return d.delegate.Dispatch(ctx, req)
}
