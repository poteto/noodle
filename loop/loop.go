package loop

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/internal/taskreg"
	"github.com/poteto/noodle/mise"
	"github.com/poteto/noodle/skill"
)

func New(projectDir, noodleBin string, cfg config.Config, deps Dependencies) *Loop {
	projectDir = strings.TrimSpace(projectDir)
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if deps.Dispatcher == nil || deps.Worktree == nil || deps.Adapter == nil || deps.Mise == nil || deps.Monitor == nil {
		defaults := defaultDependencies(projectDir, runtimeDir, noodleBin, cfg)
		if deps.Dispatcher == nil {
			deps.Dispatcher = defaults.Dispatcher
		}
		if deps.Worktree == nil {
			deps.Worktree = defaults.Worktree
		}
		if deps.Adapter == nil {
			deps.Adapter = defaults.Adapter
		}
		if deps.Mise == nil {
			deps.Mise = defaults.Mise
		}
		if deps.Monitor == nil {
			deps.Monitor = defaults.Monitor
		}
		if deps.Now == nil {
			deps.Now = defaults.Now
		}
		if deps.QueueFile == "" {
			deps.QueueFile = defaults.QueueFile
		}
		if deps.QueueNextFile == "" {
			deps.QueueNextFile = defaults.QueueNextFile
		}
		if deps.StatusFile == "" {
			deps.StatusFile = defaults.StatusFile
		}
	}
	if deps.Now == nil {
		deps.Now = time.Now
	}
	if deps.QueueFile == "" {
		deps.QueueFile = filepath.Join(runtimeDir, "queue.json")
	}
	if deps.QueueNextFile == "" {
		deps.QueueNextFile = filepath.Join(runtimeDir, "queue-next.json")
	}
	if deps.StatusFile == "" {
		deps.StatusFile = filepath.Join(runtimeDir, "status.json")
	}

	registry := deps.Registry
	var registryErr error
	if len(registry.All()) == 0 {
		registry, registryErr = discoverRegistry(projectDir, cfg)
	}
	if builder, ok := deps.Mise.(*mise.Builder); ok {
		builder.TaskTypes = registryToTaskTypeSummaries(registry)
	}

	return &Loop{
		projectDir:            projectDir,
		runtimeDir:            runtimeDir,
		config:                cfg,
		registry:              registry,
		registryErr:           registryErr,
		deps:                  deps,
		state:                 StateRunning,
		activeByTarget:        map[string]*activeCook{},
		activeByID:            map[string]*activeCook{},
		adoptedTargets:        map[string]string{},
		adoptedSessions:       []string{},
		failedTargets:         map[string]string{},
		pendingReview:         map[string]*pendingReviewCook{},
		pendingRetry:          map[string]*pendingRetryCook{},
		processedIDs:          map[string]struct{}{},
	}
}

func discoverRegistry(projectDir string, cfg config.Config) (taskreg.Registry, error) {
	resolver := skill.Resolver{SearchPaths: cfg.Skills.Paths}
	skills, err := resolver.DiscoverTaskTypes()
	if err != nil {
		return taskreg.NewFromSkills(nil), fmt.Errorf("task type discovery failed: %w", err)
	}
	return taskreg.NewFromSkills(skills), nil
}

// Shutdown kills all active agent sessions. Called during process exit.
func (l *Loop) Shutdown() {
	for _, cook := range l.activeByID {
		_ = cook.session.Kill()
	}
	// Kill adopted sessions from previous runs that are still alive.
	for _, sessionID := range l.adoptedSessions {
		name := tmuxSessionName(sessionID)
		_ = killTmuxSession(name)
	}
}

func (l *Loop) Run(ctx context.Context) error {
	if strings.TrimSpace(l.projectDir) == "" {
		return fmt.Errorf("project directory not set")
	}
	if err := os.MkdirAll(l.runtimeDir, 0o755); err != nil {
		return fmt.Errorf("create runtime directory: %w", err)
	}
	if err := l.loadFailedTargets(); err != nil {
		return err
	}
	if err := l.reconcile(ctx); err != nil {
		return err
	}
	if err := l.hydrateProcessedCommands(); err != nil {
		return err
	}
	if err := l.Cycle(ctx); err != nil {
		return err
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create fsnotify watcher: %w", err)
	}
	defer watcher.Close()
	if err := watcher.Add(l.runtimeDir); err != nil {
		return fmt.Errorf("watch runtime directory: %w", err)
	}
	plansDir := filepath.Join(l.projectDir, "brain", "plans")
	if info, err := os.Stat(plansDir); err == nil && info.IsDir() {
		if err := watcher.Add(plansDir); err != nil {
			return fmt.Errorf("watch plans directory: %w", err)
		}
	}

	ticker := time.NewTicker(l.pollInterval())
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := l.Cycle(ctx); err != nil {
				return err
			}
			if l.state == StateDraining && len(l.activeByID) == 0 {
				return nil
			}
		case ev := <-watcher.Events:
			if strings.HasSuffix(ev.Name, "queue.json") || strings.HasSuffix(ev.Name, "queue-next.json") || strings.HasSuffix(ev.Name, "control.ndjson") {
				if err := l.Cycle(ctx); err != nil {
					return err
				}
			}
			if strings.Contains(ev.Name, filepath.Join("brain", "plans")) {
				if l.state == StateIdle {
					l.state = StateRunning
				}
				if err := l.Cycle(ctx); err != nil {
					return err
				}
			}
		case err := <-watcher.Errors:
			if err != nil {
				return fmt.Errorf("watch runtime directory: %w", err)
			}
		}
	}
}

func (l *Loop) Cycle(ctx context.Context) error {
	ready, err := l.runCycleMaintenance(ctx)
	if err != nil {
		return err
	}
	if !ready {
		return l.stampStatus()
	}

	brief, warnings, running, err := l.buildCycleBrief(ctx)
	if err != nil {
		return err
	}
	if !running {
		return l.stampStatus()
	}

	queue, shouldContinue, err := l.prepareQueueForCycle(brief, warnings)
	if err != nil {
		return err
	}
	if !shouldContinue {
		return l.stampStatus()
	}

	plan := l.planCycleSpawns(queue, brief)
	if err := l.spawnPlannedItems(ctx, plan); err != nil {
		return err
	}
	return l.stampStatus()
}

func (l *Loop) runCycleMaintenance(ctx context.Context) (bool, error) {
	if l.registryErr != nil {
		return false, l.registryErr
	}
	if err := l.processControlCommands(); err != nil {
		return false, err
	}
	if err := l.collectCompleted(ctx); err != nil {
		return false, err
	}
	if _, err := l.deps.Monitor.RunOnce(ctx); err != nil {
		return false, err
	}
	if err := l.processPendingRetries(ctx); err != nil {
		return false, err
	}
	return true, nil
}

func (l *Loop) buildCycleBrief(ctx context.Context) (mise.Brief, []string, bool, error) {
	l.refreshAdoptedTargets()
	brief, warnings, err := l.deps.Mise.Build(ctx)
	if err != nil {
		return mise.Brief{}, warnings, false, err
	}
	if l.state != StateRunning && l.state != StateIdle {
		return brief, warnings, false, nil
	}
	if l.state == StateIdle {
		l.state = StateRunning
	}
	return brief, warnings, true, nil
}

func (l *Loop) prepareQueueForCycle(brief mise.Brief, warnings []string) (Queue, bool, error) {
	// Consume queue-next.json if the prioritize session wrote one.
	// The loop is the single writer of queue.json — prioritize writes
	// to queue-next.json to avoid racing with loop state stamps.
	// Errors are non-fatal: a transient/partial write shouldn't crash
	// the loop — log and continue.
	if err := consumeQueueNext(l.deps.QueueNextFile, l.deps.QueueFile); err != nil {
		fmt.Fprintf(os.Stderr, "loop.queue-next: %v\n", err)
	}
	queue, err := readQueue(l.deps.QueueFile)
	if err != nil {
		return Queue{}, false, err
	}
	if normalizedQueue, changed, err := normalizeAndValidateQueue(queue, brief.NeedsScheduling, l.registry, l.config); err != nil {
		return Queue{}, false, err
	} else if changed {
		queue = normalizedQueue
		if err := writeQueueAtomic(l.deps.QueueFile, queue); err != nil {
			return Queue{}, false, err
		}
	}
	if shouldRecoverMissingSyncScripts(warnings, queue) &&
		len(l.activeByID) == 0 &&
		len(l.adoptedTargets) == 0 {
		return Queue{}, false, fmt.Errorf("mise.sync: %s", strings.Join(warnings, "; "))
	}
	if len(l.activeByID) == 0 && len(l.adoptedTargets) == 0 {
		if hasNonPrioritizeItems(queue) {
			filtered := filterStalePrioritizeItems(queue)
			if len(filtered.Items) != len(queue.Items) {
				filtered.GeneratedAt = l.deps.Now().UTC()
				queue = filtered
				if err := writeQueueAtomic(l.deps.QueueFile, queue); err != nil {
					return Queue{}, false, err
				}
			}
		} else {
			if len(brief.Plans) == 0 && len(brief.NeedsScheduling) == 0 {
				l.state = StateIdle
				return Queue{}, false, nil
			}
			queue = bootstrapPrioritizeQueue(l.config, "", l.deps.Now().UTC())
			if err := writeQueueAtomic(l.deps.QueueFile, queue); err != nil {
				return Queue{}, false, err
			}
		}
	}
	if updatedQueue, changed := applyQueueRoutingDefaults(queue, l.registry, l.config); changed {
		queue = updatedQueue
		if err := writeQueueAtomic(l.deps.QueueFile, queue); err != nil {
			return Queue{}, false, err
		}
	}
	return queue, true, nil
}

func (l *Loop) planCycleSpawns(queue Queue, brief mise.Brief) []QueueItem {
	busyTargets := make(map[string]struct{}, len(l.activeByTarget))
	for targetID := range l.activeByTarget {
		busyTargets[targetID] = struct{}{}
	}

	failedTargets := make(map[string]struct{}, len(l.failedTargets))
	for targetID := range l.failedTargets {
		failedTargets[targetID] = struct{}{}
	}

	adoptedTargets := make(map[string]struct{}, len(l.adoptedTargets))
	for targetID := range l.adoptedTargets {
		adoptedTargets[targetID] = struct{}{}
	}

	pendingTargets := make(map[string]struct{}, len(l.pendingReview))
	for targetID := range l.pendingReview {
		pendingTargets[targetID] = struct{}{}
	}

	for targetID := range pendingTargets {
		busyTargets[targetID] = struct{}{}
	}

	for targetID := range l.pendingRetry {
		busyTargets[targetID] = struct{}{}
	}

	return planSpawnItems(spawnPlanInput{
		QueueItems:      queue.Items,
		Capacity:        l.config.Concurrency.MaxCooks,
		ActiveCount:     len(l.activeByID),
		AdoptedCount:    len(l.adoptedTargets),
		BusyTargets:     busyTargets,
		FailedTargets:   failedTargets,
		AdoptedTargets:  adoptedTargets,
		TicketedTargets: activeTicketTargetSet(brief),
	})
}

func (l *Loop) spawnPlannedItems(ctx context.Context, items []QueueItem) error {
	for _, item := range items {
		if err := l.spawnCook(ctx, item, 0, ""); err != nil {
			return err
		}
	}
	return nil
}
