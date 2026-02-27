package loop

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/poteto/noodle/adapter"
	"github.com/poteto/noodle/dispatcher"
	"github.com/poteto/noodle/event"
	"github.com/poteto/noodle/internal/stringx"
	"github.com/poteto/noodle/internal/taskreg"
	"github.com/poteto/noodle/recover"
	"github.com/poteto/noodle/worktree"
)

// ensureSkillFresh verifies the skill is resolvable via the registry.
// If not found, it rebuilds the registry once and retries.
// Returns true if the skill exists after potential rebuild.
func (l *Loop) ensureSkillFresh(skillName string) bool {
	if _, ok := l.registry.ByKey(skillName); ok {
		return true
	}
	l.rebuildRegistry()
	_, ok := l.registry.ByKey(skillName)
	return ok
}

type spawnOptions struct {
	attempt     int
	resume      string
	displayName string // preserved across retries; empty = compute from session ID
}

func (l *Loop) atMaxConcurrency() bool {
	maxCooks := l.config.Concurrency.MaxCooks
	if maxCooks <= 0 {
		maxCooks = 1
	}
	return len(l.activeCooksByOrder)+len(l.adoptedTargets) >= maxCooks
}

func (l *Loop) spawnCook(ctx context.Context, cand dispatchCandidate, order Order, opts spawnOptions) error {
	if isScheduleStage(cand.Stage) {
		return l.spawnSchedule(ctx, order, opts.attempt, opts.resume)
	}

	stage := cand.Stage

	// Belt-and-suspenders: give the registry one last chance to pick up
	// the skill before dispatch, in case fsnotify missed an event.
	if skillName := strings.TrimSpace(stage.Skill); skillName != "" {
		l.ensureSkillFresh(skillName)
	}

	name := cookBaseName(cand.OrderID, cand.StageIndex, stage.TaskKey)

	created, err := l.ensureWorktree(name)
	if err != nil {
		return fmt.Errorf("create worktree %s: %w", name, err)
	}

	// Guard: only one cook may use a worktree at a time.
	for _, active := range l.activeCooksByOrder {
		if active.worktreeName == name {
			return fmt.Errorf("worktree %s already in use by session %s", name, active.session.ID())
		}
	}

	resumePrompt := opts.resume
	worktreePath := l.worktreePath(name)
	if !created {
		if opts.attempt > 0 {
			resetWorktreeState(worktreePath)
		}
		if hint := worktreeResumeContext(worktreePath, name); hint != "" {
			resumePrompt = joinPromptParts(hint, resumePrompt)
		}
	}
	prompt := buildCookPrompt(cand.OrderID, stage, order.Plan, order.Rationale, resumePrompt)

	taskType, _ := l.registry.ByKey(stage.TaskKey)
	req := dispatcher.DispatchRequest{
		Name:         name,
		Prompt:       prompt,
		Provider:     nonEmpty(stage.Provider, l.config.Routing.Defaults.Provider),
		Model:        nonEmpty(stage.Model, l.config.Routing.Defaults.Model),
		Skill:        stage.Skill,
		WorktreePath: worktreePath,
		TaskKey:      taskType.Key,
		Runtime:      nonEmpty(stage.Runtime, "tmux"),
		DisplayName:  opts.displayName,
		Title:        order.Title,
		RetryCount:   opts.attempt,
	}
	if taskType.DomainSkill != "" {
		req.DomainSkill = taskType.DomainSkill
	}

	// Persist active status BEFORE spawning session — restart safety.
	if err := l.persistOrderStageStatus(cand.OrderID, cand.StageIndex, cand.IsOnFailure, StageStatusActive); err != nil {
		if created {
			_ = l.deps.Worktree.Cleanup(name, true)
		}
		return err
	}

	session, err := l.deps.Dispatcher.Dispatch(ctx, req)
	if err != nil {
		// Revert stage status to pending — otherwise restart sees "active"
		// with no session, permanently stranding the stage.
		_ = l.persistOrderStageStatus(cand.OrderID, cand.StageIndex, cand.IsOnFailure, StageStatusPending)
		if created {
			_ = l.deps.Worktree.Cleanup(name, true)
		}
		return err
	}

	displayName := strings.TrimSpace(opts.displayName)
	if displayName == "" {
		displayName = stringx.KitchenName(session.ID())
	}

	cook := &cookHandle{
		orderID:      cand.OrderID,
		stageIndex:   cand.StageIndex,
		stage:        stage,
		isOnFailure:  cand.IsOnFailure,
		orderStatus:  order.Status,
		plan:         order.Plan,
		session:      session,
		worktreeName: name,
		worktreePath: req.WorktreePath,
		attempt:      opts.attempt,
		generation:   l.nextDispatchGeneration(),
		displayName:  displayName,
	}
	l.activeCooksByOrder[cand.OrderID] = cook
	l.startSessionWatcher(ctx, cook, false)
	l.logger.Info("cook dispatched", "order", cand.OrderID, "stage", cand.StageIndex, "session", session.ID(), "worktree", name, "attempt", opts.attempt)
	return nil
}

func (l *Loop) drainCompletions(ctx context.Context) error {
	drainedAny := false
	for {
		select {
		case result := <-l.completions:
			drainedAny = true
			if err := l.applyStageResult(ctx, result); err != nil {
				return err
			}
		default:
			goto drainedChannel
		}
	}

drainedChannel:
	if !drainedAny && l.watcherCount.Load() > 0 {
		waitCtx := ctx
		if waitCtx == nil {
			waitCtx = context.Background()
		}
		select {
		case result := <-l.completions:
			if err := l.applyStageResult(waitCtx, result); err != nil {
				return err
			}
			for {
				select {
				case late := <-l.completions:
					if err := l.applyStageResult(waitCtx, late); err != nil {
						return err
					}
				default:
					goto lateDrainDone
				}
			}
		case <-time.After(2 * time.Millisecond):
		case <-waitCtx.Done():
		}
	}

lateDrainDone:
	overflow := l.takeCompletionOverflow()
	for _, result := range overflow {
		if err := l.applyStageResult(ctx, result); err != nil {
			return err
		}
	}
	return l.collectAdoptedCompletions(ctx)
}

func (l *Loop) applyStageResult(ctx context.Context, result StageResult) error {
	if result.IsBootstrap {
		l.handleBootstrapResult(result)
		return nil
	}
	cook, exists := l.activeCooksByOrder[result.OrderID]
	if !exists {
		return nil
	}
	if cook.generation != result.Generation {
		return nil
	}
	delete(l.activeCooksByOrder, cook.orderID)
	if err := l.handleCompletion(ctx, cook); err != nil {
		if conflictErr := l.handleMergeConflict(cook, err); conflictErr != nil {
			l.pendingRetry[cook.orderID] = &pendingRetryCook{
				orderID:     cook.orderID,
				stageIndex:  cook.stageIndex,
				stage:       cook.stage,
				isOnFailure: cook.isOnFailure,
				orderStatus: cook.orderStatus,
				plan:        cook.plan,
				attempt:     cook.attempt + 1,
				displayName: cook.displayName,
			}
			return conflictErr
		}
	}
	return nil
}

func (l *Loop) handleBootstrapResult(result StageResult) {
	if l.bootstrapInFlight == nil {
		return
	}
	if l.bootstrapInFlight.generation != result.Generation {
		return
	}
	l.bootstrapInFlight = nil

	if result.Status == StageResultCompleted {
		l.rebuildRegistry()
		eventsPath := filepath.Join(l.runtimeDir, "queue-events.ndjson")
		appendQueueEvent(eventsPath, QueueAuditEvent{
			At:   l.deps.Now().UTC(),
			Type: "bootstrap_complete",
		})
		l.logger.Info("bootstrap completed")
		return
	}

	l.bootstrapAttempts++
	if l.bootstrapAttempts >= 3 {
		l.bootstrapExhausted = true
	}
	l.logger.Warn("bootstrap failed", "attempt", l.bootstrapAttempts, "status", string(result.Status))
}

func (l *Loop) nextDispatchGeneration() uint64 {
	return l.dispatchGeneration.Add(1)
}

func (l *Loop) startSessionWatcher(ctx context.Context, cook *cookHandle, isBootstrap bool) {
	if cook == nil || cook.session == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}

	l.watcherWG.Add(1)
	l.watcherCount.Add(1)
	go func(sessionID string, handle *cookHandle, watcherCtx context.Context) {
		defer l.watcherWG.Done()
		defer l.watcherCount.Add(-1)

		<-handle.session.Done()
		result := StageResult{
			OrderID:      handle.orderID,
			StageIndex:   handle.stageIndex,
			Attempt:      handle.attempt,
			IsOnFailure:  handle.isOnFailure,
			Status:       stageResultStatus(handle.session.Status()),
			SessionID:    sessionID,
			Generation:   handle.generation,
			IsSchedule:   isScheduleStage(handle.stage),
			IsBootstrap:  isBootstrap,
			WorktreeName: handle.worktreeName,
			WorktreePath: handle.worktreePath,
			CompletedAt:  l.deps.Now(),
		}
		l.enqueueCompletion(watcherCtx, result)
	}(cook.session.ID(), cook, ctx)
}

func stageResultStatus(raw string) StageResultStatus {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "completed":
		return StageResultCompleted
	case "killed", "cancelled", "canceled", "stopped":
		return StageResultCancelled
	default:
		return StageResultFailed
	}
}

func (l *Loop) enqueueCompletion(ctx context.Context, result StageResult) {
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case l.completions <- result:
		return
	default:
	}

	overflowCap := cap(l.completionOverflow)
	if overflowCap <= 0 {
		overflowCap = maxCompletionOverflow(l.config)
	}

	l.completionOverflowMu.Lock()
	if len(l.completionOverflow) < overflowCap {
		l.completionOverflow = append(l.completionOverflow, result)
		l.completionOverflowMu.Unlock()
		return
	}
	l.completionOverflowSaturated++
	l.completionOverflowMu.Unlock()

	select {
	case l.completions <- result:
	case <-ctx.Done():
	}
}

func (l *Loop) takeCompletionOverflow() []StageResult {
	l.completionOverflowMu.Lock()
	defer l.completionOverflowMu.Unlock()
	if len(l.completionOverflow) == 0 {
		return nil
	}
	drained := append([]StageResult(nil), l.completionOverflow...)
	l.completionOverflow = l.completionOverflow[:0]
	return drained
}

func (l *Loop) handleCompletion(ctx context.Context, cook *cookHandle) error {
	status := strings.ToLower(strings.TrimSpace(cook.session.Status()))
	success := status == "completed"
	if success {
		if isScheduleStage(cook.stage) {
			l.logger.Info("schedule completed", "session", cook.session.ID())
			return l.removeOrder(cook.orderID)
		}

		canMerge := l.canMergeStage(cook.stage)

		// In approve autonomy mode, park for human review.
		if l.config.PendingApproval() {
			l.logger.Info("cook parked for review", "order", cook.orderID, "session", cook.session.ID())
			return l.parkPendingReview(cook, "")
		}

		// Quality verdict gate (auto autonomy mode only).
		if canMerge {
			verdict, hasVerdict := l.readQualityVerdict(cook.session.ID())
			if hasVerdict && !verdict.Accept {
				l.logger.Warn("quality verdict rejected", "order", cook.orderID, "session", cook.session.ID(), "feedback", verdict.Feedback)
				return l.failAndPersist(cook, "quality rejected: "+verdict.Feedback)
			}
		}

		if !canMerge {
			// Non-mergeable stage (e.g., debate, schedule) — advance without merge.
			return l.advanceAndPersist(ctx, cook)
		}

		l.logger.Info("cook completing", "order", cook.orderID, "session", cook.session.ID())
		if err := l.mergeCookWorktree(ctx, cook); err != nil {
			return err
		}
		return l.advanceAndPersist(ctx, cook)
	}
	return l.retryCook(ctx, cook, "cook exited with status "+status)
}

// readQualityVerdict reads the quality verdict file for a session.
// Returns (verdict, true) when a valid verdict exists, (zero, false) when no
// verdict file is present. Parse errors log a warning and return (zero, false)
// so a corrupt file doesn't silently bypass the quality gate on retry.
func (l *Loop) readQualityVerdict(sessionID string) (QualityVerdict, bool) {
	path := filepath.Join(l.runtimeDir, "quality", sessionID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return QualityVerdict{}, false
	}
	var verdict QualityVerdict
	if err := json.Unmarshal(data, &verdict); err != nil {
		l.logger.Warn("corrupt quality verdict file", "path", path, "err", err)
		return QualityVerdict{}, false
	}
	return verdict, true
}

// advanceAndPersist advances the order stage and persists the result.
func (l *Loop) advanceAndPersist(ctx context.Context, cook *cookHandle) error {
	orders, err := readOrders(l.deps.OrdersFile)
	if err != nil {
		return err
	}
	orders, removed, err := advanceOrder(orders, cook.orderID)
	if err != nil {
		return err
	}
	if err := writeOrdersAtomic(l.deps.OrdersFile, orders); err != nil {
		return err
	}
	if removed {
		if cook.orderStatus == OrderStatusFailing || cook.isOnFailure {
			// OnFailure pipeline completed — the original failure stands.
			return l.markFailed(cook.orderID, "on-failure pipeline completed")
		}
		// Final stage of a non-failing order — fire adapter "done".
		if _, err := l.deps.Adapter.Run(ctx, "backlog", "done", adapter.RunOptions{Args: []string{cook.orderID}}); err != nil {
			if !isMissingAdapter(err) {
				return err
			}
		}
	}
	return nil
}

// failAndPersist calls failStage and persists the result.
func (l *Loop) failAndPersist(cook *cookHandle, reason string) error {
	orders, err := readOrders(l.deps.OrdersFile)
	if err != nil {
		return err
	}
	orders, terminal, err := failStage(orders, cook.orderID, reason)
	if err != nil {
		return err
	}
	if err := writeOrdersAtomic(l.deps.OrdersFile, orders); err != nil {
		return err
	}
	if terminal {
		if err := l.markFailed(cook.orderID, reason); err != nil {
			return err
		}
	}
	// Clean up worktree on failure.
	if strings.TrimSpace(cook.worktreeName) != "" {
		_ = l.deps.Worktree.Cleanup(cook.worktreeName, true)
	}
	return nil
}

// mergeCookWorktree merges the cook's worktree to main.
func (l *Loop) mergeCookWorktree(ctx context.Context, cook *cookHandle) error {
	syncResult, hasSyncResult, err := l.readSessionSyncResult(cook.session.ID())
	if err != nil {
		return err
	}
	if hasSyncResult && syncResult.Type == dispatcher.SyncResultTypeBranch && strings.TrimSpace(syncResult.Branch) != "" {
		if err := l.deps.Worktree.MergeRemoteBranch(syncResult.Branch); err != nil {
			return fmt.Errorf("merge remote branch %s: %w", syncResult.Branch, err)
		}
	} else {
		if err := l.deps.Worktree.Merge(cook.worktreeName); err != nil {
			return fmt.Errorf("merge %s: %w", cook.worktreeName, err)
		}
	}
	l.logger.Info("cook merged", "order", cook.orderID, "worktree", cook.worktreeName)
	return nil
}

func (l *Loop) canMergeStage(stage Stage) bool {
	taskType, ok := l.registry.ResolveStage(taskreg.StageInput{
		TaskKey: stage.TaskKey,
		Skill:   stage.Skill,
	})
	if !ok {
		return true
	}
	return taskType.CanMerge
}


func (l *Loop) readSessionSyncResult(sessionID string) (dispatcher.SyncResult, bool, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return dispatcher.SyncResult{}, false, nil
	}
	path := filepath.Join(l.runtimeDir, "sessions", sessionID, "spawn.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return dispatcher.SyncResult{}, false, nil
		}
		return dispatcher.SyncResult{}, false, fmt.Errorf("read spawn metadata: %w", err)
	}
	var payload struct {
		Sync dispatcher.SyncResult `json:"sync"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return dispatcher.SyncResult{}, false, fmt.Errorf("parse spawn metadata: %w", err)
	}
	if strings.TrimSpace(payload.Sync.Type) == "" && strings.TrimSpace(payload.Sync.Branch) == "" {
		return dispatcher.SyncResult{}, false, nil
	}
	payload.Sync.Type = strings.ToLower(strings.TrimSpace(payload.Sync.Type))
	payload.Sync.Branch = strings.TrimSpace(payload.Sync.Branch)
	return payload.Sync, true, nil
}

func (l *Loop) collectAdoptedCompletions(ctx context.Context) error {
	for targetID, sessionID := range l.adoptedTargets {
		status, ok, err := l.readSessionStatus(sessionID)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		switch status {
		case "running", "stuck", "spawning":
			continue
		}
		cook, processable, err := l.buildAdoptedCook(targetID, sessionID, status)
		if err != nil {
			return err
		}
		if !processable {
			l.logger.Info("adopted session dropped", "order", targetID, "session", sessionID)
			l.dropAdoptedTarget(targetID, sessionID)
			continue
		}
		l.logger.Info("adopted session completed", "order", targetID, "session", sessionID, "status", status)
		if err := l.handleCompletion(ctx, cook); err != nil {
			if conflictErr := l.handleMergeConflict(cook, err); conflictErr != nil {
				return conflictErr
			}
		}
		l.dropAdoptedTarget(targetID, sessionID)
	}
	return nil
}

func (l *Loop) handleMergeConflict(cook *cookHandle, err error) error {
	var conflictErr *worktree.MergeConflictError
	if !errors.As(err, &conflictErr) {
		return err
	}
	if isScheduleStage(cook.stage) {
		return err
	}
	// Park for pending review instead of permanent failure.
	reason := "merge conflict: " + conflictErr.Error()
	l.logger.Warn("merge conflict, parking for review", "order", cook.orderID, "reason", reason)
	if parkErr := l.parkPendingReview(cook, reason); parkErr != nil {
		return parkErr
	}
	return nil
}

func (l *Loop) worktreePath(name string) string {
	return filepath.Join(l.projectDir, ".worktrees", name)
}

// resetWorktreeState discards uncommitted changes in a worktree so the next
// agent starts from a clean working tree. Committed progress on the worktree
// branch is preserved. Errors are logged but not fatal — a dirty worktree is
// better than failing to dispatch.
func resetWorktreeState(worktreePath string) {
	// Reset tracked files to HEAD.
	checkout := exec.Command("git", "-C", worktreePath, "checkout", ".")
	checkout.Stdout, checkout.Stderr = nil, nil
	_ = checkout.Run()

	// Remove untracked files and directories.
	clean := exec.Command("git", "-C", worktreePath, "clean", "-fd")
	clean.Stdout, clean.Stderr = nil, nil
	_ = clean.Run()
}

func (l *Loop) ensureWorktree(name string) (bool, error) {
	path := l.worktreePath(name)
	info, err := os.Stat(path)
	if err == nil {
		if !info.IsDir() {
			return false, fmt.Errorf("worktree path %s is not a directory", path)
		}
		return false, nil
	}
	if !os.IsNotExist(err) {
		return false, fmt.Errorf("stat worktree path %s: %w", path, err)
	}

	if err := l.deps.Worktree.Create(name); err != nil {
		if isWorktreeAlreadyExistsError(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (l *Loop) readSessionStatus(sessionID string) (string, bool, error) {
	metaPath := filepath.Join(l.runtimeDir, "sessions", sessionID, "meta.json")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, err
	}
	var payload struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return "", false, err
	}
	return strings.ToLower(strings.TrimSpace(payload.Status)), true, nil
}

func (l *Loop) buildAdoptedCook(targetID string, sessionID string, status string) (*cookHandle, bool, error) {
	// Try orders-based lookup first.
	orders, err := readOrders(l.deps.OrdersFile)
	if err != nil {
		return nil, false, err
	}
	for _, order := range orders.Orders {
		if order.ID != targetID {
			continue
		}
		idx, stg := activeStageForOrder(order)
		if idx < 0 || stg == nil {
			return nil, false, nil
		}
		name := cookBaseName(order.ID, idx, stg.TaskKey)
		worktreePath := l.worktreePath(name)
		return &cookHandle{
			orderID:     order.ID,
			stageIndex:  idx,
			stage:       *stg,
			isOnFailure: order.Status == OrderStatusFailing,
			orderStatus: order.Status,
			plan:        order.Plan,
			session: &adoptedSession{
				id:     sessionID,
				status: status,
			},
			worktreeName: name,
			worktreePath: worktreePath,
			attempt:      recover.RecoveryChainLength(name),
		}, true, nil
	}

	return nil, false, nil
}


func (l *Loop) dropAdoptedTarget(targetID string, sessionID string) {
	delete(l.adoptedTargets, targetID)
	filtered := l.adoptedSessions[:0]
	for _, id := range l.adoptedSessions {
		if id == sessionID {
			continue
		}
		filtered = append(filtered, id)
	}
	l.adoptedSessions = filtered
}

func (l *Loop) processPendingRetries(ctx context.Context) error {
	if len(l.pendingRetry) == 0 {
		return nil
	}
	pending := l.pendingRetry
	l.pendingRetry = map[string]*pendingRetryCook{}
	for _, p := range pending {
		if l.atMaxConcurrency() {
			l.pendingRetry[p.orderID] = p
			continue
		}
		cand := dispatchCandidate{
			OrderID:     p.orderID,
			StageIndex:  p.stageIndex,
			Stage:       p.stage,
			IsOnFailure: p.isOnFailure,
		}
		order := Order{
			ID:     p.orderID,
			Status: p.orderStatus,
			Plan:   p.plan,
		}
		if err := l.spawnCook(ctx, cand, order, spawnOptions{
			attempt:     p.attempt,
			displayName: p.displayName,
		}); err != nil {
			if p.attempt >= l.config.Recovery.MaxRetries {
				fmt.Fprintf(os.Stderr, "loop.pending-retry: %s exhausted retries: %v\n", p.orderID, err)
				if markErr := l.markFailed(p.orderID, err.Error()); markErr != nil {
					fmt.Fprintf(os.Stderr, "loop.pending-retry: mark failed %s: %v\n", p.orderID, markErr)
				}
				continue
			}
			l.pendingRetry[p.orderID] = &pendingRetryCook{
				orderID:     p.orderID,
				stageIndex:  p.stageIndex,
				stage:       p.stage,
				isOnFailure: p.isOnFailure,
				orderStatus: p.orderStatus,
				plan:        p.plan,
				attempt:     p.attempt + 1,
				displayName: p.displayName,
			}
			continue
		}
	}
	return nil
}

func (l *Loop) retryCook(ctx context.Context, cook *cookHandle, reason string) error {
	nextAttempt := cook.attempt + 1
	info, err := recover.CollectRecoveryInfo(ctx, l.runtimeDir, cook.session.ID())
	if err != nil {
		info = recover.RecoveryInfo{SessionID: cook.session.ID(), ExitReason: reason}
	}
	resolvedReason := retryFailureReason(reason, info)
	if nextAttempt > l.config.Recovery.MaxRetries {
		if isScheduleStage(cook.stage) {
			return fmt.Errorf("schedule failed after retries: %s", resolvedReason)
		}
		l.logger.Warn("cook failed permanently", "order", cook.orderID, "session", cook.session.ID(), "reason", resolvedReason)
		if err := l.failAndPersist(cook, resolvedReason); err != nil {
			return err
		}
		return nil
	}
	l.logger.Info("cook retrying", "order", cook.orderID, "session", cook.session.ID(), "attempt", nextAttempt, "reason", resolvedReason)

	if l.atMaxConcurrency() {
		l.pendingRetry[cook.orderID] = &pendingRetryCook{
			orderID:     cook.orderID,
			stageIndex:  cook.stageIndex,
			stage:       cook.stage,
			isOnFailure: cook.isOnFailure,
			orderStatus: cook.orderStatus,
			plan:        cook.plan,
			attempt:     nextAttempt,
			displayName: cook.displayName,
		}
		l.logger.Info("retry deferred: at max concurrency", "order", cook.orderID, "attempt", nextAttempt)
		return nil
	}

	if strings.TrimSpace(info.ExitReason) == "" {
		info.ExitReason = resolvedReason
	}
	resume := recover.BuildResumeContext(info, nextAttempt, l.config.Recovery.MaxRetries)
	cand := dispatchCandidate{
		OrderID:     cook.orderID,
		StageIndex:  cook.stageIndex,
		Stage:       cook.stage,
		IsOnFailure: cook.isOnFailure,
	}
	order := Order{
		ID:        cook.orderID,
		Status:    cook.orderStatus,
		Plan:      cook.plan,
		Rationale: "",
	}
	return l.spawnCook(ctx, cand, order, spawnOptions{
		attempt:     nextAttempt,
		resume:      resume.Summary,
		displayName: cook.displayName,
	})
}

func retryFailureReason(base string, info recover.RecoveryInfo) string {
	base = strings.TrimSpace(base)
	if base == "" {
		base = "cook failed"
	}

	exitReason := strings.TrimSpace(info.ExitReason)
	if exitReason == "" {
		return base
	}
	if strings.EqualFold(exitReason, "session exited without explicit reason") {
		return base
	}

	if strings.HasPrefix(strings.ToLower(base), "cook exited with status") {
		return exitReason
	}
	return base
}

// removeOrder removes an order from orders.json by ID.
func (l *Loop) removeOrder(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("remove requires order ID")
	}
	orders, err := readOrders(l.deps.OrdersFile)
	if err != nil {
		return err
	}
	filtered := make([]Order, 0, len(orders.Orders))
	for _, order := range orders.Orders {
		if order.ID == id {
			continue
		}
		filtered = append(filtered, order)
	}
	orders.Orders = filtered
	if err := writeOrdersAtomic(l.deps.OrdersFile, orders); err != nil {
		return err
	}
	l.logger.Info("order removed", "order", id)
	return nil
}

func (l *Loop) killCook(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("kill requires name")
	}
	for _, cook := range l.activeCooksByOrder {
		if cook.worktreeName == name || cook.session.ID() == name {
			return cook.session.Kill()
		}
	}
	return fmt.Errorf("session not found")
}

func (l *Loop) steer(target string, prompt string) error {
	target = strings.TrimSpace(target)
	if target == "" {
		return fmt.Errorf("steer requires target")
	}
	if strings.EqualFold(target, ScheduleTaskKey()) {
		return l.rescheduleForChefPrompt(prompt)
	}
	for _, cook := range l.activeCooksByOrder {
		if cook.worktreeName != target && cook.session.ID() != target {
			continue
		}
		resumeCtx := buildSteerResumeContext(l.runtimeDir, cook.session.ID())
		steerPrompt := strings.TrimSpace(prompt)
		if resumeCtx != "" {
			steerPrompt = "Resume context: " + resumeCtx + "\n\nChef steering: " + steerPrompt
		}

		if err := cook.session.Kill(); err != nil {
			return err
		}
		delete(l.activeCooksByOrder, cook.orderID)
		cand := dispatchCandidate{
			OrderID:     cook.orderID,
			StageIndex:  cook.stageIndex,
			Stage:       cook.stage,
			IsOnFailure: cook.isOnFailure,
		}
		order := Order{
			ID:     cook.orderID,
			Status: cook.orderStatus,
			Plan:   cook.plan,
		}
		return l.spawnCook(context.Background(), cand, order, spawnOptions{
			attempt:     cook.attempt,
			resume:      steerPrompt,
			displayName: cook.displayName,
		})
	}
	return errors.New("session not found")
}

func (l *Loop) persistOrderStageStatus(orderID string, stageIndex int, isOnFailure bool, status string) error {
	orders, err := readOrders(l.deps.OrdersFile)
	if err != nil {
		return err
	}
	for i := range orders.Orders {
		if orders.Orders[i].ID != orderID {
			continue
		}
		stages := &orders.Orders[i].Stages
		if isOnFailure {
			stages = &orders.Orders[i].OnFailure
		}
		if stageIndex < len(*stages) {
			(*stages)[stageIndex].Status = status
		}
		break
	}
	return writeOrdersAtomic(l.deps.OrdersFile, orders)
}

// buildSteerResumeContext reads a session's event log and extracts a progress
// summary so the respawned session doesn't start from scratch.
func buildSteerResumeContext(runtimeDir string, sessionID string) string {
	reader := event.NewEventReader(runtimeDir)
	events, err := reader.ReadSession(sessionID, event.EventFilter{})
	if err != nil || len(events) == 0 {
		return ""
	}

	files := make(map[string]struct{})
	var lastActions []string
	var ticketProgress []string

	for _, ev := range events {
		switch ev.Type {
		case event.EventAction:
			var action struct {
				Tool    string `json:"tool"`
				Path    string `json:"path"`
				Summary string `json:"summary"`
			}
			_ = json.Unmarshal(ev.Payload, &action)
			tool := strings.ToLower(strings.TrimSpace(action.Tool))
			if path := strings.TrimSpace(action.Path); path != "" {
				switch tool {
				case "read", "edit", "write":
					files[path] = struct{}{}
				}
			}
			summary := strings.TrimSpace(action.Summary)
			if summary == "" {
				summary = strings.TrimSpace(action.Tool)
			}
			if summary != "" {
				lastActions = append(lastActions, summary)
			}
		case event.EventTicketProgress, event.EventTicketDone:
			var payload struct {
				Summary string `json:"summary"`
				Outcome string `json:"outcome"`
			}
			_ = json.Unmarshal(ev.Payload, &payload)
			if s := strings.TrimSpace(payload.Summary); s != "" {
				ticketProgress = append(ticketProgress, s)
			} else if s := strings.TrimSpace(payload.Outcome); s != "" {
				ticketProgress = append(ticketProgress, s)
			}
		}
	}

	var parts []string
	if len(files) > 0 {
		fileList := make([]string, 0, len(files))
		for f := range files {
			fileList = append(fileList, f)
		}
		if len(fileList) > 10 {
			fileList = fileList[:10]
		}
		parts = append(parts, fmt.Sprintf("Files touched: %s", strings.Join(fileList, ", ")))
	}
	if len(ticketProgress) > 0 {
		if len(ticketProgress) > 3 {
			ticketProgress = ticketProgress[len(ticketProgress)-3:]
		}
		parts = append(parts, fmt.Sprintf("Progress: %s", strings.Join(ticketProgress, "; ")))
	}
	if len(lastActions) > 0 {
		tail := lastActions
		if len(tail) > 5 {
			tail = tail[len(tail)-5:]
		}
		parts = append(parts, fmt.Sprintf("Recent actions: %s", strings.Join(tail, " → ")))
	}

	return strings.Join(parts, ". ")
}
