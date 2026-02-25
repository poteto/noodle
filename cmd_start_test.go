package main

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/poteto/noodle/config"
)

var testServerCfg = config.Config{Server: config.ServerConfig{Port: 0}}

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

func TestRunStartOnceUsesLoopCycle(t *testing.T) {
	projectDir := t.TempDir()
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

func TestRunStartWithTUIStopsLoopOnExit(t *testing.T) {
	originalRunStartTUI := runStartTUI
	runStartTUI = func(context.Context, string) error { return nil }
	defer func() { runStartTUI = originalRunStartTUI }()

	loop := &fakeStartLoop{
		runFn: func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err := runStartWithTUI(ctx, cancel, loop, ".noodle", testServerCfg, false)
	if err != nil {
		t.Fatalf("runStartWithTUI returned error: %v", err)
	}
}

func TestRunStartWithTUIPropagatesLoopError(t *testing.T) {
	originalRunStartTUI := runStartTUI
	runStartTUI = func(context.Context, string) error { return nil }
	defer func() { runStartTUI = originalRunStartTUI }()

	loop := &fakeStartLoop{runErr: errors.New("loop failed")}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := runStartWithTUI(ctx, cancel, loop, ".noodle", testServerCfg, false)
	if err == nil || err.Error() != "loop failed" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunStartWithTUIExitsWhenLoopFailsBeforeTUIExits(t *testing.T) {
	originalRunStartTUI := runStartTUI
	runStartTUI = func(ctx context.Context, _ string) error {
		<-ctx.Done()
		return ctx.Err()
	}
	t.Cleanup(func() { runStartTUI = originalRunStartTUI })

	loop := &fakeStartLoop{runErr: errors.New("loop failed")}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- runStartWithTUI(ctx, cancel, loop, ".noodle", testServerCfg, false)
	}()

	select {
	case err := <-errCh:
		if err == nil || err.Error() != "loop failed" {
			t.Fatalf("unexpected error: %v", err)
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("runStartWithTUI did not exit after loop failure")
	}
}

func TestRunStartWithTUIPropagatesTUIError(t *testing.T) {
	originalRunStartTUI := runStartTUI
	runStartTUI = func(context.Context, string) error { return errors.New("tui failed") }
	defer func() { runStartTUI = originalRunStartTUI }()

	loop := &fakeStartLoop{
		runFn: func(ctx context.Context) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(50 * time.Millisecond):
				return nil
			}
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := runStartWithTUI(ctx, cancel, loop, ".noodle", testServerCfg, false)
	if err == nil || err.Error() != "tui failed" {
		t.Fatalf("unexpected error: %v", err)
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
		{"nil+headless", nil, false, false},
		{"true+headless", boolPtr(true), false, true},
		{"false+interactive", boolPtr(false), true, false},
	}
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

func TestRunStartWithTUIStartsServer(t *testing.T) {
	originalRunStartTUI := runStartTUI
	runStartTUI = func(context.Context, string) error { return nil }
	defer func() { runStartTUI = originalRunStartTUI }()

	originalOpenBrowser := openBrowserFunc
	var browserURL string
	openBrowserFunc = func(url string) { browserURL = url }
	defer func() { openBrowserFunc = originalOpenBrowser }()

	loop := &fakeStartLoop{
		runFn: func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Server enabled with port 0 (auto-assign).
	cfg := config.Config{Server: config.ServerConfig{Port: 0}}
	err := runStartWithTUI(ctx, cancel, loop, t.TempDir(), cfg, true)
	if err != nil {
		t.Fatalf("runStartWithTUI returned error: %v", err)
	}
	if browserURL == "" {
		t.Fatal("openBrowser was not called")
	}
}
