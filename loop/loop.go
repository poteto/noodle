package loop

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/internal/queuex"
	"github.com/poteto/noodle/internal/taskreg"
	"github.com/poteto/noodle/mise"
	"github.com/poteto/noodle/skill"
)

func New(projectDir, noodleBin string, cfg config.Config, deps Dependencies) *Loop {
	projectDir = strings.TrimSpace(projectDir)
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if deps.Logger == nil {
		deps.Logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}
	if deps.Dispatcher == nil || deps.Worktree == nil || deps.Adapter == nil || deps.Mise == nil || deps.Monitor == nil {
		defaults := defaultDependencies(projectDir, runtimeDir, noodleBin, cfg, deps.Logger)
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
		logger:                deps.Logger,
		state:                 StateRunning, // Direct assignment; setState() is not used here to avoid logging the initial state.
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

func (l *Loop) setState(next State) {
	if l.state == next {
		return
	}
	l.logger.Info("state changed", "from", string(l.state), "to", string(next))
	l.state = next
}

func discoverRegistry(projectDir string, cfg config.Config) (taskreg.Registry, error) {
	resolver := skill.Resolver{SearchPaths: cfg.Skills.Paths}
	skills, err := resolver.DiscoverTaskTypes()
	if err != nil {
		return taskreg.NewFromSkills(nil), fmt.Errorf("task type discovery failed: %w", err)
	}
	return taskreg.NewFromSkills(skills), nil
}

func (l *Loop) rebuildRegistry() {
	oldRegistry := l.registry
	registry, err := discoverRegistry(l.projectDir, l.config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "skill.rebuild-registry: %v\n", err)
		return
	}
	l.registry = registry
	l.registryErr = nil
	l.registryFailCount = 0
	if builder, ok := l.deps.Mise.(*mise.Builder); ok {
		builder.TaskTypes = registryToTaskTypeSummaries(registry)
	}

	// Track what changed and emit events.
	diff := diffRegistryKeys(oldRegistry, registry)
	if len(diff.Added) > 0 || len(diff.Removed) > 0 {
		fmt.Fprintf(os.Stderr, "skill registry rebuilt: added %v, removed %v\n", diff.Added, diff.Removed)
	}

	eventsPath := filepath.Join(l.runtimeDir, "queue-events.ndjson")
	rebuildEvent := QueueAuditEvent{
		At:      l.deps.Now().UTC(),
		Type:    "registry_rebuild",
		Added:   diff.Added,
		Removed: diff.Removed,
	}
	appendQueueEvent(eventsPath, rebuildEvent)

	l.auditQueue()
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

	// Watch skill paths for hot-reload.
	skillWatcher, skillWatchErr := skill.NewSkillWatcher(l.config.Skills.Paths, func() {
		l.registryStale.Store(true)
	})
	if skillWatchErr != nil {
		fmt.Fprintf(os.Stderr, "skill.watcher: %v\n", skillWatchErr)
	} else {
		go skillWatcher.Run(ctx)
		defer skillWatcher.Close()
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
					l.setState(StateRunning)
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
	if l.registryStale.Load() {
		l.rebuildRegistry()
		l.registryStale.Store(false)
	}

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
		l.registryFailCount++
		if l.registryFailCount >= 3 {
			return false, l.registryErr
		}
		fmt.Fprintf(os.Stderr, "skill registry error (attempt %d/3, skipping cycle): %v\n", l.registryFailCount, l.registryErr)
		return false, nil
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
		l.setState(StateRunning)
	}
	return brief, warnings, true, nil
}

func (l *Loop) prepareQueueForCycle(brief mise.Brief, warnings []string) (Queue, bool, error) {
	// Consume queue-next.json if the schedule session wrote one.
	// The loop is the single writer of queue.json — schedule writes
	// to queue-next.json to avoid racing with loop state stamps.
	// Errors are non-fatal: a transient/partial write shouldn't crash
	// the loop — log and continue.
	promoted, err := consumeQueueNext(l.deps.QueueNextFile, l.deps.QueueFile)
	if err != nil {
		l.logger.Warn("queue-next promotion failed", "error", err)
	} else if promoted {
		l.logger.Info("queue-next promoted")
	}
	queue, err := readQueue(l.deps.QueueFile)
	if err != nil {
		return Queue{}, false, err
	}
	if normalizedQueue, changed, err := normalizeAndValidateQueue(queue, brief.NeedsScheduling, l.registry, l.config); err != nil {
		if !errors.Is(err, queuex.ErrUnknownTaskType) {
			return Queue{}, false, err
		}
		// Unknown task type — rebuild registry and retry.
		l.rebuildRegistry()
		if normalizedQueue, changed, err = normalizeAndValidateQueue(queue, brief.NeedsScheduling, l.registry, l.config); err != nil {
			if !errors.Is(err, queuex.ErrUnknownTaskType) {
				return Queue{}, false, err
			}
			// Still unknown after re-scan — drop offending items via audit and continue.
			fmt.Fprintf(os.Stderr, "loop.queue-rescan: %v — dropping unknown items\n", err)
			l.auditQueue()
			queue, err = readQueue(l.deps.QueueFile)
			if err != nil {
				return Queue{}, false, err
			}
			if normalizedQueue, changed, err = normalizeAndValidateQueue(queue, brief.NeedsScheduling, l.registry, l.config); err != nil {
				return Queue{}, false, err
			}
			if changed {
				queue = normalizedQueue
				if err := writeQueueAtomic(l.deps.QueueFile, queue); err != nil {
					return Queue{}, false, err
				}
				l.logger.Info("queue normalized")
			}
		} else if changed {
			queue = normalizedQueue
			if err := writeQueueAtomic(l.deps.QueueFile, queue); err != nil {
				return Queue{}, false, err
			}
			l.logger.Info("queue normalized")
		}
	} else if changed {
		queue = normalizedQueue
		if err := writeQueueAtomic(l.deps.QueueFile, queue); err != nil {
			return Queue{}, false, err
		}
		l.logger.Info("queue normalized")
	}
	if shouldRecoverMissingSyncScripts(warnings, queue) &&
		len(l.activeByID) == 0 &&
		len(l.adoptedTargets) == 0 {
		return Queue{}, false, fmt.Errorf("mise.sync: %s", strings.Join(warnings, "; "))
	}
	if len(l.activeByID) == 0 && len(l.adoptedTargets) == 0 {
		if hasNonScheduleItems(queue) {
			filtered := filterStaleScheduleItems(queue)
			if len(filtered.Items) != len(queue.Items) {
				filtered.GeneratedAt = l.deps.Now().UTC()
				queue = filtered
				if err := writeQueueAtomic(l.deps.QueueFile, queue); err != nil {
					return Queue{}, false, err
				}
			}
		} else {
			if len(brief.Plans) == 0 && len(brief.NeedsScheduling) == 0 {
				l.setState(StateIdle)
				return Queue{}, false, nil
			}
			queue = bootstrapScheduleQueue(l.config, "", l.deps.Now().UTC())
			if err := writeQueueAtomic(l.deps.QueueFile, queue); err != nil {
				return Queue{}, false, err
			}
			l.logger.Info("queue empty, bootstrapping schedule")
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
		if err := l.spawnCook(ctx, item, spawnOptions{}); err != nil {
			return err
		}
	}
	return nil
}
