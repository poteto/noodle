package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/loop"
)

type fakeStartLoop struct {
	cycleErr error
	runErr   error
	runFn    func(context.Context) error
	cycles   int
	runs     int
}

func (f *fakeStartLoop) Cycle(context.Context) error {
	f.cycles++
	return f.cycleErr
}

func (f *fakeStartLoop) Run(ctx context.Context) error {
	f.runs++
	if f.runFn != nil {
		return f.runFn(ctx)
	}
	return f.runErr
}

func (f *fakeStartLoop) Shutdown() {}

func (f *fakeStartLoop) State() loop.LoopState { return loop.LoopState{} }

func TestRunStartOnceUsesLoopCycle(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, ".noodle"), 0o755); err != nil {
		t.Fatalf("mkdir .noodle: %v", err)
	}
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("chdir project dir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(originalDir) })

	fakeLoop := &fakeStartLoop{}
	originalFactory := newStartRuntimeLoop
	newStartRuntimeLoop = func(string, string, config.Config) startRuntimeLoop {
		return fakeLoop
	}
	t.Cleanup(func() { newStartRuntimeLoop = originalFactory })

	app := &App{Config: config.DefaultConfig()}
	if err := runStart(context.Background(), app, startOptions{once: true}); err != nil {
		t.Fatalf("runStart --once: %v", err)
	}
	if fakeLoop.cycles != 1 {
		t.Fatalf("cycle calls = %d, want 1", fakeLoop.cycles)
	}
	if fakeLoop.runs != 0 {
		t.Fatalf("run calls = %d, want 0", fakeLoop.runs)
	}
}

func TestShouldStartServer(t *testing.T) {
	boolPtr := func(v bool) *bool { return &v }

	tests := []struct {
		name        string
		enabled     *bool
		interactive bool
		want        bool
	}{
		{"nil+interactive", nil, true, true},
		{"nil+non-interactive", nil, false, false},
		{"true+non-interactive", boolPtr(true), false, true},
		{"false+interactive", boolPtr(false), true, false},
	}
	// env var override: NOODLE_SERVER=1 forces server on even in non-interactive mode.
	t.Run("nil+non-interactive+NOODLE_SERVER", func(t *testing.T) {
		t.Setenv("NOODLE_SERVER", "1")
		cfg := config.ServerConfig{Enabled: nil}
		if got := shouldStartServer(cfg, false); !got {
			t.Fatal("shouldStartServer should return true when NOODLE_SERVER=1")
		}
	})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.ServerConfig{Enabled: tt.enabled}
			got := shouldStartServer(cfg, tt.interactive)
			if got != tt.want {
				t.Fatalf("shouldStartServer(%v, %v) = %v, want %v", tt.enabled, tt.interactive, got, tt.want)
			}
		})
	}
}
