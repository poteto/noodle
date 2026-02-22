package main

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeStartLoop struct {
	cycleErr error
	runErr   error
	runFn    func(context.Context) error
}

func (f *fakeStartLoop) Cycle(context.Context) error {
	return f.cycleErr
}

func (f *fakeStartLoop) Run(ctx context.Context) error {
	if f.runFn != nil {
		return f.runFn(ctx)
	}
	return f.runErr
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
