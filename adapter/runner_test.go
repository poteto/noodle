package adapter

import (
	"context"
	"strings"
	"testing"

	"github.com/poteto/noodle/config"
)

func TestRunnerRunSuccessWithArgs(t *testing.T) {
	runner := NewRunner(t.TempDir(), config.Config{
		Adapters: map[string]config.AdapterConfig{
			"backlog": {
				Scripts: map[string]string{
					"done": "printf '%s' \"$1\"",
				},
			},
		},
	})

	output, err := runner.Run(context.Background(), "backlog", "done", RunOptions{Args: []string{"item 42"}})
	if err != nil {
		t.Fatalf("run done action: %v", err)
	}
	if strings.TrimSpace(output) != "item 42" {
		t.Fatalf("output = %q", output)
	}
}

func TestRunnerRunCapturesStderrOnError(t *testing.T) {
	runner := NewRunner(t.TempDir(), config.Config{
		Adapters: map[string]config.AdapterConfig{
			"backlog": {
				Scripts: map[string]string{
					"sync": "echo 'boom' 1>&2; exit 7",
				},
			},
		},
	})

	_, err := runner.Run(context.Background(), "backlog", "sync", RunOptions{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected stderr in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "echo 'boom' 1>&2; exit 7") {
		t.Fatalf("expected command in error, got: %v", err)
	}
}

func TestParseBacklogItemsValidation(t *testing.T) {
	items, warnings, err := ParseBacklogItems(`{"title":"Missing ID","status":"open"}`)
	if err != nil {
		t.Fatalf("expected recoverable parse warning, got error: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("items len = %d, want 0", len(items))
	}
	if len(warnings) == 0 {
		t.Fatal("expected validation warning")
	}
	if !strings.Contains(warnings[0], "missing required field id") {
		t.Fatalf("unexpected warning: %v", warnings[0])
	}
}

func TestParseBacklogItemsSkipsInvalidJSON(t *testing.T) {
	items, warnings, err := ParseBacklogItems(strings.Join([]string{
		`{"id":"42","title":"valid","status":"open"}`,
		`P0 fix me`,
		`{"id":"43","title":"valid two","status":"open"}`,
	}, "\n"))
	if err != nil {
		t.Fatalf("expected recoverable parse warning, got error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("items len = %d, want 2", len(items))
	}
	if len(warnings) != 1 {
		t.Fatalf("warnings len = %d, want 1", len(warnings))
	}
	if !strings.Contains(warnings[0], "parse backlog sync line 2") {
		t.Fatalf("unexpected warning: %v", warnings[0])
	}
	if !strings.Contains(warnings[0], "invalid character") {
		t.Fatalf("unexpected warning: %v", warnings[0])
	}
}
