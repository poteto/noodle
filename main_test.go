package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/internal/failure"
)

func TestReportConfigDiagnosticsWarnsForReadOnlyCommands(t *testing.T) {
	validation := config.ValidationResult{
		Diagnostics: []config.ConfigDiagnostic{
			{
				FieldPath: "agents.claude.path",
				Message:   "directory not found",
				Severity:  config.DiagnosticSeverityFatal,
				Fix:       "Set agents.claude.path in .noodle.toml.",
			},
		},
	}

	var stderr bytes.Buffer
	if err := reportConfigDiagnostics(
		context.Background(),
		&stderr,
		strings.NewReader(""),
		"status",
		&App{},
		validation,
	); err != nil {
		t.Fatalf("report diagnostics returned error for read-only command: %v", err)
	}

	output := stderr.String()
	if !strings.Contains(output, "config fatal: agents.claude.path: directory not found") {
		t.Fatalf("unexpected diagnostic output: %q", output)
	}
	if !strings.Contains(output, "Fix: Set agents.claude.path in .noodle.toml.") {
		t.Fatalf("expected fix instructions in output: %q", output)
	}
}

func TestReportConfigDiagnosticsFailsStartOnFatal(t *testing.T) {
	validation := config.ValidationResult{
		Diagnostics: []config.ConfigDiagnostic{
			{
				FieldPath: "agents.claude.path",
				Message:   "directory not found",
				Severity:  config.DiagnosticSeverityFatal,
				Fix:       "Set agents.claude.path in .noodle.toml.",
			},
		},
	}

	var stderr bytes.Buffer
	err := reportConfigDiagnostics(
		context.Background(),
		&stderr,
		strings.NewReader(""),
		"start",
		&App{},
		validation,
	)
	if err == nil {
		t.Fatal("expected fatal diagnostics to block start")
	}
	if !strings.Contains(err.Error(), "fatal config diagnostics prevent start") {
		t.Fatalf("unexpected error: %v", err)
	}
	envelope := requireStartFailureEnvelope(t, err)
	if envelope.Outcome != StartBoundaryOutcomeAbort {
		t.Fatalf("outcome = %q, want %q", envelope.Outcome, StartBoundaryOutcomeAbort)
	}
	if envelope.Class != failure.FailureClassBackendInvariant {
		t.Fatalf("class = %q, want %q", envelope.Class, failure.FailureClassBackendInvariant)
	}
	if envelope.Recoverability != failure.FailureRecoverabilityHard {
		t.Fatalf("recoverability = %q, want %q", envelope.Recoverability, failure.FailureRecoverabilityHard)
	}
	assertFailureStateMessage(t, envelope.Message)
}

func TestReportConfigDiagnosticsGroupsMissingScripts(t *testing.T) {
	validation := config.ValidationResult{
		Diagnostics: []config.ConfigDiagnostic{
			{
				FieldPath: "adapters.backlog.scripts.sync",
				Message:   `script path ".noodle/adapters/backlog-sync" not found`,
				Severity:  config.DiagnosticSeverityRepairable,
				Fix:       "Create .noodle/adapters/backlog-sync or update adapters.backlog.scripts.sync.",
				Code:      config.DiagnosticCodeAdapterScriptMissing,
				Meta: map[string]string{
					"adapter": "backlog",
					"action":  "sync",
					"path":    ".noodle/adapters/backlog-sync",
				},
			},
		},
	}

	originalTerminalCheck := terminalInteractiveCheck
	terminalInteractiveCheck = func() bool { return false }
	defer func() { terminalInteractiveCheck = originalTerminalCheck }()

	var stderr bytes.Buffer
	if err := reportConfigDiagnostics(
		context.Background(),
		&stderr,
		strings.NewReader(""),
		"start",
		&App{},
		validation,
	); err != nil {
		t.Fatalf("report diagnostics returned unexpected error: %v", err)
	}

	output := stderr.String()
	if !strings.Contains(output, "config repairable: 1 adapter script path(s) are missing.") {
		t.Fatalf("expected grouped summary in output: %q", output)
	}
	if !strings.Contains(output, "backlog:") {
		t.Fatalf("expected adapter grouping in output: %q", output)
	}
	if !strings.Contains(output, "config repair prompt:") {
		t.Fatalf("expected repair prompt block in output: %q", output)
	}
}

func TestReportConfigDiagnosticsStartPromptsAndLaunchesRepair(t *testing.T) {
	validation := config.ValidationResult{
		Diagnostics: []config.ConfigDiagnostic{
			{
				FieldPath: "adapters.backlog.scripts.sync",
				Message:   `script path ".noodle/adapters/backlog-sync" not found`,
				Severity:  config.DiagnosticSeverityRepairable,
				Code:      config.DiagnosticCodeAdapterScriptMissing,
				Meta: map[string]string{
					"adapter": "backlog",
					"action":  "sync",
					"path":    ".noodle/adapters/backlog-sync",
				},
			},
		},
	}

	originalTerminalCheck := terminalInteractiveCheck
	originalPrompt := repairSelectionPromptFunc
	originalLauncher := repairSessionLauncherFunc
	defer func() {
		terminalInteractiveCheck = originalTerminalCheck
		repairSelectionPromptFunc = originalPrompt
		repairSessionLauncherFunc = originalLauncher
	}()

	terminalInteractiveCheck = func() bool { return true }
	repairSelectionPromptFunc = func(input io.Reader, w io.Writer) (string, bool, error) {
		return "codex", true, nil
	}
	repairSessionLauncherFunc = func(
		ctx context.Context,
		app *App,
		provider string,
		prompt string,
	) (repairLaunchResult, error) {
		if provider != "codex" {
			t.Fatalf("unexpected provider: %s", provider)
		}
		if !strings.Contains(prompt, "Missing adapter scripts:") {
			t.Fatalf("expected missing scripts in prompt: %q", prompt)
		}
		return repairLaunchResult{
			SessionID:    "repair-session-1",
			WorktreePath: ".worktrees/repair-config-1",
		}, nil
	}

	var stderr bytes.Buffer
	err := reportConfigDiagnostics(
		context.Background(),
		&stderr,
		strings.NewReader(""),
		"start",
		&App{Validation: validation},
		validation,
	)
	if err == nil {
		t.Fatal("expected start to stop after launching repair session")
	}
	if !strings.Contains(err.Error(), "repair session started") {
		t.Fatalf("unexpected error: %v", err)
	}
	envelope := requireStartFailureEnvelope(t, err)
	if envelope.Outcome != StartBoundaryOutcomePromptRepair {
		t.Fatalf("outcome = %q, want %q", envelope.Outcome, StartBoundaryOutcomePromptRepair)
	}
	if envelope.Class != failure.FailureClassBackendRecoverable {
		t.Fatalf("class = %q, want %q", envelope.Class, failure.FailureClassBackendRecoverable)
	}
	if envelope.Recoverability != failure.FailureRecoverabilityRecoverable {
		t.Fatalf("recoverability = %q, want %q", envelope.Recoverability, failure.FailureRecoverabilityRecoverable)
	}
	if envelope.RepairPrompt == nil {
		t.Fatal("repair prompt metadata not captured")
	}
	if envelope.RepairPrompt.Kind != RepairPromptKindMissingScripts {
		t.Fatalf("repair prompt kind = %q, want %q", envelope.RepairPrompt.Kind, RepairPromptKindMissingScripts)
	}
	if envelope.RepairPrompt.Provider != "codex" {
		t.Fatalf("repair prompt provider = %q, want codex", envelope.RepairPrompt.Provider)
	}
	if !strings.Contains(stderr.String(), "started codex session repair-session-1") {
		t.Fatalf("expected launch confirmation in output: %q", stderr.String())
	}
}

func TestReportConfigDiagnosticsStartRepairLaunchFailure(t *testing.T) {
	validation := config.ValidationResult{
		Diagnostics: []config.ConfigDiagnostic{
			{
				FieldPath: "adapters.backlog.scripts.sync",
				Message:   `script path ".noodle/adapters/backlog-sync" not found`,
				Severity:  config.DiagnosticSeverityRepairable,
				Code:      config.DiagnosticCodeAdapterScriptMissing,
				Meta: map[string]string{
					"adapter": "backlog",
					"action":  "sync",
					"path":    ".noodle/adapters/backlog-sync",
				},
			},
		},
	}

	originalTerminalCheck := terminalInteractiveCheck
	originalPrompt := repairSelectionPromptFunc
	originalLauncher := repairSessionLauncherFunc
	defer func() {
		terminalInteractiveCheck = originalTerminalCheck
		repairSelectionPromptFunc = originalPrompt
		repairSessionLauncherFunc = originalLauncher
	}()

	terminalInteractiveCheck = func() bool { return true }
	repairSelectionPromptFunc = func(input io.Reader, w io.Writer) (string, bool, error) {
		return "claude", true, nil
	}
	repairSessionLauncherFunc = func(
		ctx context.Context,
		app *App,
		provider string,
		prompt string,
	) (repairLaunchResult, error) {
		return repairLaunchResult{}, errors.New("boom")
	}

	var stderr bytes.Buffer
	err := reportConfigDiagnostics(
		context.Background(),
		&stderr,
		strings.NewReader(""),
		"start",
		&App{Validation: validation},
		validation,
	)
	if err == nil {
		t.Fatal("expected repair launch error")
	}
	if !strings.Contains(err.Error(), "start repair session") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReportConfigDiagnosticsNoBacklogAdapterDoesNotPromptOrLaunch(t *testing.T) {
	validation := config.ValidationResult{
		Diagnostics: []config.ConfigDiagnostic{
			{
				FieldPath: "adapters.backlog",
				Message:   "no backlog adapter configured",
				Severity:  config.DiagnosticSeverityRepairable,
				Code:      config.DiagnosticCodeNoBacklogAdapter,
			},
		},
	}

	originalTerminalCheck := terminalInteractiveCheck
	originalLauncher := repairSessionLauncherFunc
	defer func() {
		terminalInteractiveCheck = originalTerminalCheck
		repairSessionLauncherFunc = originalLauncher
	}()

	terminalInteractiveCheck = func() bool { return true }
	launched := false
	repairSessionLauncherFunc = func(
		ctx context.Context,
		app *App,
		provider string,
		prompt string,
	) (repairLaunchResult, error) {
		launched = true
		return repairLaunchResult{}, nil
	}

	app := &App{
		Config:     config.DefaultConfig(),
		Validation: validation,
	}

	var stderr bytes.Buffer
	err := reportConfigDiagnostics(
		context.Background(),
		&stderr,
		strings.NewReader(""),
		"start",
		app,
		validation,
	)
	if err != nil {
		t.Fatalf("no backlog adapter should not block start: %v", err)
	}
	if launched {
		t.Fatal("no backlog adapter should not launch a repair session")
	}
	if !strings.Contains(stderr.String(), "config repairable: adapters.backlog: no backlog adapter configured") {
		t.Fatalf("expected no-backlog warning in output: %q", stderr.String())
	}
}

func TestReportConfigDiagnosticsNoBacklogNonInteractive(t *testing.T) {
	validation := config.ValidationResult{
		Diagnostics: []config.ConfigDiagnostic{
			{
				FieldPath: "adapters.backlog",
				Message:   "no backlog adapter configured",
				Severity:  config.DiagnosticSeverityRepairable,
				Code:      config.DiagnosticCodeNoBacklogAdapter,
			},
		},
	}

	originalTerminalCheck := terminalInteractiveCheck
	defer func() { terminalInteractiveCheck = originalTerminalCheck }()

	terminalInteractiveCheck = func() bool { return false }

	var stderr bytes.Buffer
	err := reportConfigDiagnostics(
		context.Background(),
		&stderr,
		strings.NewReader(""),
		"start",
		&App{Config: config.DefaultConfig(), Validation: validation},
		validation,
	)
	if err != nil {
		t.Fatalf("non-interactive should not error: %v", err)
	}
	if !strings.Contains(stderr.String(), "config repairable: adapters.backlog: no backlog adapter configured") {
		t.Fatalf("expected no-backlog warning in output: %q", stderr.String())
	}
}

func requireStartFailureEnvelope(t *testing.T, err error) StartFailureEnvelope {
	t.Helper()

	var envelope StartFailureEnvelope
	if !errors.As(err, &envelope) {
		t.Fatalf("error = %T (%v), want StartFailureEnvelope", err, err)
	}
	return envelope
}

func assertFailureStateMessage(t *testing.T, message string) {
	t.Helper()
	isWordBoundary := func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < 'A' || r > 'Z')
	}
	words := strings.FieldsFunc(strings.ToLower(message), isWordBoundary)
	for _, term := range []string{"must", "required", "requires", "expected"} {
		for _, word := range words {
			if word == term {
				t.Fatalf("message %q contains expectation-style term %q", message, term)
			}
		}
	}
}
