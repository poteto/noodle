package plan

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreate_MakesDirectoryAndFiles(t *testing.T) {
	dir := t.TempDir()
	plansDir := filepath.Join(dir, "plans")
	mustMkdir(t, plansDir)
	writeFile(t, filepath.Join(plansDir, "index.md"), "# Plans\n")

	created, err := Create(plansDir, 99, "test-plan")
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	// Verify directory was created.
	info, err := os.Stat(created)
	if err != nil {
		t.Fatalf("created directory not found: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("created path not a directory")
	}

	// Verify overview.md.
	overview, err := os.ReadFile(filepath.Join(created, "overview.md"))
	if err != nil {
		t.Fatalf("overview.md not found: %v", err)
	}
	overviewStr := string(overview)
	if !strings.Contains(overviewStr, "id: 99") {
		t.Errorf("overview missing id: %s", overviewStr)
	}
	if !strings.Contains(overviewStr, "status: draft") {
		t.Errorf("overview missing status: %s", overviewStr)
	}
	if !strings.Contains(overviewStr, "# Test Plan") {
		t.Errorf("overview missing title: %s", overviewStr)
	}
	if !strings.Contains(overviewStr, "created:") {
		t.Errorf("overview missing created date: %s", overviewStr)
	}

	// Verify phase-01-scaffold.md.
	phase01, err := os.ReadFile(filepath.Join(created, "phase-01-scaffold.md"))
	if err != nil {
		t.Fatalf("phase-01-scaffold.md not found: %v", err)
	}
	phase01Str := string(phase01)
	if !strings.Contains(phase01Str, "[[plans/99-test-plan/overview]]") {
		t.Errorf("phase-01 missing back link: %s", phase01Str)
	}
	if !strings.Contains(phase01Str, "# Phase 1: Scaffold") {
		t.Errorf("phase-01 missing heading: %s", phase01Str)
	}

	// Verify index.md was updated.
	index, err := os.ReadFile(filepath.Join(plansDir, "index.md"))
	if err != nil {
		t.Fatalf("index.md not readable: %v", err)
	}
	if !strings.Contains(string(index), "[[plans/99-test-plan/overview]]") {
		t.Errorf("index.md missing wikilink: %s", string(index))
	}
}

func TestCreate_CreatesIndexIfMissing(t *testing.T) {
	dir := t.TempDir()
	plansDir := filepath.Join(dir, "plans")
	mustMkdir(t, plansDir)

	_, err := Create(plansDir, 50, "no-index")
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	index, err := os.ReadFile(filepath.Join(plansDir, "index.md"))
	if err != nil {
		t.Fatalf("index.md not created: %v", err)
	}
	if !strings.Contains(string(index), "# Plans") {
		t.Errorf("index.md missing header: %s", string(index))
	}
	if !strings.Contains(string(index), "[[plans/50-no-index/overview]]") {
		t.Errorf("index.md missing wikilink: %s", string(index))
	}
}

func TestDone_FlipsStatusFromDraft(t *testing.T) {
	dir := t.TempDir()
	plansDir := filepath.Join(dir, "plans")
	planDir := filepath.Join(plansDir, "10-my-plan")
	mustMkdir(t, planDir)

	writeFile(t, filepath.Join(planDir, "overview.md"),
		"---\nid: 10\ncreated: 2026-02-20\nstatus: draft\n---\n\n# My Plan\n")

	if err := Done(plansDir, 10); err != nil {
		t.Fatalf("Done returned error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(planDir, "overview.md"))
	if err != nil {
		t.Fatalf("overview.md not readable: %v", err)
	}
	if !strings.Contains(string(data), "status: done") {
		t.Errorf("status not changed to done: %s", string(data))
	}
	if strings.Contains(string(data), "status: draft") {
		t.Errorf("draft status still present: %s", string(data))
	}
}

func TestDone_FlipsStatusFromActive(t *testing.T) {
	dir := t.TempDir()
	plansDir := filepath.Join(dir, "plans")
	planDir := filepath.Join(plansDir, "11-active-plan")
	mustMkdir(t, planDir)

	writeFile(t, filepath.Join(planDir, "overview.md"),
		"---\nid: 11\ncreated: 2026-02-20\nstatus: active\n---\n\n# Active Plan\n")

	if err := Done(plansDir, 11); err != nil {
		t.Fatalf("Done returned error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(planDir, "overview.md"))
	if err != nil {
		t.Fatalf("overview.md not readable: %v", err)
	}
	if !strings.Contains(string(data), "status: done") {
		t.Errorf("status not changed to done: %s", string(data))
	}
}

func TestDone_ErrorsOnAlreadyDone(t *testing.T) {
	dir := t.TempDir()
	plansDir := filepath.Join(dir, "plans")
	planDir := filepath.Join(plansDir, "12-done-plan")
	mustMkdir(t, planDir)

	writeFile(t, filepath.Join(planDir, "overview.md"),
		"---\nid: 12\ncreated: 2026-02-20\nstatus: done\n---\n\n# Done Plan\n")

	err := Done(plansDir, 12)
	if err == nil {
		t.Fatal("expected error for already-done plan")
	}
	if !strings.Contains(err.Error(), "not draft or active") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDone_ErrorsOnMissingPlan(t *testing.T) {
	dir := t.TempDir()
	plansDir := filepath.Join(dir, "plans")
	mustMkdir(t, plansDir)

	err := Done(plansDir, 999)
	if err == nil {
		t.Fatal("expected error for missing plan")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPhaseAdd_CreatesNumberedFile(t *testing.T) {
	dir := t.TempDir()
	plansDir := filepath.Join(dir, "plans")
	planDir := filepath.Join(plansDir, "20-my-plan")
	mustMkdir(t, planDir)

	writeFile(t, filepath.Join(planDir, "overview.md"),
		"---\nid: 20\ncreated: 2026-02-20\nstatus: active\n---\n\n# My Plan\n")
	writeFile(t, filepath.Join(planDir, "phase-01-scaffold.md"),
		"# Phase 1: Scaffold\n")

	created, err := PhaseAdd(plansDir, 20, "Implementation")
	if err != nil {
		t.Fatalf("PhaseAdd returned error: %v", err)
	}

	if !strings.Contains(created, "phase-02-implementation.md") {
		t.Errorf("unexpected filename: %s", created)
	}

	data, err := os.ReadFile(created)
	if err != nil {
		t.Fatalf("phase file not readable: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "[[plans/20-my-plan/overview]]") {
		t.Errorf("phase missing back link: %s", content)
	}
	if !strings.Contains(content, "# Phase 2: Implementation") {
		t.Errorf("phase missing heading: %s", content)
	}
}

func TestPhaseAdd_FirstPhase(t *testing.T) {
	dir := t.TempDir()
	plansDir := filepath.Join(dir, "plans")
	planDir := filepath.Join(plansDir, "30-empty-plan")
	mustMkdir(t, planDir)

	writeFile(t, filepath.Join(planDir, "overview.md"),
		"---\nid: 30\ncreated: 2026-02-20\nstatus: draft\n---\n\n# Empty Plan\n")

	created, err := PhaseAdd(plansDir, 30, "Setup")
	if err != nil {
		t.Fatalf("PhaseAdd returned error: %v", err)
	}

	if !strings.Contains(created, "phase-01-setup.md") {
		t.Errorf("unexpected filename: %s", created)
	}
}

func TestPhaseAdd_MultiWordName(t *testing.T) {
	dir := t.TempDir()
	plansDir := filepath.Join(dir, "plans")
	planDir := filepath.Join(plansDir, "40-multi")
	mustMkdir(t, planDir)

	writeFile(t, filepath.Join(planDir, "overview.md"),
		"---\nid: 40\ncreated: 2026-02-20\nstatus: active\n---\n\n# Multi\n")

	created, err := PhaseAdd(plansDir, 40, "Native Planning")
	if err != nil {
		t.Fatalf("PhaseAdd returned error: %v", err)
	}

	if !strings.Contains(created, "phase-01-native-planning.md") {
		t.Errorf("unexpected filename: %s", created)
	}

	data, err := os.ReadFile(created)
	if err != nil {
		t.Fatalf("phase file not readable: %v", err)
	}
	if !strings.Contains(string(data), "# Phase 1: Native Planning") {
		t.Errorf("phase missing heading: %s", string(data))
	}
}

func TestPhaseAdd_ErrorsOnMissingPlan(t *testing.T) {
	dir := t.TempDir()
	plansDir := filepath.Join(dir, "plans")
	mustMkdir(t, plansDir)

	_, err := PhaseAdd(plansDir, 999, "Oops")
	if err == nil {
		t.Fatal("expected error for missing plan")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestFindPlanDir_MatchesByID(t *testing.T) {
	dir := t.TempDir()
	plansDir := filepath.Join(dir, "plans")
	mustMkdir(t, filepath.Join(plansDir, "15-some-plan"))

	found, err := findPlanDir(plansDir, 15)
	if err != nil {
		t.Fatalf("findPlanDir returned error: %v", err)
	}
	if filepath.Base(found) != "15-some-plan" {
		t.Errorf("unexpected dir: %s", found)
	}
}

func TestFindPlanDir_ErrorsOnMissing(t *testing.T) {
	dir := t.TempDir()
	plansDir := filepath.Join(dir, "plans")
	mustMkdir(t, plansDir)

	_, err := findPlanDir(plansDir, 99)
	if err == nil {
		t.Fatal("expected error for missing plan")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestTitleFromSlug(t *testing.T) {
	tests := []struct {
		slug string
		want string
	}{
		{"test-plan", "Test Plan"},
		{"single", "Single"},
		{"multi-word-slug", "Multi Word Slug"},
	}
	for _, tt := range tests {
		got := titleFromSlug(tt.slug)
		if got != tt.want {
			t.Errorf("titleFromSlug(%q) = %q, want %q", tt.slug, got, tt.want)
		}
	}
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"Implementation", "implementation"},
		{"Native Planning", "native-planning"},
		{"Phase  With   Spaces", "phase-with-spaces"},
		{"Special!@#Characters", "special-characters"},
	}
	for _, tt := range tests {
		got := slugify(tt.name)
		if got != tt.want {
			t.Errorf("slugify(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}
