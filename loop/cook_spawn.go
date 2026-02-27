package loop

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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
			resumePrompt = joinPromptParts(hint, resumePrompt)
		}
	}
	prompt := buildCookPrompt(cand.OrderID, stage, order.Plan, order.Rationale, resumePrompt)

	taskType, _ := l.registry.ByKey(stage.TaskKey)
	req := loopruntime.DispatchRequest{
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

	session, err := l.dispatchSession(ctx, req)
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
		cookIdentity: cookIdentity{
			orderID:    cand.OrderID,
			stageIndex: cand.StageIndex,
			stage:      stage,
			plan:       order.Plan,
		},
		isOnFailure:  cand.IsOnFailure,
		orderStatus:  order.Status,
		session:      session,
		worktreeName: name,
		worktreePath: req.WorktreePath,
		attempt:      opts.attempt,
		generation:   l.nextDispatchGeneration(),
		displayName:  displayName,
	}
	l.trackCookStarted(cook)
	l.cooks.activeCooksByOrder[cand.OrderID] = cook
	l.startSessionWatcher(ctx, cook, false)
	l.logger.Info("cook dispatched", "order", cand.OrderID, "stage", cand.StageIndex, "session", session.ID(), "worktree", name, "attempt", opts.attempt)
	return nil
}

func (l *Loop) dispatchSession(ctx context.Context, req loopruntime.DispatchRequest) (loopruntime.SessionHandle, error) {
	runtimeName := strings.ToLower(strings.TrimSpace(req.Runtime))
	if runtimeName == "" {
		runtimeName = strings.ToLower(strings.TrimSpace(l.config.Runtime.Default))
	}
	if runtimeName == "" {
		runtimeName = "tmux"
	}

	runtime := l.deps.Runtimes[runtimeName]
	if runtime == nil {
		return nil, fmt.Errorf("runtime %q not configured", runtimeName)
	}

	session, err := runtime.Dispatch(ctx, req)
	if err == nil {
		return session, nil
	}
	if runtimeName != "tmux" {
		if fallback := l.deps.Runtimes["tmux"]; fallback != nil {
			req.Runtime = "tmux"
			req.DispatchWarning = fmt.Sprintf("%s dispatch failed: %v", runtimeName, err)
			return fallback.Dispatch(ctx, req)
		}
	}
	return nil, err
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

func (l *Loop) persistOrderStageStatus(orderID string, stageIndex int, isOnFailure bool, status orderx.StageStatus) error {
	return l.ensureOrderStageStatus(orderID, stageIndex, isOnFailure, status)
}
