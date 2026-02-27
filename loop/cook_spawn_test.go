package loop

import (
	"context"
	"testing"

	"github.com/poteto/noodle/config"
	loopruntime "github.com/poteto/noodle/runtime"
)

func TestDispatchSessionFallsBackToProcess(t *testing.T) {
	processRT := newMockRuntime()

	cfg := config.DefaultConfig()
	cfg.Runtime.Default = "tmux"

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

	session, err := l.dispatchSession(context.Background(), req)
	if err != nil {
		t.Fatalf("dispatchSession should fall back to process, got error: %v", err)
	}
	if session == nil {
		t.Fatal("expected non-nil session")
	}

	processRT.mu.Lock()
	defer processRT.mu.Unlock()
	if len(processRT.calls) != 1 {
		t.Fatalf("expected 1 dispatch call to process runtime, got %d", len(processRT.calls))
	}
	if processRT.calls[0].Runtime != "process" {
		t.Fatalf("expected runtime to be rewritten to process, got %q", processRT.calls[0].Runtime)
	}
}

func TestDispatchSessionErrorsWhenProcessAlsoMissing(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Runtime.Default = "tmux"

	l := New(t.TempDir(), "noodle", cfg, Dependencies{
		Runtimes: map[string]loopruntime.Runtime{},
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

	_, err := l.dispatchSession(context.Background(), req)
	if err == nil {
		t.Fatal("expected error when both tmux and process runtimes are missing")
	}
}
