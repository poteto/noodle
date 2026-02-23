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

func TestRunnerRunCapturesStderrOnFailure(t *testing.T) {
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
}

func TestParseBacklogItemsValidation(t *testing.T) {
	_, err := ParseBacklogItems(`{"title":"Missing ID","status":"open"}`)
	if err == nil {
		t.Fatal("expected missing id error")
	}
	if !strings.Contains(err.Error(), "missing required field id") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseBacklogItemsOptionalDefaults(t *testing.T) {
	items, err := ParseBacklogItems(`{"id":"1","title":"Fix bug","status":"open"}`)
	if err != nil {
		t.Fatalf("parse backlog: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("item count = %d", len(items))
	}
	if items[0].Tags == nil || len(items[0].Tags) != 0 {
		t.Fatalf("tags should default to empty array: %#v", items[0].Tags)
	}
}

