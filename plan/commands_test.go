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
	if !strings.Contains(overviewStr, "status: ready") {
		t.Errorf("overview missing status: %s", overviewStr)
	}
	if !strings.Contains(overviewStr, "# Test Plan") {
		t.Errorf("overview missing title: %s", overviewStr)
	}
	if !strings.Contains(overviewStr, "created:") {
		t.Errorf("overview missing created date: %s", overviewStr)
	}

	// Verify no phase-01-scaffold.md created (phases added separately via PhaseAdd).
	if _, err := os.Stat(filepath.Join(created, "phase-01-scaffold.md")); err == nil {
		t.Error("phase-01-scaffold.md should not be created by Create")
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

	writeFile(t, filepath.Join(plansDir, "index.md"),
		"# Plans\n\n- [ ] [[plans/10-my-plan/overview]]\n")
	writeFile(t, filepath.Join(planDir, "overview.md"),
		"---\nid: 10\ncreated: 2026-02-20\nstatus: ready\n---\n\n# My Plan\n")

	if err := Done(plansDir, 10, "keep"); err != nil {
		t.Fatalf("Done returned error: %v", err)
	}

	archivedOverview := filepath.Join(dir, "archived_plans", "10-my-plan", "overview.md")
	data, err := os.ReadFile(archivedOverview)
	if err != nil {
		t.Fatalf("archived overview.md not readable: %v", err)
	}
	if !strings.Contains(string(data), "status: done") {
		t.Errorf("status not changed to done: %s", string(data))
	}
	if strings.Contains(string(data), "status: ready") {
		t.Errorf("draft status still present: %s", string(data))
	}
}

func TestDone_FlipsStatusFromActive(t *testing.T) {
	dir := t.TempDir()
	plansDir := filepath.Join(dir, "plans")
	planDir := filepath.Join(plansDir, "11-active-plan")
	mustMkdir(t, planDir)

	writeFile(t, filepath.Join(plansDir, "index.md"),
		"# Plans\n\n- [ ] [[plans/11-active-plan/overview]]\n")
	writeFile(t, filepath.Join(planDir, "overview.md"),
		"---\nid: 11\ncreated: 2026-02-20\nstatus: active\n---\n\n# Active Plan\n")

	if err := Done(plansDir, 11, "keep"); err != nil {
		t.Fatalf("Done returned error: %v", err)
	}

	archivedOverview := filepath.Join(dir, "archived_plans", "11-active-plan", "overview.md")
	data, err := os.ReadFile(archivedOverview)
	if err != nil {
		t.Fatalf("archived overview.md not readable: %v", err)
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

	err := Done(plansDir, 12, "keep")
	if err == nil {
		t.Fatal("expected error for already-done plan")
	}
	if !strings.Contains(err.Error(), "not ready or active") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDone_ErrorsOnMissingPlan(t *testing.T) {
	dir := t.TempDir()
	plansDir := filepath.Join(dir, "plans")
	mustMkdir(t, plansDir)

	err := Done(plansDir, 999, "keep")
	if err == nil {
		t.Fatal("expected error for missing plan")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDone_ArchivesPlanDirectory(t *testing.T) {
	dir := t.TempDir()
	plansDir := filepath.Join(dir, "plans")
	planDir := filepath.Join(plansDir, "5-archive-test")
	mustMkdir(t, planDir)

	writeFile(t, filepath.Join(plansDir, "index.md"),
		"# Plans\n\n- [ ] [[plans/5-archive-test/overview]]\n- [ ] [[plans/99-other/overview]]\n")
	writeFile(t, filepath.Join(planDir, "overview.md"),
		"---\nid: 5\ncreated: 2026-02-20\nstatus: active\n---\n\n# Archive Test\n")

	if err := Done(plansDir, 5, "keep"); err != nil {
		t.Fatalf("Done returned error: %v", err)
	}

	// Original directory should be gone.
	if _, err := os.Stat(planDir); !os.IsNotExist(err) {
		t.Error("original plan directory still exists")
	}

	// Archived directory should exist.
	archivedDir := filepath.Join(dir, "archived_plans", "5-archive-test")
	if _, err := os.Stat(archivedDir); err != nil {
		t.Fatalf("archived directory not found: %v", err)
	}

	// Plans index should still have other plan but not this one.
	index, _ := os.ReadFile(filepath.Join(plansDir, "index.md"))
	if strings.Contains(string(index), "5-archive-test") {
		t.Error("plans/index.md still contains archived plan")
	}
	if !strings.Contains(string(index), "99-other") {
		t.Error("plans/index.md lost unrelated plan")
	}

	// Archived index should have the plan.
	archivedIndex, _ := os.ReadFile(filepath.Join(dir, "archived_plans", "index.md"))
	if !strings.Contains(string(archivedIndex), "[[archived_plans/5-archive-test/overview]]") {
		t.Errorf("archived index missing wikilink: %s", string(archivedIndex))
	}
}

func TestDone_CreatesArchivedIndexIfMissing(t *testing.T) {
	dir := t.TempDir()
	plansDir := filepath.Join(dir, "plans")
	planDir := filepath.Join(plansDir, "6-first-archive")
	mustMkdir(t, planDir)

	writeFile(t, filepath.Join(plansDir, "index.md"),
		"# Plans\n\n- [ ] [[plans/6-first-archive/overview]]\n")
	writeFile(t, filepath.Join(planDir, "overview.md"),
		"---\nid: 6\ncreated: 2026-02-20\nstatus: ready\n---\n\n# First Archive\n")

	// Verify no archived_plans/ exists yet.
	archivedIndexPath := filepath.Join(dir, "archived_plans", "index.md")
	if _, err := os.Stat(archivedIndexPath); !os.IsNotExist(err) {
		t.Fatal("archived index.md already exists before test")
	}

	if err := Done(plansDir, 6, "keep"); err != nil {
		t.Fatalf("Done returned error: %v", err)
	}

	data, err := os.ReadFile(archivedIndexPath)
	if err != nil {
		t.Fatalf("archived index.md not created: %v", err)
	}
	if !strings.Contains(string(data), "# Archived Plans") {
		t.Errorf("archived index missing header: %s", string(data))
	}
	if !strings.Contains(string(data), "[[archived_plans/6-first-archive/overview]]") {
		t.Errorf("archived index missing wikilink: %s", string(data))
	}
}

func TestDone_UpdatesInternalWikilinks(t *testing.T) {
	dir := t.TempDir()
	plansDir := filepath.Join(dir, "plans")
	planDir := filepath.Join(plansDir, "7-link-test")
	mustMkdir(t, planDir)

	writeFile(t, filepath.Join(plansDir, "index.md"),
		"# Plans\n\n- [ ] [[plans/7-link-test/overview]]\n")
	writeFile(t, filepath.Join(planDir, "overview.md"),
		"---\nid: 7\ncreated: 2026-02-20\nstatus: active\n---\n\n# Link Test\n")
	writeFile(t, filepath.Join(planDir, "phase-01-setup.md"),
		"Back to [[plans/7-link-test/overview]]\n\n# Phase 1: Setup\n")
	writeFile(t, filepath.Join(planDir, "phase-02-impl.md"),
		"Back to [[plans/7-link-test/overview]]\n\n# Phase 2: Impl\n")

	if err := Done(plansDir, 7, "keep"); err != nil {
		t.Fatalf("Done returned error: %v", err)
	}

	archivedDir := filepath.Join(dir, "archived_plans", "7-link-test")

	phase1, _ := os.ReadFile(filepath.Join(archivedDir, "phase-01-setup.md"))
	if strings.Contains(string(phase1), "[[plans/7-link-test") {
		t.Errorf("phase-01 still has old wikilink: %s", string(phase1))
	}
	if !strings.Contains(string(phase1), "[[archived_plans/7-link-test/overview]]") {
		t.Errorf("phase-01 missing new wikilink: %s", string(phase1))
	}

	phase2, _ := os.ReadFile(filepath.Join(archivedDir, "phase-02-impl.md"))
	if !strings.Contains(string(phase2), "[[archived_plans/7-link-test/overview]]") {
		t.Errorf("phase-02 missing new wikilink: %s", string(phase2))
	}
}

func TestDone_UpdatesTodoLinks(t *testing.T) {
	dir := t.TempDir()
	plansDir := filepath.Join(dir, "plans")
	planDir := filepath.Join(plansDir, "8-todo-test")
	mustMkdir(t, planDir)

	writeFile(t, filepath.Join(plansDir, "index.md"),
		"# Plans\n\n- [ ] [[plans/8-todo-test/overview]]\n")
	writeFile(t, filepath.Join(planDir, "overview.md"),
		"---\nid: 8\ncreated: 2026-02-20\nstatus: active\n---\n\n# Todo Test\n")
	writeFile(t, filepath.Join(dir, "todos.md"),
		"# Todos\n\n1. [x] Some task [[plans/8-todo-test/overview]]\n2. [ ] Other task\n")

	if err := Done(plansDir, 8, "keep"); err != nil {
		t.Fatalf("Done returned error: %v", err)
	}

	todos, _ := os.ReadFile(filepath.Join(dir, "todos.md"))
	if strings.Contains(string(todos), "[[plans/8-todo-test") {
		t.Errorf("todos.md still has old wikilink: %s", string(todos))
	}
	if !strings.Contains(string(todos), "[[archived_plans/8-todo-test/overview]]") {
		t.Errorf("todos.md missing new wikilink: %s", string(todos))
	}
	if !strings.Contains(string(todos), "Other task") {
		t.Error("todos.md lost unrelated content")
	}
}

func TestDone_RemovesFromPlansIndex(t *testing.T) {
	dir := t.TempDir()
	plansDir := filepath.Join(dir, "plans")
	planDir := filepath.Join(plansDir, "9-index-test")
	mustMkdir(t, planDir)

	writeFile(t, filepath.Join(plansDir, "index.md"),
		"# Plans\n\n- [ ] [[plans/9-index-test/overview]]\n- [ ] [[plans/50-keep-me/overview]]\n")
	writeFile(t, filepath.Join(planDir, "overview.md"),
		"---\nid: 9\ncreated: 2026-02-20\nstatus: ready\n---\n\n# Index Test\n")

	if err := Done(plansDir, 9, "keep"); err != nil {
		t.Fatalf("Done returned error: %v", err)
	}

	index, _ := os.ReadFile(filepath.Join(plansDir, "index.md"))
	if strings.Contains(string(index), "9-index-test") {
		t.Errorf("plans/index.md still contains removed plan: %s", string(index))
	}
	if !strings.Contains(string(index), "50-keep-me") {
		t.Errorf("plans/index.md lost unrelated plan: %s", string(index))
	}
}

func TestPhaseAdd_CreatesNumberedFile(t *testing.T) {
	dir := t.TempDir()
	plansDir := filepath.Join(dir, "plans")
	planDir := filepath.Join(plansDir, "20-my-plan")
	mustMkdir(t, planDir)

	writeFile(t, filepath.Join(planDir, "overview.md"),
		"---\nid: 20\ncreated: 2026-02-20\nstatus: active\n---\n\n# My Plan\n")
	writeFile(t, filepath.Join(planDir, "phase-01-research.md"),
		"# Phase 1: Research\n")

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
		"---\nid: 30\ncreated: 2026-02-20\nstatus: ready\n---\n\n# Empty Plan\n")

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

func TestDone_RemoveDeletesPlanDirectory(t *testing.T) {
	dir := t.TempDir()
	plansDir := filepath.Join(dir, "plans")
	planDir := filepath.Join(plansDir, "20-remove-test")
	mustMkdir(t, planDir)

	writeFile(t, filepath.Join(plansDir, "index.md"),
		"# Plans\n\n- [ ] [[plans/20-remove-test/overview]]\n")
	writeFile(t, filepath.Join(planDir, "overview.md"),
		"---\nid: 20\ncreated: 2026-02-20\nstatus: ready\n---\n\n# Remove Test\n")

	if err := Done(plansDir, 20, "remove"); err != nil {
		t.Fatalf("Done returned error: %v", err)
	}

	// Plan directory should be gone.
	if _, err := os.Stat(planDir); !os.IsNotExist(err) {
		t.Error("plan directory still exists after remove")
	}

	// No archived directory should be created.
	archivedDir := filepath.Join(dir, "archived_plans", "20-remove-test")
	if _, err := os.Stat(archivedDir); !os.IsNotExist(err) {
		t.Error("archived directory should not exist after remove")
	}
}

func TestDone_RemoveUpdatesPlansIndex(t *testing.T) {
	dir := t.TempDir()
	plansDir := filepath.Join(dir, "plans")
	planDir := filepath.Join(plansDir, "21-remove-index")
	mustMkdir(t, planDir)

	writeFile(t, filepath.Join(plansDir, "index.md"),
		"# Plans\n\n- [ ] [[plans/21-remove-index/overview]]\n- [ ] [[plans/99-other/overview]]\n")
	writeFile(t, filepath.Join(planDir, "overview.md"),
		"---\nid: 21\ncreated: 2026-02-20\nstatus: active\n---\n\n# Remove Index\n")

	if err := Done(plansDir, 21, "remove"); err != nil {
		t.Fatalf("Done returned error: %v", err)
	}

	index, _ := os.ReadFile(filepath.Join(plansDir, "index.md"))
	if strings.Contains(string(index), "21-remove-index") {
		t.Error("plans/index.md still contains removed plan")
	}
	if !strings.Contains(string(index), "99-other") {
		t.Error("plans/index.md lost unrelated plan")
	}
}

func TestDone_RemoveUpdatesTodoLinks(t *testing.T) {
	dir := t.TempDir()
	plansDir := filepath.Join(dir, "plans")
	planDir := filepath.Join(plansDir, "22-remove-todo")
	mustMkdir(t, planDir)

	writeFile(t, filepath.Join(plansDir, "index.md"),
		"# Plans\n\n- [ ] [[plans/22-remove-todo/overview]]\n")
	writeFile(t, filepath.Join(planDir, "overview.md"),
		"---\nid: 22\ncreated: 2026-02-20\nstatus: active\n---\n\n# Remove Todo\n")
	writeFile(t, filepath.Join(dir, "todos.md"),
		"# Todos\n\n1. [x] Task with plan [[plans/22-remove-todo/overview]]\n2. [ ] Other task\n")

	if err := Done(plansDir, 22, "remove"); err != nil {
		t.Fatalf("Done returned error: %v", err)
	}

	todos, _ := os.ReadFile(filepath.Join(dir, "todos.md"))
	if strings.Contains(string(todos), "22-remove-todo") {
		t.Error("todos.md still contains removed plan reference")
	}
	if !strings.Contains(string(todos), "Other task") {
		t.Error("todos.md lost unrelated content")
	}
}

func TestActivate_FlipsReadyToActive(t *testing.T) {
	dir := t.TempDir()
	plansDir := filepath.Join(dir, "plans")
	planDir := filepath.Join(plansDir, "30-activate-test")
	mustMkdir(t, planDir)

	writeFile(t, filepath.Join(planDir, "overview.md"),
		"---\nid: 30\ncreated: 2026-02-20\nstatus: ready\n---\n\n# Activate Test\n")

	if err := Activate(plansDir, 30); err != nil {
		t.Fatalf("Activate returned error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(planDir, "overview.md"))
	if err != nil {
		t.Fatalf("overview.md not readable: %v", err)
	}
	if !strings.Contains(string(data), "status: active") {
		t.Errorf("status not changed to active: %s", string(data))
	}
}

func TestActivate_ErrorsOnAlreadyActive(t *testing.T) {
	dir := t.TempDir()
	plansDir := filepath.Join(dir, "plans")
	planDir := filepath.Join(plansDir, "31-already-active")
	mustMkdir(t, planDir)

	writeFile(t, filepath.Join(planDir, "overview.md"),
		"---\nid: 31\ncreated: 2026-02-20\nstatus: active\n---\n\n# Already Active\n")

	err := Activate(plansDir, 31)
	if err == nil {
		t.Fatal("expected error for already-active plan")
	}
	if !strings.Contains(err.Error(), "not ready") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestActivate_ErrorsOnMissingPlan(t *testing.T) {
	dir := t.TempDir()
	plansDir := filepath.Join(dir, "plans")
	mustMkdir(t, plansDir)

	err := Activate(plansDir, 999)
	if err == nil {
		t.Fatal("expected error for missing plan")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("unexpected error: %v", err)
	}
}
