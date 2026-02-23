package schemadoc

import (
	"strings"
	"testing"
)

func TestListTargetsIncludesMiseAndQueue(t *testing.T) {
	targets := ListTargets()
	seen := map[string]bool{}
	for _, target := range targets {
		seen[target.Name] = true
	}
	if !seen["mise"] {
		t.Fatal("missing mise schema target")
	}
	if !seen["queue"] {
		t.Fatal("missing queue schema target")
	}
}

func TestRenderMarkdownQueue(t *testing.T) {
	out, err := RenderMarkdown("queue")
	if err != nil {
		t.Fatalf("render queue markdown: %v", err)
	}
	if !strings.Contains(out, "# queue.json Schema") {
		t.Fatalf("missing queue title: %q", out)
	}
	if !strings.Contains(out, `"items": [`) {
		t.Fatalf("missing items field in output: %q", out)
	}
	if !strings.Contains(out, "must match a task_types[].key from mise") {
		t.Fatalf("missing task_key description: %q", out)
	}
	if !strings.Contains(out, "## Constraints") {
		t.Fatalf("missing constraints section: %q", out)
	}
}

func TestRenderPromptJSONQueue(t *testing.T) {
	out, err := RenderPromptJSON("queue")
	if err != nil {
		t.Fatalf("render queue prompt schema: %v", err)
	}
	if !strings.Contains(out, "queue.json schema (JSON):") {
		t.Fatalf("missing queue prompt heading: %q", out)
	}
	if !strings.Contains(out, `"generated_at":`) {
		t.Fatalf("missing generated_at in queue prompt schema: %q", out)
	}
}

func TestRenderUnknownTargetReturnsError(t *testing.T) {
	_, err := RenderMarkdown("does-not-exist")
	if err == nil {
		t.Fatal("expected unknown target error")
	}
	if !strings.Contains(err.Error(), "schema target not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateSpecCoverage(t *testing.T) {
	for _, target := range schemaTargets {
		if err := validateSpec(target); err != nil {
			t.Fatalf("validate %s: %v", target.Info.Name, err)
		}
	}
}
