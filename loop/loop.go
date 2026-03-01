package loop

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/event"
	"github.com/poteto/noodle/internal/ingest"
	"github.com/poteto/noodle/internal/reducer"
	"github.com/poteto/noodle/internal/state"
	"github.com/poteto/noodle/internal/taskreg"
	"github.com/poteto/noodle/mise"
	"github.com/poteto/noodle/monitor"
	"github.com/poteto/noodle/skill"
)

func New(projectDir, noodleBin string, cfg config.Config, deps Dependencies) *Loop {
	projectDir = strings.TrimSpace(projectDir)
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if deps.Logger == nil {
		deps.Logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}
	if deps.Runtimes == nil || deps.Worktree == nil || deps.Adapter == nil || deps.Mise == nil || deps.Monitor == nil {
		defaults := defaultDependencies(projectDir, runtimeDir, noodleBin, cfg, deps.Logger, deps.EventSink)
		if deps.Runtimes == nil {
			deps.Runtimes = defaults.Runtimes
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

	eventsWriter := event.NewLoopEventWriter(filepath.Join(runtimeDir, "loop-events.ndjson"))

	loop := &Loop{
		projectDir:  projectDir,
		runtimeDir:  runtimeDir,
		config:      cfg,
		registry:    registry,
		registryErr: registryErr,
		deps:        deps,
		logger:      deps.Logger,
		events:      eventsWriter,
		state:       StateRunning, // Direct assignment; setState() is not used here to avoid logging the initial state.
		cooks: cookTracker{
			activeCooksByOrder: map[string]*cookHandle{},
			adoptedTargets:     map[string]string{},
			adoptedSessions:    []string{},
			pendingReview:      map[string]*pendingReviewCook{},
		},
		cmds: cmdProcessor{
			processedIDs: map[string]struct{}{},
		},
		completionBuf: completionBuffer{
			completions:        make(chan StageResult, 1024),
			completionOverflow: make([]StageResult, 0, maxCompletionOverflow(cfg)),
		},
		activeSummary: mise.ActiveSummary{
			ByTaskKey: map[string]int{},
			ByStatus:  map[string]int{},
			ByRuntime: map[string]int{},
		},
		recentHistory: make([]mise.HistoryItem, 0, 20),
	}
	loop.mergeQueue = NewMergeQueue(context.Background(), func(ctx context.Context, req MergeRequest) error {
		if req.Cook == nil {
			return nil
		}
		return loop.mergeCookWorktree(ctx, req.Cook)
	})
	loop.canonical = state.State{
		Orders: map[string]state.OrderNode{},
		Mode:   state.RunMode(cfg.Mode),
	}
	loop.publishState()
	return loop
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

	_ = l.events.Emit(LoopEventRegistryRebuilt, RegistryRebuiltPayload{
		Added:   diff.Added,
		Removed: diff.Removed,
	})

	l.auditOrders()
}

// Shutdown kills all active agent sessions. Called during process exit.
func (l *Loop) Shutdown() {
	if l.mergeQueue != nil {
		l.mergeQueue.Close()
	}
	for _, cook := range l.cooks.activeCooksByOrder {
		_ = cook.session.Kill()
	}
	// Kill adopted sessions from previous runs that are still alive.
	for _, sessionID := range l.cooks.adoptedSessions {
		monitor.KillSessionByPID(l.runtimeDir, sessionID)
	}
}

func (l *Loop) Run(ctx context.Context) error {
	if strings.TrimSpace(l.projectDir) == "" {
		return fmt.Errorf("project directory not set")
	}
	if err := os.MkdirAll(l.runtimeDir, 0o755); err != nil {
		return fmt.Errorf("create runtime directory: %w", err)
	}
	if err := l.loadOrdersState(); err != nil {
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
			mergeSettled := true
			if l.mergeQueue != nil {
				mergeSettled = l.mergeQueue.Pending() == 0 && l.mergeQueue.InFlight() == 0
			}
			if l.state == StateDraining && l.watcherCount.Load() == 0 && mergeSettled {
				if err := l.drainCompletions(context.Background()); err != nil {
					return err
				}
				if err := l.drainMergeResults(context.Background()); err != nil {
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
	if err := l.loadOrdersState(); err != nil {
		return l.classifySystemHard(
			"cycle.load_orders",
			formatCycleFailureMessage("cycle.load_orders", "load orders"),
			err,
		)
	}

	// Snapshot capacity before control commands can mutate it.
	cycleCapacity := l.config.Concurrency.MaxCooks

	ready, err := l.runCycleMaintenance(ctx)
	if err != nil {
		return l.classifySystemHard(
			"cycle.maintenance",
			formatCycleFailureMessage("cycle.maintenance", "maintenance"),
			err,
		)
	}
	if !ready {
		if err := l.stampStatus(); err != nil {
			return l.classifySystemHard(
				"persist.status_stamp",
				formatCycleFailureMessage("persist.status_stamp", "stamp status"),
				err,
			)
		}
		return nil
	}

	brief, warnings, running, miseChanged, err := l.buildCycleBrief(ctx)
	if err != nil {
		return l.classifySystemHard(
			"build.brief",
			formatCycleFailureMessage("build.brief", "build cycle brief"),
			err,
		)
	}
	if !running {
		if err := l.stampStatus(); err != nil {
			return l.classifySystemHard(
				"persist.status_stamp",
				formatCycleFailureMessage("persist.status_stamp", "stamp status"),
				err,
			)
		}
		return nil
	}

	orders, shouldContinue, err := l.prepareOrdersForCycle(brief, warnings, miseChanged)
	if err != nil {
		return l.classifySystemHard(
			"build.prepare_orders",
			formatCycleFailureMessage("build.prepare_orders", "prepare orders"),
			err,
		)
	}
	if !shouldContinue {
		if err := l.stampStatus(); err != nil {
			return l.classifySystemHard(
				"persist.status_stamp",
				formatCycleFailureMessage("persist.status_stamp", "stamp status"),
				err,
			)
		}
		return nil
	}

	candidates := l.planCycleSpawns(orders, brief, cycleCapacity)
	if err := l.spawnPlannedCandidates(ctx, candidates, orders); err != nil {
		return l.classifySystemHard(
			"cycle.spawn",
			formatCycleFailureMessage("cycle.spawn", "spawn cooks"),
			err,
		)
	}
	// Flush all in-memory state to disk at cycle end.
	if err := l.flushState(); err != nil {
		return l.classifySystemHard(
			"persist.flush_state",
			formatCycleFailureMessage("persist.flush_state", "flush state"),
			err,
		)
	}
	l.publishState()
	if err := l.stampStatus(); err != nil {
		return l.classifySystemHard(
			"persist.status_stamp",
			formatCycleFailureMessage("persist.status_stamp", "stamp status"),
			err,
		)
	}
	return nil
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
	l.runMonitorPass(ctx)
	if err := l.enqueueTerminalActiveCompletions(ctx); err != nil {
		return false, err
	}
	if err := l.drainCompletions(ctx); err != nil {
		return false, err
	}
	if err := l.drainMergeResults(ctx); err != nil {
		return false, err
	}
	if err := l.processControlCommands(); err != nil {
		return false, err
	}
	if err := l.drainCompletions(ctx); err != nil {
		return false, err
	}
	if err := l.drainMergeResults(ctx); err != nil {
		return false, err
	}
	return true, nil
}

func (l *Loop) shutdownAndDrain() {
	l.Shutdown()

	timeout, err := time.ParseDuration(l.config.Concurrency.ShutdownTimeout)
	if err != nil || timeout <= 0 {
		timeout = 30 * time.Second
	}

	done := make(chan struct{})
	go func() {
		l.watcherWG.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All watchers quiesced normally.
	case <-time.After(timeout):
		l.logger.Warn("shutdown timeout exceeded, killing leaked sessions",
			"timeout", timeout,
			"leaked_watchers", l.watcherCount.Load(),
		)
		for orderID, cook := range l.cooks.activeCooksByOrder {
			_ = cook.session.Kill()
			l.logger.Warn("cancelled leaked session", "order_id", orderID, "session_id", cook.session.ID())
		}
	}

	_ = l.drainCompletions(context.Background())
}

// emitEvent creates a StateEvent and applies it to the canonical state via the
// reducer. Effects are logged but not executed — the loop already handles
// execution via its existing paths.
func (l *Loop) emitEvent(eventType ingest.EventType, payload any) {
	id := l.eventCounter.Add(1)
	raw, err := json.Marshal(payload)
	if err != nil {
		l.logger.Warn("canonical event payload encoding failed", "type", string(eventType), "error", err)
		return
	}
	evt := ingest.StateEvent{
		ID:        ingest.EventID(id),
		Source:    string(ingest.SourceInternal),
		Type:      string(eventType),
		Timestamp: l.deps.Now(),
		Payload:   json.RawMessage(raw),
	}
	next, effects, err := reducer.Reduce(l.canonical, evt)
	if err != nil {
		l.logger.Warn("canonical reducer failed", "type", string(eventType), "error", err)
		return
	}
	l.canonical = next
	if len(effects) > 0 {
		l.logger.Debug("canonical effects emitted", "type", string(eventType), "count", len(effects))
	}
}

func maxCompletionOverflow(cfg config.Config) int {
	if cfg.Concurrency.MaxCompletionOverflow <= 0 {
		return 1024
	}
	return cfg.Concurrency.MaxCompletionOverflow
}
