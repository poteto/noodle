package loop

import (
	"context"
	"fmt"
	"log/slog"
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
		if deps.StatusFile == "" {
			deps.StatusFile = defaults.StatusFile
		}
	}
	if deps.Now == nil {
		deps.Now = time.Now
	}
	if deps.OrdersFile == "" {
		deps.OrdersFile = filepath.Join(runtimeDir, "orders.json")
	}
	if deps.OrdersNextFile == "" {
		deps.OrdersNextFile = filepath.Join(runtimeDir, "orders-next.json")
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
		activeCooksByOrder:    map[string]*cookHandle{},
		adoptedTargets:        map[string]string{},
		adoptedSessions:       []string{},
		failedTargets:         map[string]string{},
		pendingReview:         map[string]*pendingReviewCook{},
		pendingRetry:          map[string]*pendingRetryCook{},
		processedIDs:          map[string]struct{}{},
		completions:           make(chan StageResult, 1024),
		completionOverflow:    make([]StageResult, 0, maxCompletionOverflow(cfg)),
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

	l.auditOrders()
}

// Shutdown kills all active agent sessions. Called during process exit.
func (l *Loop) Shutdown() {
	for _, cook := range l.activeCooksByOrder {
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
			l.shutdownAndDrain()
			return nil
		case <-ticker.C:
			if err := l.Cycle(ctx); err != nil {
				return err
			}
			if l.state == StateDraining && l.watcherCount.Load() == 0 {
				if err := l.drainCompletions(context.Background()); err != nil {
					return err
				}
				return nil
			}
		case ev := <-watcher.Events:
			if strings.HasSuffix(ev.Name, "orders.json") || strings.HasSuffix(ev.Name, "orders-next.json") || strings.HasSuffix(ev.Name, "control.ndjson") {
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

	// Snapshot capacity before control commands can mutate it.
	cycleCapacity := l.config.Concurrency.MaxCooks

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

	orders, shouldContinue, err := l.prepareOrdersForCycle(brief, warnings)
	if err != nil {
		return err
	}
	if !shouldContinue {
		return l.stampStatus()
	}

	candidates := l.planCycleSpawns(orders, brief, cycleCapacity)
	if err := l.spawnPlannedCandidates(ctx, candidates, orders); err != nil {
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
	if err := l.drainCompletions(ctx); err != nil {
		return false, err
	}
	if err := l.processControlCommands(); err != nil {
		return false, err
	}
	if _, err := l.deps.Monitor.RunOnce(ctx); err != nil {
		return false, err
	}
	if err := l.processPendingRetries(ctx); err != nil {
		return false, err
	}
	if err := l.drainCompletions(ctx); err != nil {
		return false, err
	}
	return true, nil
}

func (l *Loop) shutdownAndDrain() {
	l.Shutdown()
	l.watcherWG.Wait()
	_ = l.drainCompletions(context.Background())
}

func maxCompletionOverflow(cfg config.Config) int {
	if cfg.Concurrency.MaxCompletionOverflow <= 0 {
		return 1024
	}
	return cfg.Concurrency.MaxCompletionOverflow
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

func (l *Loop) prepareOrdersForCycle(brief mise.Brief, warnings []string) (OrdersFile, bool, error) {
	// Consume orders-next.json if the schedule session wrote one.
	promoted, err := consumeOrdersNext(l.deps.OrdersNextFile, l.deps.OrdersFile)
	if err != nil {
		l.logger.Warn("orders-next promotion failed", "error", err)
	} else if promoted {
		l.logger.Info("orders-next promoted")
	}

	orders, err := readOrders(l.deps.OrdersFile)
	if err != nil {
		return OrdersFile{}, false, err
	}

	// Normalize and validate orders.
	normalizedOrders, changed, normErr := NormalizeAndValidateOrders(orders, l.registry, l.config)
	if normErr != nil {
		// Rebuild registry and retry on unknown task type.
		l.rebuildRegistry()
		normalizedOrders, changed, normErr = NormalizeAndValidateOrders(orders, l.registry, l.config)
		if normErr != nil {
			l.auditOrders()
			orders, err = readOrders(l.deps.OrdersFile)
			if err != nil {
				return OrdersFile{}, false, err
			}
			normalizedOrders, changed, normErr = NormalizeAndValidateOrders(orders, l.registry, l.config)
			if normErr != nil {
				return OrdersFile{}, false, normErr
			}
		}
	}
	if changed {
		orders = normalizedOrders
		if err := writeOrdersAtomic(l.deps.OrdersFile, orders); err != nil {
			return OrdersFile{}, false, err
		}
		l.logger.Info("orders normalized")
	}

	if hasSyncWarnings(warnings) {
		l.logger.Warn("sync script issue, continuing with empty backlog", "warnings", strings.Join(warnings, "; "))
		eventsPath := filepath.Join(l.runtimeDir, "queue-events.ndjson")
		appendQueueEvent(eventsPath, QueueAuditEvent{
			At:     l.deps.Now().UTC(),
			Type:   "sync_degraded",
			Reason: strings.Join(warnings, "; "),
		})
	}

	// Simplified filtering (#60): check for non-schedule orders.
	if len(l.activeCooksByOrder) == 0 && len(l.adoptedTargets) == 0 {
		if !hasNonScheduleOrders(orders) {
			if len(brief.Plans) == 0 {
				l.setState(StateIdle)
				return OrdersFile{}, false, nil
			}
			// Bootstrap a schedule order.
			orders = bootstrapScheduleOrder(l.config)
			if err := writeOrdersAtomic(l.deps.OrdersFile, orders); err != nil {
				return OrdersFile{}, false, err
			}
			l.logger.Info("orders empty, bootstrapping schedule")
		}
	}

	if updatedOrders, changed := ApplyOrderRoutingDefaults(orders, l.registry, l.config); changed {
		orders = updatedOrders
		if err := writeOrdersAtomic(l.deps.OrdersFile, orders); err != nil {
			return OrdersFile{}, false, err
		}
	}
	return orders, true, nil
}

func (l *Loop) planCycleSpawns(orders OrdersFile, brief mise.Brief, capacity int) []dispatchCandidate {
	orderBusyTargets := BusyTargets(orders)
	busyTargets := make(map[string]struct{}, len(orderBusyTargets)+len(l.pendingRetry)+len(l.activeCooksByOrder)+len(l.adoptedTargets))
	for targetID, busy := range orderBusyTargets {
		if busy {
			busyTargets[targetID] = struct{}{}
		}
	}

	failedTargets := make(map[string]struct{}, len(l.failedTargets))
	for targetID := range l.failedTargets {
		failedTargets[targetID] = struct{}{}
	}

	adoptedTargets := make(map[string]struct{}, len(l.adoptedTargets))
	for targetID := range l.adoptedTargets {
		adoptedTargets[targetID] = struct{}{}
	}

	for targetID := range l.pendingReview {
		busyTargets[targetID] = struct{}{}
	}

	for targetID := range l.pendingRetry {
		busyTargets[targetID] = struct{}{}
	}
	for targetID := range l.activeCooksByOrder {
		busyTargets[targetID] = struct{}{}
	}
	for targetID := range l.adoptedTargets {
		busyTargets[targetID] = struct{}{}
	}

	candidates := dispatchableStages(orders, busyTargets, failedTargets, adoptedTargets, activeTicketTargetSet(brief))

	// Limit to capacity.
	limit := capacity
	if limit <= 0 {
		limit = 1
	}
	current := len(l.activeCooksByOrder) + len(l.adoptedTargets)
	available := limit - current
	if available <= 0 {
		return nil
	}
	if len(candidates) > available {
		candidates = candidates[:available]
	}
	return candidates
}

func (l *Loop) spawnPlannedCandidates(ctx context.Context, candidates []dispatchCandidate, orders OrdersFile) error {
	// Build order lookup for candidate dispatch.
	orderMap := make(map[string]Order, len(orders.Orders))
	for _, o := range orders.Orders {
		orderMap[o.ID] = o
	}
	for _, cand := range candidates {
		if l.atMaxConcurrency() {
			break
		}
		order, ok := orderMap[cand.OrderID]
		if !ok {
			continue
		}
		if err := l.spawnCook(ctx, cand, order, spawnOptions{}); err != nil {
			return err
		}
	}
	return nil
}
