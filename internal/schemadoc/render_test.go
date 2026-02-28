package schemadoc

import (
	"strings"
	"testing"
)

func TestListTargetsIncludesMiseAndOrders(t *testing.T) {
	targets := ListTargets()
	seen := map[string]bool{}
	for _, target := range targets {
		seen[target.Name] = true
	}
	if !seen["mise"] {
		t.Fatal("missing mise schema target")
	}
	if !seen["orders"] {
		t.Fatal("missing orders schema target")
	}
}

func TestRenderMarkdownOrders(t *testing.T) {
	out, err := RenderMarkdown("orders")
	if err != nil {
		t.Fatalf("render orders markdown: %v", err)
	}
	if !strings.Contains(out, "# orders.json Schema") {
		t.Fatalf("missing orders title: %q", out)
	}
	if !strings.Contains(out, `"orders": [`) {
		t.Fatalf("missing orders field in output: %q", out)
	}
}

func TestRenderPromptJSONOrders(t *testing.T) {
	out, err := RenderPromptJSON("orders")
	if err != nil {
		t.Fatalf("render orders prompt schema: %v", err)
	}
	if !strings.Contains(out, "orders.json schema (JSON):") {
		t.Fatalf("missing orders prompt heading: %q", out)
	}
	if !strings.Contains(out, "```json\n") {
		t.Fatalf("missing opening json code fence: %q", out)
	}
	if !strings.HasSuffix(out, "\n```") {
		t.Fatalf("missing closing code fence: %q", out)
	}
	if !strings.Contains(out, `"generated_at":`) {
		t.Fatalf("missing generated_at in orders prompt schema: %q", out)
	}
}

func TestRenderPromptJSONMiseIncludesBacklogPlan(t *testing.T) {
	out, err := RenderPromptJSON("mise")
	if err != nil {
		t.Fatalf("render mise prompt schema: %v", err)
	}
	if !strings.Contains(out, `"plan"`) {
		t.Fatalf("missing backlog[].plan in mise prompt schema: %q", out)
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
