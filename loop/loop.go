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
	}
	if deps.Now == nil {
		deps.Now = time.Now
	}
	if deps.QueueFile == "" {
		deps.QueueFile = filepath.Join(runtimeDir, "queue.json")
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
		processedIDs:          map[string]struct{}{},
		runtimeRepairAttempts: map[string]int{},
	}
}

func discoverRegistry(projectDir string, cfg config.Config) (taskreg.Registry, error) {
	paths := make([]string, 0, len(cfg.Skills.Paths))
	for _, p := range cfg.Skills.Paths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if !filepath.IsAbs(p) {
			p = filepath.Join(projectDir, p)
		}
		paths = append(paths, p)
	}
	resolver := skill.Resolver{SearchPaths: paths}
	skills, err := resolver.DiscoverTaskTypes()
	if err != nil {
		return taskreg.NewFromSkills(nil), fmt.Errorf("task type discovery failed: %w", err)
	}
	return taskreg.NewFromSkills(skills), nil
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
			if strings.HasSuffix(ev.Name, "queue.json") || strings.HasSuffix(ev.Name, "control.ndjson") {
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
	if l.registryErr != nil {
		return l.registryErr
	}
	if err := l.processControlCommands(); err != nil {
		return l.handleRuntimeIssue(ctx, "loop.control", err, nil)
	}
	if err := l.collectCompleted(ctx); err != nil {
		return l.handleRuntimeIssue(ctx, "loop.collect", err, nil)
	}
	if _, err := l.deps.Monitor.RunOnce(ctx); err != nil {
		return l.handleRuntimeIssue(ctx, "loop.monitor", err, nil)
	}
	if err := l.advanceRuntimeRepair(ctx); err != nil {
		return err
	}
	if l.runtimeRepairInFlight != nil {
		return nil
	}
	l.refreshAdoptedTargets()
	brief, warnings, err := l.deps.Mise.Build(ctx)
	if err != nil {
		return l.handleRuntimeIssue(ctx, "mise.build", err, warnings)
	}
	if l.state != StateRunning {
		return nil
	}

	queue, err := readQueue(l.deps.QueueFile)
	if err != nil {
		return err
	}
	if normalizedQueue, changed, err := normalizeAndValidateQueue(queue, brief.Backlog, l.registry, l.config); err != nil {
		return l.handleRuntimeIssue(ctx, "loop.queue", err, nil)
	} else if changed {
		queue = normalizedQueue
		if err := writeQueueAtomic(l.deps.QueueFile, queue); err != nil {
			return err
		}
	}
	if shouldRecoverMissingSyncScripts(warnings, queue) &&
		len(l.activeByID) == 0 &&
		len(l.adoptedTargets) == 0 {
		return l.handleRuntimeIssue(ctx, "mise.sync", nil, warnings)
	}
	if len(queue.Items) == 0 &&
		len(l.activeByID) == 0 &&
		len(l.adoptedTargets) == 0 {
		queue = bootstrapPrioritizeQueue(l.config, "", l.deps.Now().UTC())
		if err := writeQueueAtomic(l.deps.QueueFile, queue); err != nil {
			return err
		}
	}
	if updatedQueue, changed := applyQueueRoutingDefaults(queue, l.registry, l.config); changed {
		queue = updatedQueue
		if err := writeQueueAtomic(l.deps.QueueFile, queue); err != nil {
			return err
		}
	}

	limit := l.config.Concurrency.MaxCooks
	if limit <= 0 {
		limit = 1
	}
	for _, item := range queue.Items {
		if l.hasBlockingActive(queue.Items) {
			break
		}
		if len(l.activeByID)+len(l.adoptedTargets) >= limit {
			break
		}
		if isBlockingQueueItem(l.registry, item) && len(l.activeByID)+len(l.adoptedTargets) > 0 {
			continue
		}
		if _, busy := l.activeByTarget[item.ID]; busy {
			continue
		}
		if _, failed := l.failedTargets[item.ID]; failed {
			continue
		}
		if _, adopted := l.adoptedTargets[item.ID]; adopted {
			continue
		}
		if hasActiveTicket(brief, item.ID) {
			continue
		}
		if err := l.spawnCook(ctx, item, 0, ""); err != nil {
			return l.handleRuntimeIssue(ctx, "loop.spawn", err, nil)
		}
	}
	return nil
}

func (l *Loop) hasBlockingActive(queueItems []QueueItem) bool {
	for _, cook := range l.activeByID {
		if isBlockingQueueItem(l.registry, cook.queueItem) {
			return true
		}
	}
	for targetID := range l.adoptedTargets {
		if item, ok := findQueueItemByTarget(queueItems, targetID); ok {
			if isBlockingQueueItem(l.registry, item) {
				return true
			}
		}
	}
	return false
}
