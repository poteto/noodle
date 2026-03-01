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
	"github.com/poteto/noodle/internal/taskreg"
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
		if err := l.deps.Worktree.Merge(cook.worktreeName, ""); err != nil {
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

// Merge metadata Extra keys.
const (
	mergeExtraWorktree   = "merge_worktree"
	mergeExtraBranch     = "merge_branch"
	mergeExtraGeneration = "merge_generation"
	mergeExtraMode       = "merge_mode"
)

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

// persistMergeMetadata writes merge-related fields into the stage's Extra map
// and sets the status to "merging" atomically. On crash recovery, reconcile
// reads these fields to decide how to resume or fail the merge.
func (l *Loop) persistMergeMetadata(cook *cookHandle, mode string, branch string) error {
	return l.mutateOrdersState(func(orders *OrdersFile) (bool, error) {
		for i := range orders.Orders {
			if orders.Orders[i].ID != cook.orderID {
				continue
			}
			stages := &orders.Orders[i].Stages
			if cook.stageIndex < 0 || cook.stageIndex >= len(*stages) {
				return false, nil
			}
			s := &(*stages)[cook.stageIndex]
			if s.Extra == nil {
				s.Extra = make(map[string]json.RawMessage)
			}
			s.Extra[mergeExtraWorktree] = jsonQuote(cook.worktreeName)
			s.Extra[mergeExtraBranch] = jsonQuote(branch)
			s.Extra[mergeExtraGeneration] = jsonQuote(fmt.Sprintf("%d", cook.generation))
			s.Extra[mergeExtraMode] = jsonQuote(mode)
			s.Status = StageStatusMerging
			return true, nil
		}
		return true, nil
	})
}

// jsonQuote returns a JSON-encoded string value as json.RawMessage.
func jsonQuote(s string) json.RawMessage {
	b, _ := json.Marshal(s)
	return json.RawMessage(b)
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
		// Emit V2 canonical state event for merge failure.
		l.emitEvent(ingest.EventMergeFailed, map[string]any{
			"order_id":    cook.orderID,
			"stage_index": cook.stageIndex,
			"error":       reason,
		})

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

	orders, ordersErr := l.currentOrders()
	if ordersErr != nil {
		return ordersErr
	}
	orders, ordersErr = failStage(orders, cook.orderID, reason)
	if ordersErr != nil {
		return ordersErr
	}
	if writeErr := l.writeOrdersState(orders); writeErr != nil {
		return writeErr
	}
	l.recordStageFailure(cook, reason, OrderFailureClassStageTerminal, nil)
	l.emitEvent(ingest.EventMergeFailed, map[string]any{
		"order_id":    cook.orderID,
		"stage_index": cook.stageIndex,
		"error":       reason,
	})
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
