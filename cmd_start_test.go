package main

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/internal/failure"
	"github.com/poteto/noodle/internal/statever"
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
	var capturedDeps loop.Dependencies
	originalFactory := newStartRuntimeLoop
	newStartRuntimeLoop = func(_ string, _ string, _ config.Config, deps loop.Dependencies) startRuntimeLoop {
		capturedDeps = deps
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
	if capturedDeps.Logger == nil {
		t.Fatal("expected runStart to inject a logger dependency")
	}
}

func TestRunStartRefusesFutureStateVersion(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir .noodle: %v", err)
	}
	if err := statever.Write(filepath.Join(runtimeDir, "state.json"), statever.StateMarker{
		SchemaVersion: statever.Current + 1,
		GeneratedAt:   time.Now(),
	}); err != nil {
		t.Fatalf("write state marker: %v", err)
	}

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("chdir project dir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(originalDir) })

	loopCreated := false
	originalFactory := newStartRuntimeLoop
	newStartRuntimeLoop = func(_ string, _ string, _ config.Config, _ loop.Dependencies) startRuntimeLoop {
		loopCreated = true
		return &fakeStartLoop{}
	}
	t.Cleanup(func() { newStartRuntimeLoop = originalFactory })

	app := &App{Config: config.DefaultConfig()}
	err = runStart(context.Background(), app, startOptions{once: true})
	if err == nil {
		t.Fatal("expected state compatibility error")
	}
	if !strings.Contains(err.Error(), "state version") {
		t.Fatalf("error = %q, want state version failure", err)
	}
	if loopCreated {
		t.Fatal("loop should not be created when state version is incompatible")
	}
}

func TestShouldStartServer(t *testing.T) {
	boolPtr := func(v bool) *bool { return &v }

	tests := []struct {
		name    string
		enabled *bool
		want    bool
	}{
		{"nil defaults to true", nil, true},
		{"explicit true", boolPtr(true), true},
		{"explicit false", boolPtr(false), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.ServerConfig{Enabled: tt.enabled}
			got := shouldStartServer(cfg)
			if got != tt.want {
				t.Fatalf("shouldStartServer(%v) = %v, want %v", tt.enabled, got, tt.want)
			}
		})
	}
}

func TestNewAPILogger(t *testing.T) {
	var buf bytes.Buffer
	logger := newAPILogger(&buf)

	if got := logger.GetPrefix(); got != "api" {
		t.Fatalf("prefix = %q, want api", got)
	}

	slog.New(logger).Info("orders-next promoted", "order", "80")
	out := buf.String()
	if !strings.Contains(out, "orders-next promoted") {
		t.Fatalf("log output missing message: %q", out)
	}
	if !strings.Contains(out, "order=80") {
		t.Fatalf("log output missing attr: %q", out)
	}
}

func TestNormalizeLoopbackHost(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "ipv4 loopback", in: "127.0.0.1:3000", want: "localhost:3000"},
		{name: "ipv6 loopback", in: "[::1]:3000", want: "localhost:3000"},
		{name: "hostname stays", in: "example.com:3000", want: "example.com:3000"},
		{name: "empty stays empty", in: "", want: ""},
		{name: "invalid addr unchanged", in: "bad-address", want: "bad-address"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeLoopbackHost(tt.in); got != tt.want {
				t.Fatalf("normalizeLoopbackHost(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestOpenBrowserClassifiesBestEffortAsWarningOnly(t *testing.T) {
	originalLaunch := launchBrowserCommandFunc
	launchBrowserCommandFunc = func(string) error {
		return errors.New("display unavailable")
	}
	t.Cleanup(func() { launchBrowserCommandFunc = originalLaunch })

	envelope := openBrowser("http://localhost:3000")
	if envelope.Outcome != StartBoundaryOutcomeWarningOnly {
		t.Fatalf("outcome = %q, want %q", envelope.Outcome, StartBoundaryOutcomeWarningOnly)
	}
	if envelope.Class != failure.FailureClassWarningOnly {
		t.Fatalf("class = %q, want %q", envelope.Class, failure.FailureClassWarningOnly)
	}
	if envelope.Recoverability != failure.FailureRecoverabilityDegrade {
		t.Fatalf("recoverability = %q, want %q", envelope.Recoverability, failure.FailureRecoverabilityDegrade)
	}
	if envelope.Cause == nil {
		t.Fatal("cause missing for browser launch failure")
	}
	if !strings.Contains(envelope.Error(), "display unavailable") {
		t.Fatalf("error = %q, want launch failure details", envelope.Error())
	}
}
