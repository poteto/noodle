package loop

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
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

func TestAuditQueueAllValid(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	queuePath := filepath.Join(runtimeDir, "queue.json")

	queue := Queue{
		GeneratedAt: time.Now().UTC(),
		Items: []QueueItem{
			{ID: "execute-1", TaskKey: "execute"},
			{ID: "schedule-1", TaskKey: "schedule"},
		},
	}
	if err := writeQueueAtomic(queuePath, queue); err != nil {
		t.Fatalf("write queue: %v", err)
	}

	l := &Loop{
		projectDir: projectDir,
		runtimeDir: runtimeDir,
		registry:   testLoopRegistry(),
		deps: Dependencies{
			QueueFile: queuePath,
			Now:       time.Now,
		},
	}

	dropped := l.auditQueue()
	if len(dropped) != 0 {
		t.Fatalf("expected 0 dropped items, got %d", len(dropped))
	}

	// Verify queue is unchanged.
	after, err := readQueue(queuePath)
	if err != nil {
		t.Fatalf("read queue after audit: %v", err)
	}
	if len(after.Items) != 2 {
		t.Fatalf("expected 2 items after audit, got %d", len(after.Items))
	}
}

func TestAuditQueueDropsNonexistentSkill(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	queuePath := filepath.Join(runtimeDir, "queue.json")

	queue := Queue{
		GeneratedAt: time.Now().UTC(),
		Items: []QueueItem{
			{ID: "execute-1", TaskKey: "execute"},
			{ID: "deploy-1", TaskKey: "deploy"},
			{ID: "schedule-1", TaskKey: "schedule"},
		},
	}
	if err := writeQueueAtomic(queuePath, queue); err != nil {
		t.Fatalf("write queue: %v", err)
	}

	l := &Loop{
		projectDir: projectDir,
		runtimeDir: runtimeDir,
		registry:   testLoopRegistry(), // has execute, schedule, reflect, meditate, oops, review — no deploy
		deps: Dependencies{
			QueueFile: queuePath,
			Now:       time.Now,
		},
	}

	dropped := l.auditQueue()
	if len(dropped) != 1 {
		t.Fatalf("expected 1 dropped item, got %d", len(dropped))
	}
	if dropped[0].ID != "deploy-1" {
		t.Fatalf("expected dropped item deploy-1, got %q", dropped[0].ID)
	}

	// Verify queue was rewritten without the dropped item.
	after, err := readQueue(queuePath)
	if err != nil {
		t.Fatalf("read queue after audit: %v", err)
	}
	if len(after.Items) != 2 {
		t.Fatalf("expected 2 items after audit, got %d", len(after.Items))
	}
	for _, item := range after.Items {
		if item.TaskKey == "deploy" {
			t.Fatal("deploy item should have been removed")
		}
	}

	// Verify event was written.
	eventsPath := filepath.Join(runtimeDir, "queue-events.ndjson")
	data, err := os.ReadFile(eventsPath)
	if err != nil {
		t.Fatalf("read events: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) < 1 {
		t.Fatal("expected at least one event line")
	}
	var event QueueAuditEvent
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &event); err != nil {
		t.Fatalf("unmarshal event: %v", err)
	}
	if event.Type != "queue_drop" {
		t.Fatalf("event type = %q, want queue_drop", event.Type)
	}
	if event.Target != "deploy-1" {
		t.Fatalf("event target = %q, want deploy-1", event.Target)
	}
	if event.Skill != "deploy" {
		t.Fatalf("event skill = %q, want deploy", event.Skill)
	}
}

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
			QueueFile:  filepath.Join(runtimeDir, "queue.json"),
			StatusFile: filepath.Join(runtimeDir, "status.json"),
		},
		state:          StateRunning,
		activeByTarget: map[string]*activeCook{},
		activeByID:     map[string]*activeCook{},
		adoptedTargets: map[string]string{},
		failedTargets:  map[string]string{},
		pendingReview:  map[string]*pendingReviewCook{},
		pendingRetry:   map[string]*pendingRetryCook{},
		processedIDs:   map[string]struct{}{},
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
			Mise:      &fakeMise{},
			Now:       time.Now,
			QueueFile: filepath.Join(runtimeDir, "queue.json"),
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

func TestPrepareQueueRescanRecoversMissingSkill(t *testing.T) {
	// Simulate: queue has an item with a task type that only appears
	// after a registry rebuild (e.g., skill was just installed and
	// fsnotify hasn't fired yet).
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	queuePath := filepath.Join(runtimeDir, "queue.json")

	queue := Queue{
		GeneratedAt: time.Now().UTC(),
		Items: []QueueItem{
			{ID: "deploy-1", TaskKey: "deploy", Skill: "deploy", Provider: "claude", Model: "opus"},
		},
	}
	if err := writeQueueAtomic(queuePath, queue); err != nil {
		t.Fatalf("write queue: %v", err)
	}

	// Start with a registry that does NOT have "deploy".
	initialReg := testLoopRegistry()

	// Create the deploy skill on disk so rebuildRegistry() will find it.
	skillDir := filepath.Join(homeDir, ".noodle", "skills", "deploy")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir skill: %v", err)
	}
	content := "---\nname: deploy\ndescription: Deploy\nnoodle:\n  schedule: \"After execute\"\n---\n# Deploy\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.Skills.Paths = []string{"~/.noodle/skills"}

	l := &Loop{
		projectDir: projectDir,
		runtimeDir: runtimeDir,
		config:     cfg,
		registry:   initialReg,
		deps: Dependencies{
			Dispatcher: &fakeDispatcher{},
			Worktree:   &fakeWorktree{},
			Adapter:    &fakeAdapterRunner{},
			Mise:       &fakeMise{},
			Monitor:    fakeMonitor{},
			Now:        time.Now,
			QueueFile:  queuePath,
			StatusFile: filepath.Join(runtimeDir, "status.json"),
		},
		state:          StateRunning,
		activeByTarget: map[string]*activeCook{},
		activeByID:     map[string]*activeCook{},
		adoptedTargets: map[string]string{},
		failedTargets:  map[string]string{},
		pendingReview:  map[string]*pendingReviewCook{},
		pendingRetry:   map[string]*pendingRetryCook{},
		processedIDs:   map[string]struct{}{},
	}

	brief := mise.Brief{NeedsScheduling: nil}
	result, shouldContinue, err := l.prepareQueueForCycle(brief, nil)
	if err != nil {
		t.Fatalf("prepareQueueForCycle: %v", err)
	}
	if !shouldContinue {
		t.Fatal("expected cycle to continue")
	}
	// The deploy item should still be in the queue — registry was rebuilt.
	found := false
	for _, item := range result.Items {
		if item.ID == "deploy-1" {
			found = true
		}
	}
	if !found {
		t.Fatalf("deploy-1 should be in queue after re-scan, got: %+v", result.Items)
	}
	// Verify the registry now has deploy.
	if _, ok := l.registry.ByKey("deploy"); !ok {
		t.Fatal("registry should contain deploy after rebuild")
	}
}

func TestPrepareQueueRescanDropsGenuinelyUnknown(t *testing.T) {
	// Simulate: queue has an item with a task type that doesn't exist
	// even after rebuild — should be dropped via auditQueue.
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	queuePath := filepath.Join(runtimeDir, "queue.json")

	queue := Queue{
		GeneratedAt: time.Now().UTC(),
		Items: []QueueItem{
			{ID: "execute-1", TaskKey: "execute", Skill: "execute", Provider: "claude", Model: "opus"},
			{ID: "bogus-1", TaskKey: "bogus", Skill: "bogus", Provider: "claude", Model: "opus"},
		},
	}
	if err := writeQueueAtomic(queuePath, queue); err != nil {
		t.Fatalf("write queue: %v", err)
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
		deps: Dependencies{
			Dispatcher: &fakeDispatcher{},
			Worktree:   &fakeWorktree{},
			Adapter:    &fakeAdapterRunner{},
			Mise:       &fakeMise{},
			Monitor:    fakeMonitor{},
			Now:        time.Now,
			QueueFile:  queuePath,
			StatusFile: filepath.Join(runtimeDir, "status.json"),
		},
		state:          StateRunning,
		activeByTarget: map[string]*activeCook{},
		activeByID:     map[string]*activeCook{},
		adoptedTargets: map[string]string{},
		failedTargets:  map[string]string{},
		pendingReview:  map[string]*pendingReviewCook{},
		pendingRetry:   map[string]*pendingRetryCook{},
		processedIDs:   map[string]struct{}{},
	}

	brief := mise.Brief{NeedsScheduling: nil}
	result, shouldContinue, err := l.prepareQueueForCycle(brief, nil)
	if err != nil {
		t.Fatalf("prepareQueueForCycle: %v", err)
	}
	if !shouldContinue {
		t.Fatal("expected cycle to continue after dropping unknown items")
	}
	// bogus-1 should be gone, execute-1 should remain.
	for _, item := range result.Items {
		if item.ID == "bogus-1" {
			t.Fatal("bogus-1 should have been dropped")
		}
	}
	// Verify execute-1 survived.
	found := false
	for _, item := range result.Items {
		if item.ID == "execute-1" {
			found = true
		}
	}
	if !found {
		t.Fatalf("execute-1 should still be in queue, got: %+v", result.Items)
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
			Mise:      &fakeMise{},
			Now:       time.Now,
			QueueFile: filepath.Join(runtimeDir, "queue.json"),
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
			Mise:      &fakeMise{},
			Now:       time.Now,
			QueueFile: filepath.Join(runtimeDir, "queue.json"),
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
