package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/poteto/noodle/config"
)

func TestReportConfigDiagnosticsWarnsForReadOnlyCommands(t *testing.T) {
	validation := config.ValidationResult{
		Diagnostics: []config.ConfigDiagnostic{
			{
				FieldPath: "runtime.tmux",
				Message:   "tmux is not available on PATH",
				Severity:  config.DiagnosticSeverityFatal,
				Fix:       "Install tmux.",
			},
		},
	}

	var stderr bytes.Buffer
	if err := reportConfigDiagnostics(&stderr, "commands", validation); err != nil {
		t.Fatalf("report diagnostics returned error for read-only command: %v", err)
	}

	output := stderr.String()
	if !strings.Contains(output, "config fatal: runtime.tmux: tmux is not available on PATH") {
		t.Fatalf("unexpected diagnostic output: %q", output)
	}
	if !strings.Contains(output, "Fix: Install tmux.") {
		t.Fatalf("expected fix instructions in output: %q", output)
	}
}

func TestReportConfigDiagnosticsFailsStartOnFatal(t *testing.T) {
	validation := config.ValidationResult{
		Diagnostics: []config.ConfigDiagnostic{
			{
				FieldPath: "agents.claude_dir",
				Message:   "directory not found",
				Severity:  config.DiagnosticSeverityFatal,
				Fix:       "Set agents.claude_dir in noodle.toml.",
			},
		},
	}

	var stderr bytes.Buffer
	err := reportConfigDiagnostics(&stderr, "start", validation)
	if err == nil {
		t.Fatal("expected fatal diagnostics to block start")
	}
	if !strings.Contains(err.Error(), "fatal config diagnostics prevent start") {
		t.Fatalf("unexpected error: %v", err)
	}
}
