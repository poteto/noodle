package loop

import (
	"context"
	"errors"
	"testing"

	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/dispatcher"
	"github.com/poteto/noodle/internal/failure"
	loopruntime "github.com/poteto/noodle/runtime"
)

func TestDispatchSessionFallsBackToProcess(t *testing.T) {
	tmuxRT := newMockRuntime()
	tmuxRT.dispatchErr = errors.New("tmux runtime unavailable")
	processRT := newMockRuntime()

	cfg := config.DefaultConfig()
	cfg.Runtime.Default = "tmux"

	l := New(t.TempDir(), "noodle", cfg, Dependencies{
		Runtimes: map[string]loopruntime.Runtime{
			"tmux":    tmuxRT,
			"process": processRT,
		},
		Worktree: &fakeWorktree{},
		Adapter:  &fakeAdapterRunner{},
		Mise:     &fakeMise{},
		Monitor:  fakeMonitor{},
		Registry: testLoopRegistry(),
	})

	req := loopruntime.DispatchRequest{
		Name:    "test-cook",
		Prompt:  "do something",
		Runtime: "tmux",
	}

	session, fallback, err := l.dispatchSession(context.Background(), req)
	if err != nil {
		t.Fatalf("dispatchSession should fall back to process, got error: %v", err)
	}
	if session == nil {
		t.Fatal("expected non-nil session")
	}
	if fallback.Class != AgentStartFailureClassFallback {
		t.Fatalf("fallback class = %q, want %q", fallback.Class, AgentStartFailureClassFallback)
	}
	if fallback.Recoverability != failure.FailureRecoverabilityRecoverable {
		t.Fatalf("fallback recoverability = %q, want %q", fallback.Recoverability, failure.FailureRecoverabilityRecoverable)
	}
	if fallback.RequestedRuntime != "tmux" {
		t.Fatalf("fallback requested runtime = %q, want tmux", fallback.RequestedRuntime)
	}
	if fallback.SelectedRuntime != "process" {
		t.Fatalf("fallback selected runtime = %q, want process", fallback.SelectedRuntime)
	}

	tmuxRT.mu.Lock()
	if len(tmuxRT.calls) != 1 {
		tmuxRT.mu.Unlock()
		t.Fatalf("expected 1 dispatch call to tmux runtime, got %d", len(tmuxRT.calls))
	}
	tmuxRT.mu.Unlock()

	processRT.mu.Lock()
	defer processRT.mu.Unlock()
	if len(processRT.calls) != 1 {
		t.Fatalf("expected 1 dispatch call to process runtime, got %d", len(processRT.calls))
	}
	if processRT.calls[0].Runtime != "process" {
		t.Fatalf("expected runtime to be rewritten to process, got %q", processRT.calls[0].Runtime)
	}
}

func TestDispatchSessionInvalidRuntimeIsUnrecoverable(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Runtime.Default = "tmux"
	processRT := newMockRuntime()

	l := New(t.TempDir(), "noodle", cfg, Dependencies{
		Runtimes: map[string]loopruntime.Runtime{"process": processRT},
		Worktree: &fakeWorktree{},
		Adapter:  &fakeAdapterRunner{},
		Mise:     &fakeMise{},
		Monitor:  fakeMonitor{},
		Registry: testLoopRegistry(),
	})

	req := loopruntime.DispatchRequest{
		Name:    "test-cook",
		Prompt:  "do something",
		Runtime: "tmux",
	}

	_, _, err := l.dispatchSession(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for invalid runtime configuration")
	}
	envelope, ok := asDispatchFailureEnvelope(err)
	if !ok {
		t.Fatalf("error = %T (%v), want DispatchFailureEnvelope", err, err)
	}
	if envelope.Class != AgentStartFailureClassUnrecoverable {
		t.Fatalf("dispatch class = %q, want %q", envelope.Class, AgentStartFailureClassUnrecoverable)
	}
	if envelope.FailureClass != failure.FailureClassAgentStartUnrecoverable {
		t.Fatalf("failure class = %q, want %q", envelope.FailureClass, failure.FailureClassAgentStartUnrecoverable)
	}
	if envelope.Runtime != "tmux" {
		t.Fatalf("runtime = %q, want tmux", envelope.Runtime)
	}

	processRT.mu.Lock()
	defer processRT.mu.Unlock()
	if len(processRT.calls) != 0 {
		t.Fatalf("process runtime should not be called for invalid runtime, got %d call(s)", len(processRT.calls))
	}
}

func TestDispatchSessionProcessStartFailureIsUnrecoverable(t *testing.T) {
	processRT := newMockRuntime()
	processRT.dispatchErr = dispatcher.ProcessStartError{Cause: errors.New("binary missing")}

	l := New(t.TempDir(), "noodle", config.DefaultConfig(), Dependencies{
		Runtimes: map[string]loopruntime.Runtime{"process": processRT},
		Worktree: &fakeWorktree{},
		Adapter:  &fakeAdapterRunner{},
		Mise:     &fakeMise{},
		Monitor:  fakeMonitor{},
		Registry: testLoopRegistry(),
	})

	req := loopruntime.DispatchRequest{
		Name:    "test-cook",
		Prompt:  "do something",
		Runtime: "process",
	}

	_, _, err := l.dispatchSession(context.Background(), req)
	if err == nil {
		t.Fatal("expected process start failure")
	}
	envelope, ok := asDispatchFailureEnvelope(err)
	if !ok {
		t.Fatalf("error = %T (%v), want DispatchFailureEnvelope", err, err)
	}
	if envelope.Class != AgentStartFailureClassUnrecoverable {
		t.Fatalf("dispatch class = %q, want %q", envelope.Class, AgentStartFailureClassUnrecoverable)
	}
	if !loopruntime.IsProcessStartFailure(err) {
		t.Fatalf("error should carry typed process start failure, got %v", err)
	}
}
