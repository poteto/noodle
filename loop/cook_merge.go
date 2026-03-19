package loop

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/poteto/noodle/internal/ingest"
	"github.com/poteto/noodle/internal/stringx"
	loopruntime "github.com/poteto/noodle/runtime"
	"github.com/poteto/noodle/worktree"
)

// mergeCookWorktree merges the cook's worktree to main.
func (l *Loop) mergeCookWorktree(ctx context.Context, cook *cookHandle) error {
	syncResult, hasSyncResult, err := l.readSessionSyncResult(cook.session.ID())
	if err != nil {
		return err
	}
	if hasSyncResult && syncResult.Type == loopruntime.SyncResultTypeBranch && strings.TrimSpace(syncResult.Branch) != "" {
		if err := l.deps.Worktree.MergeRemoteBranch(syncResult.Branch); err != nil {
			return fmt.Errorf("merge remote branch %s: %w", syncResult.Branch, err)
		}
	} else {
		if err := l.deps.Worktree.Merge(cook.worktreeName); err != nil {
			return fmt.Errorf("merge %s: %w", cook.worktreeName, err)
		}
	}
	l.logger.Info("cook merged", "order", cook.orderID, "worktree", cook.worktreeName)
	_ = l.events.Emit(LoopEventWorktreeMerged, WorktreeMergedPayload{
		OrderID:      cook.orderID,
		StageIndex:   cook.stageIndex,
		WorktreeName: cook.worktreeName,
	})

	// NOTE: V2 canonical merge events are emitted by callers on the main
	// goroutine (handleCompletion / drainMergeResults), NOT here, because
	// mergeCookWorktree runs on the merge queue's background goroutine and
	// emitEvent mutates l.canonical without synchronization.

	return nil
}

// resolveMergeMode determines whether the merge uses a local worktree or a
// remote branch push. Returns the mode ("local" or "remote") and the branch
// name (empty for local merges).
func (l *Loop) resolveMergeMode(cook *cookHandle) (mode string, branch string) {
	syncResult, hasSyncResult, _ := l.readSessionSyncResult(cook.session.ID())
	if hasSyncResult && syncResult.Type == loopruntime.SyncResultTypeBranch && strings.TrimSpace(syncResult.Branch) != "" {
		return "remote", strings.TrimSpace(syncResult.Branch)
	}
	return "local", ""
}

func (l *Loop) mergeLifecyclePayload(cook *cookHandle, mergeable bool) map[string]any {
	payload := map[string]any{
		"order_id":      cook.orderID,
		"stage_index":   cook.stageIndex,
		"worktree_name": cook.worktreeName,
		"mergeable":     mergeable,
	}
	if !mergeable {
		return payload
	}
	mergeMode, mergeBranch := l.resolveMergeMode(cook)
	payload["merge_mode"] = mergeMode
	if strings.TrimSpace(mergeBranch) != "" {
		payload["merge_branch"] = mergeBranch
	}
	return payload
}

func (l *Loop) worktreeHasChanges(cook *cookHandle) (bool, error) {
	if cook == nil {
		return false, nil
	}

	// Path 1: runtime sync metadata (remote branch push).
	sessionID := ""
	if cook.session != nil {
		sessionID = cook.session.ID()
	}
	syncResult, hasSyncResult, err := l.readSessionSyncResult(sessionID)
	if err != nil {
		return false, err
	}
	if hasSyncResult && syncResult.Type == loopruntime.SyncResultTypeBranch && strings.TrimSpace(syncResult.Branch) != "" {
		return true, nil
	}

	// Path 2: local worktree branch status.
	worktreeName := strings.TrimSpace(cook.worktreeName)
	if worktreeName == "" {
		return false, nil
	}
	return l.deps.Worktree.HasUnmergedCommits(worktreeName)
}

func (l *Loop) readSessionSyncResult(sessionID string) (loopruntime.SyncResult, bool, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return loopruntime.SyncResult{}, false, nil
	}
	path := filepath.Join(l.runtimeDir, "sessions", sessionID, "spawn.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return loopruntime.SyncResult{}, false, nil
		}
		return loopruntime.SyncResult{}, false, fmt.Errorf("read spawn metadata: %w", err)
	}
	var payload struct {
		Sync loopruntime.SyncResult `json:"sync"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return loopruntime.SyncResult{}, false, fmt.Errorf("parse spawn metadata: %w", err)
	}
	if strings.TrimSpace(payload.Sync.Type) == "" && strings.TrimSpace(payload.Sync.Branch) == "" {
		return loopruntime.SyncResult{}, false, nil
	}
	payload.Sync.Type = stringx.Normalize(payload.Sync.Type)
	payload.Sync.Branch = strings.TrimSpace(payload.Sync.Branch)
	return payload.Sync, true, nil
}

func (l *Loop) handleMergeConflict(cook *cookHandle, err error) error {
	return l.handleMergeError(cook, err)
}

func (l *Loop) handleMergeError(cook *cookHandle, err error) error {
	var conflictErr *worktree.MergeConflictError
	if errors.As(err, &conflictErr) {
		if isScheduleStage(cook.stage) {
			return err
		}
		// Forward conflict to scheduler and mark the stage as failed.
		reason := "merge conflict: " + conflictErr.Error()
		l.logger.Warn("merge conflict, forwarding to scheduler", "order", cook.orderID, "reason", reason)
		l.forwardToScheduler(cook, "merge_conflict", reason, nil)
		_ = l.events.Emit(LoopEventMergeConflict, MergeConflictPayload{
			OrderID:      cook.orderID,
			StageIndex:   cook.stageIndex,
			WorktreeName: cook.worktreeName,
		})
		if err := l.emitEventChecked(ingest.EventMergeFailed, map[string]any{
			"order_id":    cook.orderID,
			"stage_index": cook.stageIndex,
			"error":       reason,
		}); err != nil {
			return err
		}

		// Park for pending review so the chef can decide.
		if parkErr := l.parkPendingReview(cook, reason); parkErr != nil {
			return parkErr
		}
		return nil
	}

	if isScheduleStage(cook.stage) {
		return err
	}
	reason := "merge failed: " + err.Error()
	schedulerHint := reason + ". Use Skill(schedule) and create a new order to fix this issue before retrying."
	l.logger.Warn("merge failed, forwarding to scheduler", "order", cook.orderID, "reason", reason)
	l.forwardToScheduler(cook, "merge_failed", schedulerHint, nil)
	if err := l.ensureCanonicalOrderFromOrders(cook.orderID); err != nil {
		return err
	}
	if err := l.emitEventChecked(ingest.EventStageFailed, map[string]any{
		"order_id":      cook.orderID,
		"stage_index":   cook.stageIndex,
		"attempt_id":    dispatchAttemptID(cook.orderID, cook.stageIndex, cook.attempt),
		"session_id":    sessionIDPtr(cook),
		"worktree_name": cook.worktreeName,
		"error":         reason,
	}); err != nil {
		return err
	}
	if err := l.mirrorLegacyOrderFromCanonical(cook.orderID); err != nil {
		return err
	}
	if err := l.syncPendingReviewProjection(); err != nil {
		return err
	}
	l.recordStageFailure(cook, reason, OrderFailureClassStageTerminal, nil)
	l.classifyOrderHard(
		"cycle.merge_terminal",
		OrderFailureClassStageTerminal,
		cook.orderID,
		cook.stageIndex,
		reason,
		err,
	)
	l.cleanupCookWorktree(cook)
	return nil
}
