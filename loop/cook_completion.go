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
	status := strings.ToLower(strings.TrimSpace(rawStatus))
	if status == "" {
		status = strings.ToLower(strings.TrimSpace(cook.session.Status()))
	}
	success := resultStatus == StageResultCompleted
	if success {
		if isScheduleStage(cook.stage) {
			l.logger.Info("schedule completed", "session", cook.session.ID())
			_ = l.events.Emit(LoopEventScheduleCompleted, ScheduleCompletedPayload{
				SessionID: cook.session.ID(),
			})
			return l.removeOrder(cook.orderID)
		}

		// Read stage_message events from the session.
		stageMsg := l.readStageMessage(cook.session.ID())
		if stageMsg != nil && stageMsg.IsBlocking() {
			l.logger.Info("stage message blocks advance", "order", cook.orderID, "session", cook.session.ID())
			l.forwardToScheduler(cook, "stage_message_blocked", stageMsg.Message)
			_ = l.parkPendingReview(cook, "blocked by stage message: "+stageMsg.Message)
			return nil
		}
		// Non-blocking message or no message — auto-advance.
		var msg *string
		if stageMsg != nil {
			msg = &stageMsg.Message
			l.forwardToScheduler(cook, "stage_message", stageMsg.Message)
		}

		canMerge := l.canMergeStage(cook.stage)

		// Emit V2 canonical stage completion event. For mergeable stages
		// this transitions canonical state to StageMerging; for non-mergeable
		// it transitions directly to StageCompleted.
		l.emitEvent(ingest.EventStageCompleted, map[string]any{
			"order_id":    cook.orderID,
			"stage_index": cook.stageIndex,
			"mergeable":   canMerge,
		})

		if !canMerge {
			// Non-mergeable stage (e.g., debate, schedule) — advance without merge.
			return l.advanceAndPersist(ctx, cook, msg)
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
			// Emit merge completion on the main goroutine after successful merge.
			l.emitEvent(ingest.EventMergeCompleted, map[string]any{
				"order_id":    cook.orderID,
				"stage_index": cook.stageIndex,
			})
			return l.advanceAndPersist(ctx, cook, msg)
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
	reason := "cook exited with status " + status
	orders, err := l.currentOrders()
	if err != nil {
		return err
	}
	orders, err = failStage(orders, cook.orderID, reason)
	if err != nil {
		return err
	}
	if err := l.writeOrdersState(orders); err != nil {
		return err
	}
	_ = l.events.Emit(LoopEventStageFailed, StageFailedPayload{
		OrderID:    cook.orderID,
		StageIndex: cook.stageIndex,
		Reason:     reason,
		SessionID:  sessionIDPtr(cook),
	})
	_ = l.events.Emit(LoopEventOrderFailed, OrderFailedPayload{
		OrderID: cook.orderID,
		Reason:  reason,
	})

	// Emit V2 canonical state event for stage failure.
	l.emitEvent(ingest.EventStageFailed, map[string]any{
		"order_id":    cook.orderID,
		"stage_index": cook.stageIndex,
		"error":       reason,
	})

	l.forwardToScheduler(cook, "stage_failed", reason)
	if strings.TrimSpace(cook.worktreeName) != "" {
		_ = l.deps.Worktree.Cleanup(cook.worktreeName, true)
	}
	return nil
}

// advanceAndPersist advances the order stage and persists the result.
// The optional message is included in the stage.completed event payload.
func (l *Loop) advanceAndPersist(ctx context.Context, cook *cookHandle, message ...*string) error {
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
func (l *Loop) forwardToScheduler(cook *cookHandle, eventType string, details string) {
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
	msg := fmt.Sprintf("[%s] order=%s stage=%d task=%s: %s",
		eventType, cook.orderID, cook.stageIndex, cook.stage.TaskKey, details)
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
