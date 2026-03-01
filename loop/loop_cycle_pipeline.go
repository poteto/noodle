package loop

import (
	"context"
	"strings"
	"time"

	"github.com/poteto/noodle/internal/ingest"
	"github.com/poteto/noodle/mise"
)

func (l *Loop) buildCycleBrief(ctx context.Context) (mise.Brief, []string, bool, bool, error) {
	l.refreshAdoptedTargets()
	brief, warnings, miseChanged, err := l.deps.Mise.Build(ctx, l.snapshotActiveSummary(), l.snapshotRecentHistory())
	if err != nil {
		return mise.Brief{}, warnings, false, false, err
	}
	if l.state != StateRunning && l.state != StateIdle {
		return brief, warnings, false, miseChanged, nil
	}
	if l.state == StateIdle {
		l.setState(StateRunning)
	}
	return brief, warnings, true, miseChanged, nil
}

func (l *Loop) prepareOrdersForCycle(brief mise.Brief, warnings []string, miseChanged bool) (OrdersFile, bool, error) {
	// Consume orders-next.json if the schedule session wrote one.
	promoted, emptyPromotion, err := consumeOrdersNext(l.deps.OrdersNextFile, l.deps.OrdersFile)
	if err != nil {
		payload := PromotionFailedPayload{Reason: err.Error()}
		if isOrdersNextRejectedError(err) {
			mistake := newSchedulerMistakeEnvelope(SchedulerMistakeReasonOrdersNextRejected)
			l.classifySchedulerMistake(
				"build.promote_orders_next",
				"orders-next promotion failed",
				err,
				SchedulerMistakeReasonOrdersNextRejected,
			)
			payload.AgentMistake = &mistake
			failureMetadata := eventFailureMetadataForLoop(CycleFailureClassDegradeContinue, "", &mistake)
			payload.Failure = &failureMetadata
		} else {
			l.classifyDegrade(
				"build.promote_orders_next",
				"orders-next promotion failed",
				err,
			)
			failureMetadata := eventFailureMetadataForLoop(CycleFailureClassDegradeContinue, "", nil)
			payload.Failure = &failureMetadata
		}
		l.logger.Warn("orders-next promotion failed", "error", err)
		l.lastPromotionError = err.Error()
		_ = l.events.Emit(LoopEventPromotionFailed, payload)
		// Mark promoted so the schedule order can complete and a new
		// schedule can be spawned. Without this, the schedule order
		// stays active forever and the loop deadlocks.
		l.schedulePromoted = true
	} else if promoted {
		l.logger.Info("orders-next promoted")
		l.schedulePromoted = true
		l.lastPromotionError = ""
		if emptyPromotion {
			l.scheduleNothingUntil = l.deps.Now().Add(5 * time.Minute)
			l.logger.Info("schedule produced no orders, entering cooldown")
		} else {
			l.scheduleNothingUntil = time.Time{}
		}
		if err := l.loadOrdersState(); err != nil {
			return OrdersFile{}, false, err
		}

		// Emit V2 canonical state events for each newly promoted order.
		promotedOrders, _ := l.currentOrders()
		for _, order := range promotedOrders.Orders {
			if _, exists := l.canonical.Orders[order.ID]; !exists {
				stages := make([]map[string]any, len(order.Stages))
				for i, s := range order.Stages {
					stages[i] = map[string]any{
						"stage_index": i,
						"status":      "pending",
						"skill":       s.Skill,
						"runtime":     s.Runtime,
					}
				}
				l.emitEvent(ingest.EventSchedulePromoted, map[string]any{
					"order_id": order.ID,
					"stages":   stages,
				})
			}
		}
	}

	// Reset cooldown when backlog changes (mise content changed).
	if miseChanged {
		l.scheduleNothingUntil = time.Time{}
	}

	orders, err := l.currentOrders()
	if err != nil {
		return OrdersFile{}, false, err
	}

	// Normalize and validate orders.
	normalizedOrders, changed, normErr := NormalizeAndValidateOrders(orders, l.registry, l.config)
	if normErr != nil {
		// Rebuild registry and retry on unknown task type.
		l.rebuildRegistry()
		normalizedOrders, changed, normErr = NormalizeAndValidateOrders(orders, l.registry, l.config)
		if normErr != nil {
			l.auditOrders()
			orders, err = l.currentOrders()
			if err != nil {
				return OrdersFile{}, false, err
			}
			normalizedOrders, changed, normErr = NormalizeAndValidateOrders(orders, l.registry, l.config)
			if normErr != nil {
				return OrdersFile{}, false, normErr
			}
		}
	}
	if changed {
		orders = normalizedOrders
		if err := l.writeOrdersState(orders); err != nil {
			return OrdersFile{}, false, err
		}
		l.logger.Info("orders normalized")
	}

	if hasSyncWarnings(warnings) {
		failureMetadata := eventFailureMetadataForLoop(CycleFailureClassDegradeContinue, "", nil)
		l.logger.Warn("sync script issue, continuing with empty backlog", "warnings", strings.Join(warnings, "; "))
		_ = l.events.Emit(LoopEventSyncDegraded, SyncDegradedPayload{
			Reason:  strings.Join(warnings, "; "),
			Failure: &failureMetadata,
		})
	}

	// Simplified filtering (#60): check for non-schedule orders.
	if len(l.cooks.activeCooksByOrder) == 0 && len(l.cooks.adoptedTargets) == 0 {
		if !hasNonScheduleOrders(orders) {
			// If schedule already exists, allow it to dispatch even with empty backlog.
			// Startup reconciliation injects schedule so scheduler is always available.
			if !hasScheduleOrder(orders) {
				if len(brief.Backlog) == 0 || l.scheduleNothingCooldownActive() {
					l.setState(StateIdle)
					return OrdersFile{}, false, nil
				}
				// Bootstrap only when a schedule order is truly absent.
				// Rewriting an existing schedule order creates a file-watch hot loop.
				orders = bootstrapScheduleOrder(l.config)
				if err := l.writeOrdersState(orders); err != nil {
					return OrdersFile{}, false, err
				}
				l.logger.Info("orders empty, bootstrapping schedule")
			}
		}
	}

	// Spawn schedule on mise.json change: if content changed and no schedule
	// cook is already active, inject a schedule order so the schedule agent
	// can react to new events mid-cycle.
	if miseChanged && !l.hasActiveScheduleCook() {
		if !hasScheduleOrder(orders) {
			orders.Orders = append(orders.Orders, scheduleOrder(l.config, ""))
			if err := l.writeOrdersState(orders); err != nil {
				return OrdersFile{}, false, err
			}
			l.logger.Info("mise changed, injecting schedule order")
		}
	}

	if updatedOrders, changed := ApplyOrderRoutingDefaults(orders, l.registry, l.config); changed {
		orders = updatedOrders
		if err := l.writeOrdersState(orders); err != nil {
			return OrdersFile{}, false, err
		}
	}
	l.setOrdersState(orders)
	return orders, true, nil
}

func (l *Loop) planCycleSpawns(orders OrdersFile, brief mise.Brief, capacity int) []dispatchCandidate {
	if l.mergeQueue != nil {
		threshold := l.config.Concurrency.MergeBackpressureThreshold
		if threshold > 0 && l.mergeQueue.Pending()+l.mergeQueue.InFlight() > threshold {
			return nil
		}
	}

	orderBusyTargets := busyTargets(orders)
	busySet := make(map[string]struct{}, len(orderBusyTargets)+len(l.cooks.activeCooksByOrder)+len(l.cooks.adoptedTargets))
	for targetID, busy := range orderBusyTargets {
		if busy {
			busySet[targetID] = struct{}{}
		}
	}

	adoptedSet := make(map[string]struct{}, len(l.cooks.adoptedTargets))
	for targetID := range l.cooks.adoptedTargets {
		adoptedSet[targetID] = struct{}{}
	}

	for targetID := range l.cooks.pendingReview {
		busySet[targetID] = struct{}{}
	}

	for targetID := range l.cooks.activeCooksByOrder {
		busySet[targetID] = struct{}{}
	}
	for targetID := range l.cooks.adoptedTargets {
		busySet[targetID] = struct{}{}
	}

	candidates := dispatchableStages(orders, busySet, adoptedSet, activeTicketTargetSet(brief))

	// Limit to capacity.
	limit := capacity
	if limit <= 0 {
		limit = 1
	}
	current := len(l.cooks.activeCooksByOrder) + len(l.cooks.adoptedTargets)
	available := limit - current
	if available <= 0 {
		return nil
	}
	if len(candidates) > available {
		candidates = candidates[:available]
	}
	return candidates
}

func (l *Loop) spawnPlannedCandidates(ctx context.Context, candidates []dispatchCandidate, orders OrdersFile) error {
	// Build order lookup for candidate dispatch.
	orderMap := make(map[string]Order, len(orders.Orders))
	for _, o := range orders.Orders {
		orderMap[o.ID] = o
	}
	for _, cand := range candidates {
		if l.atMaxConcurrency() {
			break
		}
		order, ok := orderMap[cand.OrderID]
		if !ok {
			continue
		}
		if err := l.spawnCook(ctx, cand, order, spawnOptions{}); err != nil {
			return err
		}
	}
	return nil
}
