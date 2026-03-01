package skill

import (
	"strings"
	"testing"
)

func TestParseFrontmatterComplete(t *testing.T) {
	content := []byte(`---
name: deploy
description: Deploy to production
schedule: "After successful execute on main branch"
---
# Deploy Skill

Instructions here.
`)
	fm, body, err := ParseFrontmatter(content)
	if err != nil {
		t.Fatal(err)
	}
	if fm.Name != "deploy" {
		t.Fatalf("name = %q", fm.Name)
	}
	if fm.Description != "Deploy to production" {
		t.Fatalf("description = %q", fm.Description)
	}
	if !fm.IsTaskType() {
		t.Fatal("expected IsTaskType() == true")
	}
	if fm.Schedule != "After successful execute on main branch" {
		t.Fatalf("schedule = %q", fm.Schedule)
	}
	if !strings.Contains(string(body), "# Deploy Skill") {
		t.Fatalf("body = %q", string(body))
	}
}

func TestParseFrontmatterWithoutSchedule(t *testing.T) {
	content := []byte(`---
name: debugging
description: Systematic debugging methodology
---
# Debugging

Body text.
`)
	fm, body, err := ParseFrontmatter(content)
	if err != nil {
		t.Fatal(err)
	}
	if fm.Name != "debugging" {
		t.Fatalf("name = %q", fm.Name)
	}
	if fm.IsTaskType() {
		t.Fatal("expected IsTaskType() == false")
	}
	if fm.Schedule != "" {
		t.Fatalf("expected empty schedule, got %q", fm.Schedule)
	}
	if !strings.Contains(string(body), "# Debugging") {
		t.Fatalf("body = %q", string(body))
	}
}

func TestParseFrontmatterScheduleOnly(t *testing.T) {
	content := []byte(`---
name: reflect
schedule: "After a cook session completes"
---
Body.
`)
	fm, _, err := ParseFrontmatter(content)
	if err != nil {
		t.Fatal(err)
	}
	if !fm.IsTaskType() {
		t.Fatal("expected IsTaskType() == true")
	}
	if fm.Schedule != "After a cook session completes" {
		t.Fatalf("schedule = %q", fm.Schedule)
	}
}

func TestParseFrontmatterNoScheduleDoesNotError(t *testing.T) {
	content := []byte(`---
name: bad
---
Body.
`)
	fm, _, err := ParseFrontmatter(content)
	if err != nil {
		t.Fatalf("expected no error when schedule is absent, got: %v", err)
	}
	if fm.IsTaskType() {
		t.Fatal("expected IsTaskType() == false when schedule is absent")
	}
}

func TestParseFrontmatterNoFrontmatter(t *testing.T) {
	content := []byte("# Just Markdown\n\nNo frontmatter here.\n")
	fm, body, err := ParseFrontmatter(content)
	if err != nil {
		t.Fatal(err)
	}
	if fm.Name != "" {
		t.Fatalf("expected zero frontmatter, got name = %q", fm.Name)
	}
	if fm.IsTaskType() {
		t.Fatal("expected IsTaskType() == false")
	}
	if string(body) != string(content) {
		t.Fatalf("body should equal original content")
	}
}

func TestParseFrontmatterInvalidYAML(t *testing.T) {
	content := []byte(`---
name: [invalid yaml
  : broken
---
Body.
`)
	_, _, err := ParseFrontmatter(content)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestStripFrontmatter(t *testing.T) {
	content := []byte(`---
name: test
description: Test skill
schedule: "Always"
---
# Skill Body

Instructions.
`)
	body := StripFrontmatter(content)
	if strings.Contains(string(body), "name: test") {
		t.Fatal("frontmatter not stripped")
	}
	if !strings.Contains(string(body), "# Skill Body") {
		t.Fatalf("body = %q", string(body))
	}
}

func TestStripFrontmatterNoFrontmatter(t *testing.T) {
	content := []byte("# No Frontmatter\n\nJust text.\n")
	body := StripFrontmatter(content)
	if string(body) != string(content) {
		t.Fatal("content should be unchanged")
	}
}

func TestStripFrontmatterInvalidYAML(t *testing.T) {
	content := []byte(`---
: [broken
---
Body.
`)
	body := StripFrontmatter(content)
	if string(body) != string(content) {
		t.Fatal("invalid YAML should return original content")
	}
}
