package main

import (
	"strings"
	"testing"
)

func TestRunSchemaListOutputsTargets(t *testing.T) {
	out := captureStdout(t, func() {
		if err := runSchemaList(); err != nil {
			t.Fatalf("runSchemaList: %v", err)
		}
	})
	if !strings.Contains(out, "mise\t") {
		t.Fatalf("missing mise target: %q", out)
	}
	if !strings.Contains(out, "queue\t") {
		t.Fatalf("missing queue target: %q", out)
	}
}

func TestRunSchemaQueueOutputsMarkdown(t *testing.T) {
	out := captureStdout(t, func() {
		if err := runSchema("queue"); err != nil {
			t.Fatalf("runSchema(queue): %v", err)
		}
	})
	if !strings.Contains(out, "# queue.json Schema") {
		t.Fatalf("missing queue schema heading: %q", out)
	}
	if !strings.Contains(out, "\"items\": [") {
		t.Fatalf("missing queue items in schema output: %q", out)
	}
}

func TestRunSchemaUnknownTarget(t *testing.T) {
	err := runSchema("unknown")
	if err == nil {
		t.Fatal("expected unknown schema target error")
	}
	if !strings.Contains(err.Error(), "schema target not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}
