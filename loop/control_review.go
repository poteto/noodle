package loop

import (
	"context"
	"fmt"
	"strings"

	"github.com/poteto/noodle/internal/ingest"
	"github.com/poteto/noodle/worktree"
)

func (l *Loop) controlMerge(orderID string) error {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return fmt.Errorf("merge: order ID empty")
	}
	pending, ok := l.cooks.pendingReview[orderID]
	if !ok {
		return fmt.Errorf("no pending review for %q", orderID)
	}

	// Merge the worktree.
	cook := &cookHandle{
		cookIdentity: pending.cookIdentity,
		orderStatus:  OrderStatusActive,
		worktreeName: pending.worktreeName,
		worktreePath: pending.worktreePath,
		session:      &adoptedSession{id: pending.sessionID, status: "completed"},
	}
	// Determine actual order status for advanceAndPersist.
	orders, err := l.currentOrders()
	if err != nil {
		return fmt.Errorf("merge: read orders: %w", err)
	}
	for _, o := range orders.Orders {
		if o.ID == orderID {
			cook.orderStatus = o.Status
			break
		}
	}

	canMerge := l.canMergeStage(cook.stage)
	l.emitEvent(ingest.EventStageCompleted, map[string]any{
		"order_id":    cook.orderID,
		"stage_index": cook.stageIndex,
		"mergeable":   canMerge,
	})

	if !canMerge {
		if err := l.advanceAndPersist(context.Background(), cook); err != nil {
			return err
		}
	} else {
		mergeMode, mergeBranch := l.resolveMergeMode(cook)
		if err := l.persistMergeMetadata(cook, mergeMode, mergeBranch); err != nil {
			return err
		}
		if l.mergeQueue == nil {
			if err := l.mergeCookWorktree(context.Background(), cook); err != nil {
				return err
			}
			l.emitEvent(ingest.EventMergeCompleted, map[string]any{
				"order_id":    cook.orderID,
				"stage_index": cook.stageIndex,
			})
			if err := l.advanceAndPersist(context.Background(), cook); err != nil {
				return err
			}
		} else {
			// Queued path: drainMergeResults emits merge_completed.
			l.mergeQueue.Enqueue(MergeRequest{Cook: cook})
			if err := l.drainMergeResults(context.Background()); err != nil {
				return err
			}
		}
	}
	delete(l.cooks.pendingReview, orderID)
	return l.writePendingReview()
}

func (l *Loop) controlReject(orderID string) error {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return fmt.Errorf("reject: order ID empty")
	}
	pending, ok := l.cooks.pendingReview[orderID]
	if !ok {
		return fmt.Errorf("no pending review for %q", orderID)
	}
	if strings.TrimSpace(pending.worktreeName) != "" {
		_ = l.deps.Worktree.Cleanup(pending.worktreeName, worktree.CleanupOpts{Force: true})
	}
	// Cancel and remove the order directly.
	if err := l.mutateOrdersState(func(orders *OrdersFile) (bool, error) {
		updated, err := cancelOrder(*orders, orderID)
		if err != nil {
			// Order may already be gone — not fatal.
			l.logger.Warn("controlReject: cancelOrder", "error", err)
			return false, nil
		}
		*orders = updated
		return true, nil
	}); err != nil {
		return err
	}
	mistake := newCookMistakeEnvelope(cookRejectReasonForTask(pending.stage.TaskKey), orderID, pending.stageIndex)
	cook := &cookHandle{
		cookIdentity: pending.cookIdentity,
		session:      &adoptedSession{id: pending.sessionID, status: "completed"},
	}
	l.recordStageFailure(cook, "rejected by user", OrderFailureClassOrderTerminal, &mistake)
	l.classifyCookMistake(
		"control.review_reject",
		OrderFailureClassOrderTerminal,
		orderID,
		pending.stageIndex,
		"rejected by user",
		mistake.CookReason,
	)
	l.forwardToScheduler(cook, "review_rejected", "rejected by user", &mistake)
	delete(l.cooks.pendingReview, orderID)
	return l.writePendingReview()
}

func (l *Loop) controlRequestChanges(orderID, feedback string) error {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return fmt.Errorf("request-changes: order ID empty")
	}
	pending, ok := l.cooks.pendingReview[orderID]
	if !ok {
		return fmt.Errorf("no pending review for %q", orderID)
	}
	if l.atMaxConcurrency() {
		l.logger.Info("request-changes deferred: at max concurrency", "order", orderID)
		return nil
	}

	reason := "changes requested"
	trimmedFeedback := strings.TrimSpace(feedback)
	if trimmedFeedback != "" {
		reason += ": " + trimmedFeedback
	}
	mistake := newCookMistakeEnvelope(CookMistakeReasonRequestChanges, orderID, pending.stageIndex)
	if err := l.mutateOrdersState(func(orders *OrdersFile) (bool, error) {
		updated, err := failStage(*orders, orderID, reason)
		if err != nil {
			return false, err
		}
		*orders = updated
		return true, nil
	}); err != nil {
		return err
	}
	cook := &cookHandle{
		cookIdentity: pending.cookIdentity,
		worktreeName: pending.worktreeName,
		session:      &adoptedSession{id: pending.sessionID, status: "completed"},
	}
	l.recordStageFailure(cook, reason, OrderFailureClassStageTerminal, &mistake)
	l.classifyCookMistake(
		"control.request_changes",
		OrderFailureClassStageTerminal,
		orderID,
		pending.stageIndex,
		reason,
		mistake.CookReason,
	)
	l.forwardToScheduler(cook, "request_changes", reason, &mistake)

	// Clean up the worktree for the failed stage.
	l.cleanupCookWorktree(cook)

	delete(l.cooks.pendingReview, orderID)
	return l.writePendingReview()
}
