package skill

import (
	"strings"
	"testing"
)

func TestParseFrontmatterComplete(t *testing.T) {
	content := []byte(`---
name: deploy
description: Deploy to production
noodle:
  permissions:
    merge: false
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
	if fm.Noodle.Permissions.CanMerge() {
		t.Fatal("expected canMerge == false")
	}
	if fm.Noodle.Schedule != "After successful execute on main branch" {
		t.Fatalf("schedule = %q", fm.Noodle.Schedule)
	}
	if !strings.Contains(string(body), "# Deploy Skill") {
		t.Fatalf("body = %q", string(body))
	}
}

func TestParseFrontmatterWithoutNoodle(t *testing.T) {
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
	if fm.Noodle != nil {
		t.Fatal("expected Noodle == nil")
	}
	if !strings.Contains(string(body), "# Debugging") {
		t.Fatalf("body = %q", string(body))
	}
}

func TestParseFrontmatterPartialNoodle(t *testing.T) {
	content := []byte(`---
name: reflect
noodle:
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
	if !fm.Noodle.Permissions.CanMerge() {
		t.Fatal("expected canMerge to default to true")
	}
}

func TestParseFrontmatterMergePermissionTrue(t *testing.T) {
	content := []byte(`---
name: execute
noodle:
  permissions:
    merge: true
  schedule: "When ready"
---
Body.
`)
	fm, _, err := ParseFrontmatter(content)
	if err != nil {
		t.Fatal(err)
	}
	if !fm.Noodle.Permissions.CanMerge() {
		t.Fatal("expected canMerge == true")
	}
}

func TestParseFrontmatterMissingSchedule(t *testing.T) {
	content := []byte(`---
name: bad
noodle:
  permissions:
    merge: false
---
Body.
`)
	_, _, err := ParseFrontmatter(content)
	if err == nil {
		t.Fatal("expected error for missing schedule")
	}
	if !strings.Contains(err.Error(), "schedule is required") {
		t.Fatalf("error = %q", err.Error())
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
noodle:
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
