package loop

import (
	"bufio"
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/internal/taskreg"
	"github.com/poteto/noodle/mise"
	"github.com/poteto/noodle/skill"
)

func TestQueueEventsFileTruncation(t *testing.T) {
	dir := t.TempDir()
	eventsPath := filepath.Join(dir, "queue-events.ndjson")

	// Write 300 lines.
	now := time.Now().UTC()
	for i := 0; i < 300; i++ {
		event := QueueAuditEvent{
			At:     now,
			Type:   "queue_drop",
			Target: "item",
			Reason: "test",
		}
		appendQueueEvent(eventsPath, event)
	}

	// Verify truncated to 200 lines.
	f, err := os.Open(eventsPath)
	if err != nil {
		t.Fatalf("open events file: %v", err)
	}
	defer f.Close()

	var lineCount int
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) != "" {
			lineCount++
		}
	}
	if lineCount != 200 {
		t.Fatalf("expected 200 lines after truncation, got %d", lineCount)
	}
}

func TestRegistryErrorResilience(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	cfg := config.DefaultConfig()
	registryErr := errors.New("task type discovery failed: network error")

	l := &Loop{
		projectDir:     projectDir,
		runtimeDir:     runtimeDir,
		config:         cfg,
		registry:       taskreg.NewFromSkills(nil),
		registryErr:    registryErr,
		deps: Dependencies{
			Dispatcher: &fakeDispatcher{},
			Worktree:   &fakeWorktree{},
			Adapter:    &fakeAdapterRunner{},
			Mise:       &fakeMise{},
			Monitor:    fakeMonitor{},
			Registry:   taskreg.NewFromSkills(nil),
			Now:        time.Now,
			StatusFile: filepath.Join(runtimeDir, "status.json"),
		},
		state:          StateRunning,
		activeCooksByOrder: map[string]*cookHandle{},
		adoptedTargets:     map[string]string{},
		failedTargets:      map[string]string{},
		pendingReview:      map[string]*pendingReviewCook{},
		pendingRetry:       map[string]*pendingRetryCook{},
		processedIDs:       map[string]struct{}{},
	}

	// First failure: skips cycle, no error.
	ready, err := l.runCycleMaintenance(context.Background())
	if err != nil {
		t.Fatalf("first failure should not return error, got: %v", err)
	}
	if ready {
		t.Fatal("first failure should not be ready")
	}
	if l.registryFailCount != 1 {
		t.Fatalf("registryFailCount = %d, want 1", l.registryFailCount)
	}

	// Second failure: skips cycle, no error.
	ready, err = l.runCycleMaintenance(context.Background())
	if err != nil {
		t.Fatalf("second failure should not return error, got: %v", err)
	}
	if ready {
		t.Fatal("second failure should not be ready")
	}
	if l.registryFailCount != 2 {
		t.Fatalf("registryFailCount = %d, want 2", l.registryFailCount)
	}

	// Third failure: returns fatal error.
	ready, err = l.runCycleMaintenance(context.Background())
	if err == nil {
		t.Fatal("third failure should return fatal error")
	}
	if !strings.Contains(err.Error(), "task type discovery failed") {
		t.Fatalf("expected registry error, got: %v", err)
	}
}

func TestRegistryErrorResetOnRebuild(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Create a valid skill so discoverRegistry succeeds.
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	skillDir := filepath.Join(homeDir, ".noodle", "skills", "execute")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("create skill dir: %v", err)
	}
	content := "---\nname: execute\ndescription: Execute tasks\nnoodle:\n  schedule: \"When ready\"\n---\n# Execute\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.Skills.Paths = []string{"~/.noodle/skills"}

	l := &Loop{
		projectDir:        projectDir,
		runtimeDir:        runtimeDir,
		config:            cfg,
		registry:          taskreg.NewFromSkills(nil),
		registryErr:       errors.New("transient error"),
		registryFailCount: 2,
		deps: Dependencies{
			Mise: &fakeMise{},
			Now:  time.Now,
		},
	}

	l.rebuildRegistry()

	if l.registryErr != nil {
		t.Fatalf("registryErr should be nil after rebuild, got: %v", l.registryErr)
	}
	if l.registryFailCount != 0 {
		t.Fatalf("registryFailCount should be 0 after rebuild, got: %d", l.registryFailCount)
	}
}

func TestPrepareOrdersRescanRecoversMissingSkill(t *testing.T) {
	// Simulate: orders has stages with both known and unknown task types.
	// The unknown stage should be silently dropped during normalization,
	// but a known stage alongside it should survive.
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	ordersPath := filepath.Join(runtimeDir, "orders.json")

	// Include both a known (execute) and unknown (deploy) order.
	orders := OrdersFile{
		GeneratedAt: time.Now().UTC(),
		Orders: []Order{
			{ID: "execute-1", Title: "execute task", Status: OrderStatusActive, Stages: []Stage{
				{TaskKey: "execute", Skill: "execute", Provider: "claude", Model: "opus", Status: StageStatusPending},
			}},
			{ID: "deploy-1", Title: "deploy task", Status: OrderStatusActive, Stages: []Stage{
				{TaskKey: "deploy", Skill: "deploy", Provider: "claude", Model: "opus", Status: StageStatusPending},
			}},
		},
	}
	if err := writeOrdersAtomic(ordersPath, orders); err != nil {
		t.Fatalf("write orders: %v", err)
	}

	cfg := config.DefaultConfig()

	l := &Loop{
		projectDir: projectDir,
		runtimeDir: runtimeDir,
		config:     cfg,
		registry:   testLoopRegistry(),
		logger:     slog.New(slog.NewTextHandler(os.Stderr, nil)),
		deps: Dependencies{
			Dispatcher:     &fakeDispatcher{},
			Worktree:       &fakeWorktree{},
			Adapter:        &fakeAdapterRunner{},
			Mise:           &fakeMise{},
			Monitor:        fakeMonitor{},
			Now:            time.Now,
			OrdersFile:     ordersPath,
			OrdersNextFile: filepath.Join(runtimeDir, "orders-next.json"),
			StatusFile:     filepath.Join(runtimeDir, "status.json"),
		},
		state:              StateRunning,
		activeCooksByOrder: map[string]*cookHandle{},
		adoptedTargets:     map[string]string{},
		failedTargets:      map[string]string{},
		pendingReview:      map[string]*pendingReviewCook{},
		pendingRetry:       map[string]*pendingRetryCook{},
		processedIDs:       map[string]struct{}{},
	}

	brief := mise.Brief{}
	result, shouldContinue, err := l.prepareOrdersForCycle(brief, nil)
	if err != nil {
		t.Fatalf("prepareOrdersForCycle: %v", err)
	}
	if !shouldContinue {
		t.Fatal("expected cycle to continue")
	}
	// The unknown deploy order should have been dropped.
	for _, order := range result.Orders {
		if order.ID == "deploy-1" {
			t.Fatal("deploy-1 should have been dropped (unknown task type)")
		}
	}
	// The known execute order should survive.
	found := false
	for _, order := range result.Orders {
		if order.ID == "execute-1" {
			found = true
		}
	}
	if !found {
		t.Fatalf("execute-1 should still be in orders, got: %+v", result.Orders)
	}
}

func TestPrepareOrdersRescanDropsGenuinelyUnknown(t *testing.T) {
	// Simulate: orders has a stage with a task type that doesn't exist
	// even after rebuild — should be dropped via auditOrders.
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	ordersPath := filepath.Join(runtimeDir, "orders.json")

	orders := OrdersFile{
		GeneratedAt: time.Now().UTC(),
		Orders: []Order{
			{ID: "execute-1", Title: "execute task", Status: OrderStatusActive, Stages: []Stage{
				{TaskKey: "execute", Skill: "execute", Provider: "claude", Model: "opus", Status: StageStatusPending},
			}},
			{ID: "bogus-1", Title: "bogus task", Status: OrderStatusActive, Stages: []Stage{
				{TaskKey: "bogus", Skill: "bogus", Provider: "claude", Model: "opus", Status: StageStatusPending},
			}},
		},
	}
	if err := writeOrdersAtomic(ordersPath, orders); err != nil {
		t.Fatalf("write orders: %v", err)
	}

	// Create execute skill on disk so rebuild finds it.
	for _, name := range []string{"execute", "schedule"} {
		skillDir := filepath.Join(homeDir, ".noodle", "skills", name)
		if err := os.MkdirAll(skillDir, 0o755); err != nil {
			t.Fatalf("mkdir skill: %v", err)
		}
		content := "---\nname: " + name + "\ndescription: " + name + "\nnoodle:\n  schedule: \"x\"\n---\n# " + name + "\n"
		if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
			t.Fatalf("write SKILL.md: %v", err)
		}
	}

	cfg := config.DefaultConfig()
	cfg.Skills.Paths = []string{"~/.noodle/skills"}

	l := &Loop{
		projectDir: projectDir,
		runtimeDir: runtimeDir,
		config:     cfg,
		registry:   testLoopRegistry(),
		logger:     slog.New(slog.NewTextHandler(os.Stderr, nil)),
		deps: Dependencies{
			Dispatcher:     &fakeDispatcher{},
			Worktree:       &fakeWorktree{},
			Adapter:        &fakeAdapterRunner{},
			Mise:           &fakeMise{},
			Monitor:        fakeMonitor{},
			Now:            time.Now,
			OrdersFile:     ordersPath,
			OrdersNextFile: filepath.Join(runtimeDir, "orders-next.json"),
			StatusFile:     filepath.Join(runtimeDir, "status.json"),
		},
		state:              StateRunning,
		activeCooksByOrder: map[string]*cookHandle{},
		adoptedTargets:     map[string]string{},
		failedTargets:      map[string]string{},
		pendingReview:      map[string]*pendingReviewCook{},
		pendingRetry:       map[string]*pendingRetryCook{},
		processedIDs:       map[string]struct{}{},
	}

	brief := mise.Brief{}
	result, shouldContinue, err := l.prepareOrdersForCycle(brief, nil)
	if err != nil {
		t.Fatalf("prepareOrdersForCycle: %v", err)
	}
	if !shouldContinue {
		t.Fatal("expected cycle to continue after dropping unknown orders")
	}
	// bogus-1 should be gone, execute-1 should remain.
	for _, order := range result.Orders {
		if order.ID == "bogus-1" {
			t.Fatal("bogus-1 should have been dropped")
		}
	}
	// Verify execute-1 survived.
	found := false
	for _, order := range result.Orders {
		if order.ID == "execute-1" {
			found = true
		}
	}
	if !found {
		t.Fatalf("execute-1 should still be in orders, got: %+v", result.Orders)
	}
}

func TestEnsureSkillFreshExistingSkill(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	l := &Loop{
		projectDir: projectDir,
		runtimeDir: runtimeDir,
		config:     config.DefaultConfig(),
		registry:   testLoopRegistry(),
		deps: Dependencies{
			Mise: &fakeMise{},
			Now:  time.Now,
		},
	}

	if !l.ensureSkillFresh("execute") {
		t.Fatal("ensureSkillFresh should return true for existing skill")
	}
}

func TestEnsureSkillFreshMissingSkill(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.Skills.Paths = []string{"~/.noodle/skills"}

	l := &Loop{
		projectDir: projectDir,
		runtimeDir: runtimeDir,
		config:     cfg,
		registry:   testLoopRegistry(),
		deps: Dependencies{
			Mise: &fakeMise{},
			Now:  time.Now,
		},
	}

	// "nonexistent" is not in the registry and not on disk.
	if l.ensureSkillFresh("nonexistent") {
		t.Fatal("ensureSkillFresh should return false for missing skill")
	}
}

func TestDiffRegistryKeys(t *testing.T) {
	old := taskreg.NewFromSkills([]skill.SkillMeta{
		{Name: "execute", Path: "/skills/execute", Frontmatter: skill.Frontmatter{Noodle: &skill.NoodleMeta{Schedule: "x"}}},
		{Name: "deploy", Path: "/skills/deploy", Frontmatter: skill.Frontmatter{Noodle: &skill.NoodleMeta{Schedule: "x"}}},
	})
	new := taskreg.NewFromSkills([]skill.SkillMeta{
		{Name: "execute", Path: "/skills/execute", Frontmatter: skill.Frontmatter{Noodle: &skill.NoodleMeta{Schedule: "x"}}},
		{Name: "staging", Path: "/skills/staging", Frontmatter: skill.Frontmatter{Noodle: &skill.NoodleMeta{Schedule: "x"}}},
	})

	diff := diffRegistryKeys(old, new)

	if len(diff.Added) != 1 || diff.Added[0] != "staging" {
		t.Fatalf("Added = %v, want [staging]", diff.Added)
	}
	if len(diff.Removed) != 1 || diff.Removed[0] != "deploy" {
		t.Fatalf("Removed = %v, want [deploy]", diff.Removed)
	}
}
