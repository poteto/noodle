package loop

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/poteto/noodle/internal/ingest"
	"github.com/poteto/noodle/internal/orderx"
	"github.com/poteto/noodle/internal/stringx"
	loopruntime "github.com/poteto/noodle/runtime"
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
	return len(l.cooks.activeCooksByOrder)+len(l.cooks.adoptedTargets) >= maxCooks
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
	for _, active := range l.cooks.activeCooksByOrder {
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
			resumePrompt = stringx.JoinNonEmpty("\n\n", hint, resumePrompt)
		}
	}
	prompt := buildCookPrompt(cand.OrderID, stage, order.Plan, order.Title, order.Rationale, resumePrompt)

	taskType, _ := l.registry.ByKey(stage.TaskKey)
	req := loopruntime.DispatchRequest{
		Name:         name,
		Prompt:       prompt,
		Provider:     nonEmpty(stage.Provider, l.config.Routing.Defaults.Provider),
		Model:        nonEmpty(stage.Model, l.config.Routing.Defaults.Model),
		Skill:        stage.Skill,
		WorktreePath: worktreePath,
		TaskKey:      taskType.Key,
		Runtime:      nonEmpty(stage.Runtime, "process"),
		DisplayName:  opts.displayName,
		Title:        order.Title,
		RetryCount:   opts.attempt,
	}
	if taskType.DomainSkill != "" {
		req.DomainSkill = taskType.DomainSkill
	}

	// Persist active status BEFORE spawning session — restart safety.
	if err := l.persistOrderStageStatus(cand.OrderID, cand.StageIndex, StageStatusActive); err != nil {
		if created {
			_ = l.deps.Worktree.Cleanup(name, true)
		}
		return err
	}

	session, fallbackOutcome, err := l.dispatchSession(ctx, req)
	if err != nil {
		return l.handleCookDispatchFailure(cand, stage, name, created, err)
	}

	displayName := strings.TrimSpace(opts.displayName)
	if displayName == "" {
		displayName = stringx.KitchenName(session.ID())
	}

	dispatchedRuntime := strings.ToLower(strings.TrimSpace(req.Runtime))
	if fallbackOutcome.Class == AgentStartFailureClassFallback {
		dispatchedRuntime = strings.ToLower(strings.TrimSpace(fallbackOutcome.SelectedRuntime))
	}
	if dispatchedRuntime == "" {
		dispatchedRuntime = "process"
	}

	cook := &cookHandle{
		cookIdentity: cookIdentity{
			orderID:    cand.OrderID,
			stageIndex: cand.StageIndex,
			stage:      stage,
			plan:       order.Plan,
		},
		orderStatus:       order.Status,
		session:           session,
		worktreeName:      name,
		worktreePath:      req.WorktreePath,
		attempt:           opts.attempt,
		generation:        l.nextDispatchGeneration(),
		displayName:       displayName,
		dispatchedRuntime: dispatchedRuntime,
	}
	l.trackCookStarted(cook)
	l.cooks.activeCooksByOrder[cand.OrderID] = cook
	l.startSessionWatcher(ctx, cook, false)

	// Emit V2 canonical state events for dispatch.
	dispatchPayload := map[string]any{
		"order_id":    cand.OrderID,
		"stage_index": cand.StageIndex,
	}
	l.emitEvent(ingest.EventDispatchRequested, dispatchPayload)
	l.emitEvent(ingest.EventDispatchCompleted, map[string]any{
		"order_id":      cand.OrderID,
		"stage_index":   cand.StageIndex,
		"session_id":    session.ID(),
		"worktree_name": name,
	})

	l.logger.Info("cook dispatched", "order", cand.OrderID, "stage", cand.StageIndex, "session", session.ID(), "worktree", name, "attempt", opts.attempt)
	return nil
}

func (l *Loop) dispatchSession(ctx context.Context, req loopruntime.DispatchRequest) (loopruntime.SessionHandle, RuntimeFallbackOutcome, error) {
	runtimeName := strings.ToLower(strings.TrimSpace(req.Runtime))
	if runtimeName == "" {
		runtimeName = strings.ToLower(strings.TrimSpace(l.config.Runtime.Default))
	}
	if runtimeName == "" {
		runtimeName = "process"
	}

	runtime := l.deps.Runtimes[runtimeName]
	if runtime == nil {
		notConfigured := newRuntimeNotConfiguredError(runtimeName)
		return nil, RuntimeFallbackOutcome{}, classifyAgentStartFailure(runtimeName, notConfigured)
	}

	req.Runtime = runtimeName
	session, err := runtime.Dispatch(ctx, req)
	if err == nil {
		return session, RuntimeFallbackOutcome{}, nil
	}
	if runtimeName != "process" {
		if fallback := l.deps.Runtimes["process"]; fallback != nil {
			l.logger.Warn("runtime dispatch failed, falling back to process", "runtime", runtimeName, "error", err)
			req.Runtime = "process"
			req.DispatchWarning = fmt.Sprintf("%s dispatch failed: %v", runtimeName, err)
			outcome := newRuntimeFallbackOutcome(
				runtimeName,
				"process",
				"runtime fallback used process dispatcher",
				err,
			)
			fallbackSession, fallbackErr := fallback.Dispatch(ctx, req)
			if fallbackErr != nil {
				return nil, outcome, classifyAgentStartFailure("process", fallbackErr)
			}
			return fallbackSession, outcome, nil
		}
	}
	return nil, RuntimeFallbackOutcome{}, classifyAgentStartFailure(runtimeName, err)
}

func (l *Loop) handleCookDispatchFailure(cand dispatchCandidate, stage Stage, worktreeName string, created bool, err error) error {
	if created {
		_ = l.deps.Worktree.Cleanup(worktreeName, true)
	}
	envelope, ok := asDispatchFailureEnvelope(err)
	if !ok {
		envelope = classifyAgentStartFailure(stage.Runtime, err)
	}
	if envelope.Class == AgentStartFailureClassRetryable {
		if persistErr := l.persistOrderStageStatus(cand.OrderID, cand.StageIndex, StageStatusPending); persistErr != nil {
			return persistErr
		}
		l.logger.Warn(
			"cook dispatch failed; stage reset to pending",
			"order", cand.OrderID,
			"stage", cand.StageIndex,
			"class", envelope.Class,
			"recoverability", envelope.Recoverability,
			"error", envelope.Cause,
		)
		return nil
	}

	reason := "dispatch failed: " + envelope.Error()
	orders, ordersErr := l.currentOrders()
	if ordersErr != nil {
		return ordersErr
	}
	orders, ordersErr = failStage(orders, cand.OrderID, reason)
	if ordersErr != nil {
		return ordersErr
	}
	if writeErr := l.writeOrdersState(orders); writeErr != nil {
		return writeErr
	}
	_ = l.events.Emit(LoopEventStageFailed, StageFailedPayload{
		OrderID:    cand.OrderID,
		StageIndex: cand.StageIndex,
		Reason:     reason,
	})
	_ = l.events.Emit(LoopEventOrderFailed, OrderFailedPayload{
		OrderID: cand.OrderID,
		Reason:  reason,
	})
	l.emitEvent(ingest.EventStageFailed, map[string]any{
		"order_id":    cand.OrderID,
		"stage_index": cand.StageIndex,
		"error":       reason,
	})
	l.forwardToScheduler(&cookHandle{
		cookIdentity: cookIdentity{
			orderID:    cand.OrderID,
			stageIndex: cand.StageIndex,
			stage:      stage,
		},
	}, "dispatch_failed", reason)
	l.classifyOrderHard(
		"cycle.dispatch_terminal",
		OrderFailureClassStageTerminal,
		cand.OrderID,
		cand.StageIndex,
		reason,
		err,
	)
	l.logger.Warn("cook dispatch failed; order marked failed",
		"order", cand.OrderID, "stage", cand.StageIndex, "error", err, "class", envelope.Class)
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

func (l *Loop) persistOrderStageStatus(orderID string, stageIndex int, status orderx.StageStatus) error {
	return l.ensureOrderStageStatus(orderID, stageIndex, status)
}
