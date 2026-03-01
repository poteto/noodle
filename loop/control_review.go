package loop

import (
	"context"
	"fmt"
	"strings"

	"github.com/poteto/noodle/internal/ingest"
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
		_ = l.deps.Worktree.Cleanup(pending.worktreeName, true)
	}
	// Cancel and remove the order directly.
	orders, err := l.currentOrders()
	if err != nil {
		return err
	}
	orders, err = cancelOrder(orders, orderID)
	if err != nil {
		// Order may already be gone — not fatal.
		l.logger.Warn("controlReject: cancelOrder", "error", err)
	} else {
		if err := l.writeOrdersState(orders); err != nil {
			return err
		}
	}
	sid := pending.sessionID
	mistake := newCookMistakeEnvelope(cookRejectReasonForTask(pending.stage.TaskKey), orderID, pending.stageIndex)
	failureMetadata := eventFailureMetadataForLoop(
		CycleFailureClassOrderHard,
		OrderFailureClassOrderTerminal,
		&mistake,
	)
	_ = l.events.Emit(LoopEventStageFailed, StageFailedPayload{
		OrderID:      orderID,
		StageIndex:   pending.stageIndex,
		Reason:       "rejected by user",
		SessionID:    &sid,
		AgentMistake: &mistake,
		Failure:      &failureMetadata,
	})
	_ = l.events.Emit(LoopEventOrderFailed, OrderFailedPayload{
		OrderID:      orderID,
		Reason:       "rejected by user",
		AgentMistake: &mistake,
		Failure:      &failureMetadata,
	})
	l.classifyCookMistake(
		"control.review_reject",
		OrderFailureClassOrderTerminal,
		orderID,
		pending.stageIndex,
		"rejected by user",
		mistake.CookReason,
	)
	l.forwardToScheduler(&cookHandle{cookIdentity: pending.cookIdentity}, "review_rejected", "rejected by user", &mistake)
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

	orders, err := l.currentOrders()
	if err != nil {
		return err
	}
	reason := "changes requested"
	trimmedFeedback := strings.TrimSpace(feedback)
	if trimmedFeedback != "" {
		reason += ": " + trimmedFeedback
	}
	mistake := newCookMistakeEnvelope(CookMistakeReasonRequestChanges, orderID, pending.stageIndex)
	orders, err = failStage(orders, orderID, reason)
	if err != nil {
		return err
	}
	if err := l.writeOrdersState(orders); err != nil {
		return err
	}
	sid := pending.sessionID
	failureMetadata := eventFailureMetadataForLoop(
		CycleFailureClassOrderHard,
		OrderFailureClassStageTerminal,
		&mistake,
	)
	_ = l.events.Emit(LoopEventStageFailed, StageFailedPayload{
		OrderID:      orderID,
		StageIndex:   pending.stageIndex,
		Reason:       reason,
		SessionID:    &sid,
		AgentMistake: &mistake,
		Failure:      &failureMetadata,
	})
	_ = l.events.Emit(LoopEventOrderFailed, OrderFailedPayload{
		OrderID:      orderID,
		Reason:       reason,
		AgentMistake: &mistake,
		Failure:      &failureMetadata,
	})
	l.classifyCookMistake(
		"control.request_changes",
		OrderFailureClassStageTerminal,
		orderID,
		pending.stageIndex,
		reason,
		mistake.CookReason,
	)
	l.forwardToScheduler(&cookHandle{cookIdentity: pending.cookIdentity}, "request_changes", reason, &mistake)

	// Clean up the worktree for the failed stage.
	if strings.TrimSpace(pending.worktreeName) != "" {
		_ = l.deps.Worktree.Cleanup(pending.worktreeName, true)
	}

	delete(l.cooks.pendingReview, orderID)
	return l.writePendingReview()
}
