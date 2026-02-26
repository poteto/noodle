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
	if !strings.Contains(out, "```json\n") {
		t.Fatalf("missing opening json code fence: %q", out)
	}
	if !strings.HasSuffix(out, "\n```") {
		t.Fatalf("missing closing code fence: %q", out)
	}
	if !strings.Contains(out, `"generated_at":`) {
		t.Fatalf("missing generated_at in queue prompt schema: %q", out)
	}
}

func TestRenderPromptJSONMiseIncludesPlanRoutingHints(t *testing.T) {
	out, err := RenderPromptJSON("mise")
	if err != nil {
		t.Fatalf("render mise prompt schema: %v", err)
	}
	if !strings.Contains(out, `"provider": "string (optional) - optional plan-level provider hint"`) {
		t.Fatalf("missing plans[].provider hint in mise prompt schema: %q", out)
	}
	if !strings.Contains(out, `"model": "string (optional) - optional plan-level model hint"`) {
		t.Fatalf("missing plans[].model hint in mise prompt schema: %q", out)
	}
	if !strings.Contains(out, `"available_runtimes": [`) {
		t.Fatalf("missing routing.available_runtimes in mise prompt schema: %q", out)
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
