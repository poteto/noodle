package loop

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/poteto/noodle/adapter"
)

func (l *Loop) drainCompletions(ctx context.Context) error {
	drainedAny := false
	for {
		select {
		case result := <-l.completionBuf.completions:
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
		case result := <-l.completionBuf.completions:
			if err := l.applyStageResult(waitCtx, result); err != nil {
				return err
			}
			for {
				select {
				case late := <-l.completionBuf.completions:
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
	cook, exists := l.cooks.activeCooksByOrder[result.OrderID]
	if !exists {
		return nil
	}
	if cook.generation != result.Generation {
		return nil
	}
	l.trackCookCompleted(cook, result)
	delete(l.cooks.activeCooksByOrder, cook.orderID)
	if err := l.handleCompletion(ctx, cook); err != nil {
		if conflictErr := l.handleMergeConflict(cook, err); conflictErr != nil {
			l.cooks.pendingRetry[cook.orderID] = &pendingRetryCook{
				cookIdentity: cook.cookIdentity,
				isOnFailure:  cook.isOnFailure,
				orderStatus:  cook.orderStatus,
				attempt:      cook.attempt + 1,
				displayName:  cook.displayName,
			}
			_ = l.writePendingRetry()
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
	cook := l.bootstrapInFlight
	l.trackCookCompleted(cook, result)
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

		mergeMode, mergeBranch := l.resolveMergeMode(cook)
		if err := l.persistMergeMetadata(cook, mergeMode, mergeBranch); err != nil {
			return err
		}
		l.logger.Info("cook completing", "order", cook.orderID, "session", cook.session.ID())
		if l.mergeQueue == nil {
			if err := l.mergeCookWorktree(ctx, cook); err != nil {
				return err
			}
			return l.advanceAndPersist(ctx, cook)
		}
		l.mergeQueue.Enqueue(MergeRequest{Cook: cook})
		l.logger.Info("cook queued for merge", "order", cook.orderID, "session", cook.session.ID(), "stage", cook.stageIndex)
		return nil
	}
	// Schedule cooks may write orders-next.json before exiting non-cleanly
	// (e.g., codex validation step fails after the file is written). If the
	// deliverable exists or was already promoted, treat the schedule as done.
	if isScheduleStage(cook.stage) {
		if _, statErr := os.Stat(l.deps.OrdersNextFile); statErr == nil {
			l.logger.Info("schedule wrote orders-next before failing, treating as complete", "session", cook.session.ID())
			return l.removeOrder(cook.orderID)
		}
		// consumeOrdersNext sets schedulePromoted when it promotes
		// orders-next.json. Only treat the schedule as done if we know
		// the promotion actually happened — pre-existing non-schedule
		// orders should not suppress a retry.
		if l.schedulePromoted {
			l.logger.Info("schedule already promoted, removing schedule order", "session", cook.session.ID())
			return l.removeOrder(cook.orderID)
		}
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
	orders, err := l.currentOrders()
	if err != nil {
		return err
	}
	orders, removed, err := advanceOrder(orders, cook.orderID)
	if err != nil {
		return err
	}
	if err := l.writeOrdersState(orders); err != nil {
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
	orders, err := l.currentOrders()
	if err != nil {
		return err
	}
	orders, terminal, err := failStage(orders, cook.orderID, reason)
	if err != nil {
		return err
	}
	if err := l.writeOrdersState(orders); err != nil {
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

func (l *Loop) collectAdoptedCompletions(ctx context.Context) error {
	for targetID, sessionID := range l.cooks.adoptedTargets {
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

// removeOrder removes an order from orders.json by ID.
func (l *Loop) removeOrder(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("remove requires order ID")
	}
	orders, err := l.currentOrders()
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
	if err := l.writeOrdersState(orders); err != nil {
		return err
	}
	l.logger.Info("order removed", "order", id)
	return nil
}
