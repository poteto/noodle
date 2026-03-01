package loop

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/poteto/noodle/adapter"
	"github.com/poteto/noodle/event"
	"github.com/poteto/noodle/internal/ingest"
	"github.com/poteto/noodle/internal/stringx"
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
	if err := l.handleCompletion(ctx, cook, result.Status, string(result.Status)); err != nil {
		if conflictErr := l.handleMergeConflict(cook, err); conflictErr != nil {
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
		_ = l.events.Emit(LoopEventBootstrapCompleted, nil)
		l.logger.Info("bootstrap completed")
		return
	}

	l.bootstrapAttempts++
	if l.bootstrapAttempts >= 3 {
		l.bootstrapExhausted = true
	}
	l.logger.Warn("bootstrap failed", "attempt", l.bootstrapAttempts, "status", string(result.Status))
}

func (l *Loop) handleCompletion(ctx context.Context, cook *cookHandle, resultStatus StageResultStatus, rawStatus string) error {
	status := stringx.Normalize(rawStatus)
	if status == "" {
		status = stringx.Normalize(cook.session.Status())
	}

	if resultStatus == StageResultCompleted {
		if isScheduleStage(cook.stage) {
			return l.handleScheduleCompletion(cook)
		}
		blocked, msg := l.processStageMessage(cook)
		if blocked {
			return nil
		}
		canMerge, err := l.worktreeHasChanges(cook)
		if err != nil {
			return l.failStage(ctx, cook, fmt.Sprintf("merge check: %v", err))
		}
		l.emitEvent(ingest.EventStageCompleted, map[string]any{
			"order_id":    cook.orderID,
			"stage_index": cook.stageIndex,
			"mergeable":   canMerge,
		})
		if canMerge {
			return l.completeWithMerge(ctx, cook, msg)
		}
		return l.completeWithoutMerge(ctx, cook, msg)
	}

	return l.handleStageFailure(ctx, cook, resultStatus, status)
}

// handleScheduleCompletion handles a successful schedule stage completion.
func (l *Loop) handleScheduleCompletion(cook *cookHandle) error {
	l.logger.Info("schedule completed", "session", cook.session.ID())
	_ = l.events.Emit(LoopEventScheduleCompleted, ScheduleCompletedPayload{
		SessionID: cook.session.ID(),
	})
	return l.removeOrder(cook.orderID)
}

// processStageMessage reads stage_message events from the session and handles
// blocking messages. Returns (blocked, message) where blocked is true if the
// stage message prevents advancement, and message is the non-blocking message
// pointer (nil if no message).
func (l *Loop) processStageMessage(cook *cookHandle) (bool, *string) {
	stageMsg := l.readStageMessage(cook.session.ID())
	if stageMsg == nil {
		return false, nil
	}
	if stageMsg.IsBlocking() {
		l.logger.Info("stage message blocks advance",
			"order", cook.orderID, "session", cook.session.ID())
		l.forwardToScheduler(cook, "stage_message_blocked", stageMsg.Message, nil)
		_ = l.parkPendingReview(cook, "blocked by stage message: "+stageMsg.Message)
		return true, nil
	}
	l.forwardToScheduler(cook, "stage_message", stageMsg.Message, nil)
	return false, &stageMsg.Message
}

// completeWithMerge handles a successful mergeable stage by persisting merge
// metadata and either directly merging or enqueueing to the merge queue.
func (l *Loop) completeWithMerge(ctx context.Context, cook *cookHandle, msg *string) error {
	mergeMode, mergeBranch := l.resolveMergeMode(cook)
	if err := l.persistMergeMetadata(cook, mergeMode, mergeBranch); err != nil {
		return err
	}
	l.logger.Info("cook completing",
		"order", cook.orderID, "session", cook.session.ID())
	if l.mergeQueue != nil {
		l.mergeQueue.Enqueue(MergeRequest{Cook: cook})
		l.logger.Info("cook queued for merge",
			"order", cook.orderID,
			"session", cook.session.ID(),
			"stage", cook.stageIndex)
		return nil
	}
	if err := l.mergeCookWorktree(ctx, cook); err != nil {
		return err
	}
	l.emitEvent(ingest.EventMergeCompleted, map[string]any{
		"order_id":    cook.orderID,
		"stage_index": cook.stageIndex,
	})
	return l.advanceAndPersist(ctx, cook, msg)
}

// completeWithoutMerge handles a successful non-mergeable stage (e.g. review)
// by advancing directly without merge.
func (l *Loop) completeWithoutMerge(ctx context.Context, cook *cookHandle, msg *string) error {
	return l.advanceAndPersist(ctx, cook, msg)
}

// handleStageFailure handles a cook that exited with a non-success status.
func (l *Loop) handleStageFailure(_ context.Context, cook *cookHandle, resultStatus StageResultStatus, status string) error {
	// Schedule cooks may write orders-next.json before exiting non-cleanly.
	if isScheduleStage(cook.stage) {
		if _, statErr := os.Stat(l.deps.OrdersNextFile); statErr == nil {
			l.logger.Info("schedule wrote orders-next before failing, treating as complete",
				"session", cook.session.ID())
			return l.removeOrder(cook.orderID)
		}
		if l.schedulePromoted {
			l.logger.Info("schedule already promoted, removing schedule order",
				"session", cook.session.ID())
			return l.removeOrder(cook.orderID)
		}
	}

	reason := "cook exited with status " + status
	if resultStatus == StageResultCancelled {
		reason = "cook cancelled with status " + status
	}
	return l.failStage(nil, cook, reason)
}

func (l *Loop) failStage(_ context.Context, cook *cookHandle, reason string) error {
	if err := l.mutateOrdersState(func(orders *OrdersFile) (bool, error) {
		updated, err := failStage(*orders, cook.orderID, reason)
		if err != nil {
			return false, err
		}
		*orders = updated
		return true, nil
	}); err != nil {
		return err
	}
	l.recordStageFailure(cook, reason, OrderFailureClassStageTerminal, nil)

	l.emitEvent(ingest.EventStageFailed, map[string]any{
		"order_id":    cook.orderID,
		"stage_index": cook.stageIndex,
		"error":       reason,
	})
	l.classifyOrderHard(
		"cycle.stage_terminal",
		OrderFailureClassStageTerminal,
		cook.orderID,
		cook.stageIndex,
		reason,
		nil,
	)

	l.forwardToScheduler(cook, "stage_failed", reason, nil)
	l.cleanupCookWorktree(cook)
	return nil
}

// advanceAndPersist advances the order stage and persists the result.
// The optional message is included in the stage.completed event payload.
func (l *Loop) advanceAndPersist(ctx context.Context, cook *cookHandle, message ...*string) error {
	removed := false
	if err := l.mutateOrdersState(func(orders *OrdersFile) (bool, error) {
		updated, orderRemoved, err := advanceOrder(*orders, cook.orderID)
		if err != nil {
			return false, err
		}
		*orders = updated
		removed = orderRemoved
		return true, nil
	}); err != nil {
		return err
	}
	var msg *string
	if len(message) > 0 {
		msg = message[0]
	}
	_ = l.events.Emit(LoopEventStageCompleted, StageCompletedPayload{
		OrderID:    cook.orderID,
		StageIndex: cook.stageIndex,
		TaskKey:    cook.stage.TaskKey,
		SessionID:  sessionIDPtr(cook),
		Message:    msg,
	})

	if removed {
		_ = l.events.Emit(LoopEventOrderCompleted, OrderCompletedPayload{
			OrderID: cook.orderID,
		})
		// Final stage of a non-failing order — fire adapter "done".
		if _, err := l.deps.Adapter.Run(ctx, "backlog", "done", adapter.RunOptions{Args: []string{cook.orderID}}); err != nil {
			if !isMissingAdapter(err) {
				return err
			}
		}
	}
	return nil
}

// forwardToScheduler sends a message to the scheduler session about an event.
// Best-effort — if the scheduler is not alive, the message is dropped and the
// order stays in the orders file for later recovery.
func (l *Loop) forwardToScheduler(cook *cookHandle, eventType string, details string, mistake *AgentMistakeEnvelope) {
	var schedulerCook *cookHandle
	for _, c := range l.cooks.activeCooksByOrder {
		if isScheduleStage(c.stage) {
			schedulerCook = c
			break
		}
	}
	if schedulerCook == nil {
		l.logger.Info("scheduler not active, event stays in orders for later recovery",
			"order", cook.orderID, "event", eventType)
		return
	}
	controller := schedulerCook.session.Controller()
	if !controller.Steerable() {
		l.logger.Info("scheduler not steerable, event stays in orders for later recovery",
			"order", cook.orderID, "event", eventType)
		return
	}
	classification := ""
	if mistake != nil {
		classification = fmt.Sprintf(" owner=%s scope=%s", mistake.Owner, mistake.Scope)
		if reason := agentMistakeReason(mistake); reason != "" {
			classification += " reason=" + reason
		}
	}
	msg := fmt.Sprintf("[%s]%s order=%s stage=%d task=%s: %s",
		eventType, classification, cook.orderID, cook.stageIndex, cook.stage.TaskKey, details)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := controller.SendMessage(ctx, msg); err != nil {
			l.logger.Warn("failed to forward to scheduler", "order", cook.orderID, "err", err)
		}
	}()
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
		if err := l.handleCompletion(ctx, cook, stageResultStatus(status), status); err != nil {
			if conflictErr := l.handleMergeConflict(cook, err); conflictErr != nil {
				return conflictErr
			}
		}
		l.dropAdoptedTarget(targetID, sessionID)
	}
	return nil
}

// readStageMessage reads the most recent stage_message event from a session's
// event log. Returns nil if no stage_message was emitted.
func (l *Loop) readStageMessage(sessionID string) *event.StageMessagePayload {
	reader := event.NewEventReader(l.runtimeDir)
	events, err := reader.ReadSession(sessionID, event.EventFilter{
		Types: map[event.EventType]struct{}{event.EventStageMessage: {}},
	})
	if err != nil || len(events) == 0 {
		return nil
	}
	last := events[len(events)-1]
	var payload event.StageMessagePayload
	if err := json.Unmarshal(last.Payload, &payload); err != nil {
		return nil
	}
	return &payload
}

// removeOrder removes an order from orders.json by ID.
func (l *Loop) removeOrder(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("remove order ID missing")
	}
	removed := false
	if err := l.mutateOrdersState(func(orders *OrdersFile) (bool, error) {
		filtered := make([]Order, 0, len(orders.Orders))
		for _, order := range orders.Orders {
			if order.ID == id {
				removed = true
				continue
			}
			filtered = append(filtered, order)
		}
		if !removed {
			return false, nil
		}
		orders.Orders = filtered
		return true, nil
	}); err != nil {
		return err
	}
	if removed {
		l.logger.Info("order removed", "order", id)
	}
	return nil
}
