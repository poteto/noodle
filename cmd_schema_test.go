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
	if !strings.Contains(out, "orders\t") {
		t.Fatalf("missing orders target: %q", out)
	}
}

func TestRunSchemaOrdersOutputsMarkdown(t *testing.T) {
	out := captureStdout(t, func() {
		if err := runSchema("orders"); err != nil {
			t.Fatalf("runSchema(orders): %v", err)
		}
	})
	if !strings.Contains(out, "# orders.json Schema") {
		t.Fatalf("missing orders schema heading: %q", out)
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
