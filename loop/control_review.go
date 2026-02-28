package loop

import (
	"context"
	"fmt"
	"strings"
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
	mergeMode, mergeBranch := l.resolveMergeMode(cook)
	if err := l.persistMergeMetadata(cook, mergeMode, mergeBranch); err != nil {
		return err
	}
	if l.mergeQueue == nil {
		if err := l.mergeCookWorktree(context.Background(), cook); err != nil {
			return err
		}
		if err := l.advanceAndPersist(context.Background(), cook); err != nil {
			return err
		}
	} else {
		l.mergeQueue.Enqueue(MergeRequest{Cook: cook})
		if err := l.drainMergeResults(context.Background()); err != nil {
			return err
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
	_ = l.events.Emit(LoopEventStageFailed, StageFailedPayload{
		OrderID:    orderID,
		StageIndex: pending.stageIndex,
		Reason:     "rejected by user",
		SessionID:  &sid,
	})
	_ = l.events.Emit(LoopEventOrderFailed, OrderFailedPayload{
		OrderID: orderID,
		Reason:  "rejected by user",
	})
	if err := l.markFailed(orderID, "rejected by user"); err != nil {
		return err
	}
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
	orders, err = failStage(orders, orderID, reason)
	if err != nil {
		return err
	}
	if err := l.writeOrdersState(orders); err != nil {
		return err
	}
	sid := pending.sessionID
	_ = l.events.Emit(LoopEventStageFailed, StageFailedPayload{
		OrderID:    orderID,
		StageIndex: pending.stageIndex,
		Reason:     reason,
		SessionID:  &sid,
	})
	_ = l.events.Emit(LoopEventOrderFailed, OrderFailedPayload{
		OrderID: orderID,
		Reason:  reason,
	})
	if err := l.markFailed(orderID, reason); err != nil {
		return err
	}

	// Clean up the worktree for the failed stage.
	if strings.TrimSpace(pending.worktreeName) != "" {
		_ = l.deps.Worktree.Cleanup(pending.worktreeName, true)
	}

	delete(l.cooks.pendingReview, orderID)
	return l.writePendingReview()
}
