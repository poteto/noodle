package main

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/poteto/noodle/config"
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

func TestRunStartCommandOnceUsesLoopCycle(t *testing.T) {
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
	if err := runStartCommand(context.Background(), app, nil, []string{"--once"}); err != nil {
		t.Fatalf("runStartCommand --once: %v", err)
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
	runStartTUI = func(string) error { return nil }
	defer func() { runStartTUI = originalRunStartTUI }()

	loop := &fakeStartLoop{
		runFn: func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err := runStartWithTUI(ctx, cancel, loop, ".noodle")
	if err != nil {
		t.Fatalf("runStartWithTUI returned error: %v", err)
	}
}

func TestRunStartWithTUIPropagatesLoopError(t *testing.T) {
	originalRunStartTUI := runStartTUI
	runStartTUI = func(string) error { return nil }
	defer func() { runStartTUI = originalRunStartTUI }()

	loop := &fakeStartLoop{runErr: errors.New("loop failed")}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := runStartWithTUI(ctx, cancel, loop, ".noodle")
	if err == nil || err.Error() != "loop failed" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunStartWithTUIPropagatesTUIError(t *testing.T) {
	originalRunStartTUI := runStartTUI
	runStartTUI = func(string) error { return errors.New("tui failed") }
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

	err := runStartWithTUI(ctx, cancel, loop, ".noodle")
	if err == nil || err.Error() != "tui failed" {
		t.Fatalf("unexpected error: %v", err)
	}
}
