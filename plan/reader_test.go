package plan

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadAll_Basic(t *testing.T) {
	dir := t.TempDir()
	plansDir := filepath.Join(dir, "plans")
	planDir := filepath.Join(plansDir, "42-my-plan")

	mustMkdir(t, planDir)

	writeFile(t, filepath.Join(plansDir, "index.md"),
		"# Plans\n\n- [[plans/42-my-plan/overview]]\n")

	writeFile(t, filepath.Join(planDir, "overview.md"),
		"---\nid: 42\ncreated: 2026-02-20\nstatus: active\n---\n\n# My Great Plan\n\nSome body text.\n")

	writeFile(t, filepath.Join(planDir, "phase-01-scaffold.md"),
		"# Scaffold the Thing\n\nDetails here.\n")

	writeFile(t, filepath.Join(planDir, "phase-02-implement.md"),
		"# Implement the Thing\n\nMore details.\n")

	plans, err := ReadAll(plansDir)
	if err != nil {
		t.Fatalf("ReadAll returned error: %v", err)
	}
	if len(plans) != 1 {
		t.Fatalf("expected 1 plan, got %d", len(plans))
	}

	p := plans[0]
	if p.Meta.ID != 42 {
		t.Errorf("expected ID 42, got %d", p.Meta.ID)
	}
	if p.Meta.Status != "active" {
		t.Errorf("expected status active, got %q", p.Meta.Status)
	}
	if p.Meta.Provider != "" {
		t.Errorf("expected empty provider, got %q", p.Meta.Provider)
	}
	if p.Meta.Model != "" {
		t.Errorf("expected empty model, got %q", p.Meta.Model)
	}
	if p.Title != "My Great Plan" {
		t.Errorf("expected title %q, got %q", "My Great Plan", p.Title)
	}
	if p.Slug != "42-my-plan" {
		t.Errorf("expected slug %q, got %q", "42-my-plan", p.Slug)
	}
	if p.Directory == "" {
		t.Error("expected non-empty directory")
	}
	if len(p.Phases) != 2 {
		t.Fatalf("expected 2 phases, got %d", len(p.Phases))
	}
	if p.Phases[0].Filename != "phase-01-scaffold.md" {
		t.Errorf("expected first phase filename %q, got %q", "phase-01-scaffold.md", p.Phases[0].Filename)
	}
	if p.Phases[0].Name != "Scaffold the Thing" {
		t.Errorf("expected first phase name %q, got %q", "Scaffold the Thing", p.Phases[0].Name)
	}
	if p.Phases[1].Filename != "phase-02-implement.md" {
		t.Errorf("expected second phase filename %q, got %q", "phase-02-implement.md", p.Phases[1].Filename)
	}
}

func TestReadAll_DonePlansExcluded(t *testing.T) {
	dir := t.TempDir()
	plansDir := filepath.Join(dir, "plans")
	doneDir := filepath.Join(plansDir, "01-done-plan")
	activeDir := filepath.Join(plansDir, "02-active-plan")

	mustMkdir(t, doneDir)
	mustMkdir(t, activeDir)

	writeFile(t, filepath.Join(plansDir, "index.md"),
		"# Plans\n\n- [[plans/01-done-plan/overview]]\n- [[plans/02-active-plan/overview]]\n")

	writeFile(t, filepath.Join(doneDir, "overview.md"),
		"---\nid: 1\ncreated: 2026-02-20\nstatus: done\n---\n\n# Done Plan\n")

	writeFile(t, filepath.Join(activeDir, "overview.md"),
		"---\nid: 2\ncreated: 2026-02-21\nstatus: ready\n---\n\n# Active Plan\n")

	plans, err := ReadAll(plansDir)
	if err != nil {
		t.Fatalf("ReadAll returned error: %v", err)
	}
	if len(plans) != 1 {
		t.Fatalf("expected 1 plan, got %d", len(plans))
	}
	if plans[0].Meta.ID != 2 {
		t.Errorf("expected ID 2, got %d", plans[0].Meta.ID)
	}
}

func TestReadAll_MissingIndex(t *testing.T) {
	dir := t.TempDir()
	plansDir := filepath.Join(dir, "plans")
	mustMkdir(t, plansDir)
	// No index.md created.

	plans, err := ReadAll(plansDir)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(plans) != 0 {
		t.Errorf("expected empty slice, got %d plans", len(plans))
	}
}

func TestReadAll_MalformedOverviewSkipped(t *testing.T) {
	dir := t.TempDir()
	plansDir := filepath.Join(dir, "plans")
	badDir := filepath.Join(plansDir, "99-bad")
	goodDir := filepath.Join(plansDir, "10-good")

	mustMkdir(t, badDir)
	mustMkdir(t, goodDir)

	writeFile(t, filepath.Join(plansDir, "index.md"),
		"# Plans\n\n- [[plans/99-bad/overview]]\n- [[plans/10-good/overview]]\n")

	// Malformed: no frontmatter delimiters.
	writeFile(t, filepath.Join(badDir, "overview.md"),
		"This file has no frontmatter at all.\n")

	writeFile(t, filepath.Join(goodDir, "overview.md"),
		"---\nid: 10\ncreated: 2026-02-22\nstatus: active\n---\n\n# Good Plan\n")

	plans, err := ReadAll(plansDir)
	if err != nil {
		t.Fatalf("ReadAll returned error: %v", err)
	}
	if len(plans) != 1 {
		t.Fatalf("expected 1 plan (bad skipped), got %d", len(plans))
	}
	if plans[0].Slug != "10-good" {
		t.Errorf("expected slug %q, got %q", "10-good", plans[0].Slug)
	}
}

func TestReadAll_PhasesDiscoveredAndSorted(t *testing.T) {
	dir := t.TempDir()
	plansDir := filepath.Join(dir, "plans")
	planDir := filepath.Join(plansDir, "05-multi-phase")

	mustMkdir(t, planDir)

	writeFile(t, filepath.Join(plansDir, "index.md"),
		"- [[plans/05-multi-phase/overview]]\n")

	writeFile(t, filepath.Join(planDir, "overview.md"),
		"---\nid: 5\ncreated: 2026-02-22\nstatus: active\n---\n\n# Multi Phase Plan\n")

	// Create phases out of numeric order to verify sorting.
	writeFile(t, filepath.Join(planDir, "phase-03-last.md"),
		"# Last Phase\n")
	writeFile(t, filepath.Join(planDir, "phase-01-first.md"),
		"# First Phase\n")
	writeFile(t, filepath.Join(planDir, "phase-02-middle.md"),
		"# Middle Phase\n")

	plans, err := ReadAll(plansDir)
	if err != nil {
		t.Fatalf("ReadAll returned error: %v", err)
	}
	if len(plans) != 1 {
		t.Fatalf("expected 1 plan, got %d", len(plans))
	}

	phases := plans[0].Phases
	if len(phases) != 3 {
		t.Fatalf("expected 3 phases, got %d", len(phases))
	}

	expectedOrder := []string{"phase-01-first.md", "phase-02-middle.md", "phase-03-last.md"}
	for i, want := range expectedOrder {
		if phases[i].Filename != want {
			t.Errorf("phase[%d]: expected filename %q, got %q", i, want, phases[i].Filename)
		}
	}

	expectedNames := []string{"First Phase", "Middle Phase", "Last Phase"}
	for i, want := range expectedNames {
		if phases[i].Name != want {
			t.Errorf("phase[%d]: expected name %q, got %q", i, want, phases[i].Name)
		}
	}
}

func TestReadAll_ChecklistPhaseStatus(t *testing.T) {
	dir := t.TempDir()
	plansDir := filepath.Join(dir, "plans")
	planDir := filepath.Join(plansDir, "07-checklist")

	mustMkdir(t, planDir)

	writeFile(t, filepath.Join(plansDir, "index.md"),
		"- [[plans/07-checklist/overview]]\n")

	// Overview with checklist: phase-01 checked, phase-02 unchecked, phase-03 unchecked.
	writeFile(t, filepath.Join(planDir, "overview.md"),
		"---\nid: 7\ncreated: 2026-02-22\nstatus: active\n---\n\n# Checklist Plan\n\n## Phases\n\n"+
			"- [x] [[plans/07-checklist/phase-01-setup]] -- Setup\n"+
			"- [ ] [[plans/07-checklist/phase-02-build]] -- Build\n"+
			"- [ ] [[plans/07-checklist/phase-03-ship]] -- Ship\n")

	writeFile(t, filepath.Join(planDir, "phase-01-setup.md"),
		"# Setup\n\nDone.\n")
	writeFile(t, filepath.Join(planDir, "phase-02-build.md"),
		"# Build\n\nIn progress.\n")
	writeFile(t, filepath.Join(planDir, "phase-03-ship.md"),
		"# Ship\n\nNot started.\n")

	plans, err := ReadAll(plansDir)
	if err != nil {
		t.Fatalf("ReadAll returned error: %v", err)
	}
	if len(plans) != 1 {
		t.Fatalf("expected 1 plan, got %d", len(plans))
	}

	phases := plans[0].Phases
	if len(phases) != 3 {
		t.Fatalf("expected 3 phases, got %d", len(phases))
	}

	wantStatuses := []struct {
		filename string
		status   string
	}{
		{"phase-01-setup.md", "done"},
		{"phase-02-build.md", "active"},
		{"phase-03-ship.md", "pending"},
	}
	for i, want := range wantStatuses {
		if phases[i].Filename != want.filename {
			t.Errorf("phase[%d]: expected filename %q, got %q", i, want.filename, phases[i].Filename)
		}
		if phases[i].Status != want.status {
			t.Errorf("phase[%d] (%s): expected status %q, got %q", i, want.filename, want.status, phases[i].Status)
		}
	}
}

func TestReadAll_NoChecklistDefaultsPending(t *testing.T) {
	dir := t.TempDir()
	plansDir := filepath.Join(dir, "plans")
	planDir := filepath.Join(plansDir, "08-no-checklist")

	mustMkdir(t, planDir)

	writeFile(t, filepath.Join(plansDir, "index.md"),
		"- [[plans/08-no-checklist/overview]]\n")

	writeFile(t, filepath.Join(planDir, "overview.md"),
		"---\nid: 8\ncreated: 2026-02-22\nstatus: ready\n---\n\n# No Checklist Plan\n\nJust a description.\n")

	writeFile(t, filepath.Join(planDir, "phase-01-only.md"),
		"# Only Phase\n")

	plans, err := ReadAll(plansDir)
	if err != nil {
		t.Fatalf("ReadAll returned error: %v", err)
	}
	if len(plans) != 1 {
		t.Fatalf("expected 1 plan, got %d", len(plans))
	}
	if plans[0].Phases[0].Status != "pending" {
		t.Errorf("expected status %q, got %q", "pending", plans[0].Phases[0].Status)
	}
}

func TestReadAll_UnreadablePhaseFileSkipped(t *testing.T) {
	dir := t.TempDir()
	plansDir := filepath.Join(dir, "plans")
	planDir := filepath.Join(plansDir, "09-bad-phase")

	mustMkdir(t, planDir)

	writeFile(t, filepath.Join(plansDir, "index.md"),
		"- [[plans/09-bad-phase/overview]]\n")

	writeFile(t, filepath.Join(planDir, "overview.md"),
		"---\nid: 9\ncreated: 2026-02-22\nstatus: active\n---\n\n# Bad Phase Plan\n")

	writeFile(t, filepath.Join(planDir, "phase-01-good.md"),
		"# Good Phase\n")

	// Create an unreadable phase file (directory with same name as expected file).
	badPhase := filepath.Join(planDir, "phase-02-bad.md")
	if err := os.Mkdir(badPhase, 0o755); err != nil {
		t.Fatalf("failed to create bad phase dir: %v", err)
	}

	plans, err := ReadAll(plansDir)
	if err != nil {
		t.Fatalf("ReadAll returned error: %v", err)
	}
	if len(plans) != 1 {
		t.Fatalf("expected 1 plan, got %d", len(plans))
	}

	// The "bad" phase should use the cleaned filename since reading fails,
	// but the glob still matched it. The phaseNameFromFile function handles
	// the read error and returns a cleaned filename.
	phases := plans[0].Phases
	if len(phases) != 2 {
		t.Fatalf("expected 2 phases (glob matched both), got %d", len(phases))
	}
	if phases[0].Name != "Good Phase" {
		t.Errorf("expected first phase name %q, got %q", "Good Phase", phases[0].Name)
	}
	// The unreadable phase falls back to cleaned filename.
	if phases[1].Name != "Phase 02 Bad" {
		t.Errorf("expected second phase name %q, got %q", "Phase 02 Bad", phases[1].Name)
	}
}

func TestExtractSlugs(t *testing.T) {
	content := `# Plans

- [[plans/01-first/overview]]
- [ ] [[plans/23-task-type-skill-suite/overview]]
- [x] [[plans/42-done/overview]]
- Some other line
`
	slugs := extractSlugs(content)
	want := []string{"01-first", "23-task-type-skill-suite", "42-done"}
	if len(slugs) != len(want) {
		t.Fatalf("expected %d slugs, got %d: %v", len(want), len(slugs), slugs)
	}
	for i, s := range want {
		if slugs[i] != s {
			t.Errorf("slug[%d]: expected %q, got %q", i, s, slugs[i])
		}
	}
}

func TestCleanFilename(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"phase-01-scaffold", "Phase 01 Scaffold"},
		{"phase-12-cleanup", "Phase 12 Cleanup"},
		{"simple", "Simple"},
	}
	for _, tt := range tests {
		got := cleanFilename(tt.input)
		if got != tt.want {
			t.Errorf("cleanFilename(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestParseFrontmatter(t *testing.T) {
	content := "---\nid: 5\ncreated: 2026-02-22\nstatus: ready\nprovider: codex\nmodel: gpt-5.3-codex\n---\n\n# Title\n\nBody text.\n"
	meta, body, ok := parseFrontmatter(content)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if meta.ID != 5 {
		t.Errorf("expected id 5, got %d", meta.ID)
	}
	if meta.Status != "ready" {
		t.Errorf("expected status draft, got %q", meta.Status)
	}
	if meta.Provider != "codex" {
		t.Errorf("expected provider codex, got %q", meta.Provider)
	}
	if meta.Model != "gpt-5.3-codex" {
		t.Errorf("expected model gpt-5.3-codex, got %q", meta.Model)
	}
	if !contains(body, "# Title") {
		t.Errorf("expected body to contain heading, got %q", body)
	}
}

func TestParseFrontmatter_NoFrontmatter(t *testing.T) {
	_, _, ok := parseFrontmatter("Just some markdown.\n")
	if ok {
		t.Error("expected ok=false for content without frontmatter")
	}
}

// helpers

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
