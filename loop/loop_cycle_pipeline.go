package loop

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/poteto/noodle/internal/ingest"
	"github.com/poteto/noodle/mise"
)

const mergeBackpressureLimit = 128

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

// mergeOrdersNext reads orders-next.json, validates it, and merges into
// current orders. Returns whether promotion occurred and whether the incoming
// orders array was empty. Does NOT handle promotion side effects (cooldown,
// canonical emission, failure classification).
func (l *Loop) mergeOrdersNext() (promoted bool, emptyPromotion bool, err error) {
	return consumeOrdersNext(l.deps.OrdersNextFile, l.deps.OrdersFile)
}

// handlePromotionResult processes the side effects of an orders-next
// promotion: error classification, cooldown management, and canonical
// event emission.
func (l *Loop) handlePromotionResult(promoted, emptyPromotion bool, err error) error {
	if err != nil {
		l.handlePromotionError(err)
		return nil
	}
	if !promoted {
		return nil
	}

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
		return err
	}
	orders, err := l.currentOrders()
	if err != nil {
		return err
	}
	l.cancelSupersededActiveCooks(orders)
	l.emitPromotedOrders()
	return nil
}

func (l *Loop) cancelSupersededActiveCooks(orders OrdersFile) {
	orderByID := make(map[string]Order, len(orders.Orders))
	for _, order := range orders.Orders {
		orderByID[order.ID] = order
	}

	for orderID, cook := range l.cooks.activeCooksByOrder {
		order, ok := orderByID[orderID]
		if !ok {
			l.cancelSupersededCook(orderID, cook)
			continue
		}
		_, currentStage := activeStageForOrder(order)
		if currentStage == nil {
			l.cancelSupersededCook(orderID, cook)
			continue
		}
		if sameStageDefinition(cook.stage, *currentStage) {
			continue
		}
		l.cancelSupersededCook(orderID, cook)
	}
}

func (l *Loop) cancelSupersededCook(orderID string, cook *cookHandle) {
	if cook == nil || cook.session == nil {
		delete(l.cooks.activeCooksByOrder, orderID)
		return
	}
	l.logger.Info("order amendment superseded active stage, cancelling cook",
		"order", orderID,
		"session", cook.session.ID(),
		"stage", cook.stage.TaskKey)
	_ = cook.session.ForceKill()
	l.trackCookCompleted(cook, StageResult{
		SessionID:   cook.session.ID(),
		Status:      StageResultCancelled,
		CompletedAt: l.deps.Now(),
	})
	delete(l.cooks.activeCooksByOrder, orderID)
	l.cleanupCookWorktree(cook)
}

// handlePromotionError classifies and emits events for a failed orders-next
// promotion. Always marks schedulePromoted so the schedule order can complete.
func (l *Loop) handlePromotionError(err error) {
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
}

// emitPromotedOrders emits V2 canonical state events for each newly promoted
// order not yet tracked in canonical state.
func (l *Loop) emitPromotedOrders() {
	promotedOrders, _ := l.currentOrders()
	for _, order := range promotedOrders.Orders {
		if _, exists := l.canonical.Orders[order.ID]; exists {
			continue
		}
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

// ensureScheduleIfNeeded checks whether a schedule order needs to be
// bootstrapped or injected. Handles idle transition, empty-backlog bootstrap,
// and mise-change injection.
func (l *Loop) ensureScheduleIfNeeded(brief mise.Brief, orders *OrdersFile, miseChanged bool) (idle bool, err error) {
	if len(l.cooks.activeCooksByOrder) == 0 && len(l.cooks.adoptedTargets) == 0 && !hasNonScheduleOrders(*orders) {
		idle, err = l.bootstrapScheduleIfEmpty(brief, orders)
		if err != nil || idle {
			return idle, err
		}
	}
	if miseChanged && !l.hasActiveScheduleCook() && !hasScheduleOrder(*orders) {
		orders.Orders = append(orders.Orders, scheduleOrder(l.config, ""))
		if err := l.writeOrdersState(*orders); err != nil {
			return false, err
		}
		l.logger.Info("mise changed, injecting schedule order")
	}
	return false, nil
}

// bootstrapScheduleIfEmpty handles the case where no non-schedule orders
// exist and no cooks are active. If no schedule order exists, either
// transitions to idle or bootstraps one.
func (l *Loop) bootstrapScheduleIfEmpty(brief mise.Brief, orders *OrdersFile) (idle bool, err error) {
	if hasScheduleOrder(*orders) {
		return false, nil
	}
	if len(brief.Backlog) == 0 || l.scheduleNothingCooldownActive() {
		l.setState(StateIdle)
		return true, nil
	}
	*orders = bootstrapScheduleOrder(l.config)
	if err := l.writeOrdersState(*orders); err != nil {
		return false, err
	}
	l.logger.Info("orders empty, bootstrapping schedule")
	return false, nil
}

// applyRoutingDefaults normalizes runtime and provider defaults on orders,
// persisting if any changed.
func (l *Loop) applyRoutingDefaults(orders *OrdersFile) error {
	updated, changed := ApplyOrderRoutingDefaults(*orders, l.registry, l.config)
	if !changed {
		return nil
	}
	*orders = updated
	return l.writeOrdersState(*orders)
}

func (l *Loop) prepareOrdersForCycle(brief mise.Brief, warnings []string, miseChanged bool) (OrdersFile, bool, error) {
	promoted, emptyPromotion, err := l.mergeOrdersNext()
	if err := l.handlePromotionResult(promoted, emptyPromotion, err); err != nil {
		return OrdersFile{}, false, err
	}

	// Reset cooldown when backlog changes (mise content changed).
	if miseChanged {
		l.scheduleNothingUntil = time.Time{}
	}

	orders, err := l.currentOrders()
	if err != nil {
		return OrdersFile{}, false, err
	}

	orders, err = l.normalizeOrders(orders)
	if err != nil {
		orders, err = l.recoverFromOrdersValidationError(err)
		if err != nil {
			return OrdersFile{}, false, err
		}
	}

	l.emitSyncWarnings(warnings)

	idle, err := l.ensureScheduleIfNeeded(brief, &orders, miseChanged)
	if err != nil {
		return OrdersFile{}, false, err
	}
	if idle {
		return OrdersFile{}, false, nil
	}

	if err := l.applyRoutingDefaults(&orders); err != nil {
		return OrdersFile{}, false, err
	}
	l.setOrdersState(orders)
	return orders, true, nil
}

// normalizeOrders validates and normalizes orders, rebuilding the registry
// and auditing on repeated failures.
func (l *Loop) normalizeOrders(orders OrdersFile) (OrdersFile, error) {
	normalizedOrders, changed, normErr := NormalizeAndValidateOrders(orders, l.registry, l.config)
	if normErr != nil {
		l.rebuildRegistry()
		normalizedOrders, changed, normErr = NormalizeAndValidateOrders(orders, l.registry, l.config)
	}
	if normErr != nil {
		l.auditOrders()
		var err error
		orders, err = l.currentOrders()
		if err != nil {
			return OrdersFile{}, err
		}
		normalizedOrders, changed, normErr = NormalizeAndValidateOrders(orders, l.registry, l.config)
	}
	if normErr != nil {
		return OrdersFile{}, normErr
	}
	if !changed {
		return orders, nil
	}
	if err := l.writeOrdersState(normalizedOrders); err != nil {
		return OrdersFile{}, err
	}
	l.logger.Info("orders normalized")
	return normalizedOrders, nil
}

func (l *Loop) recoverFromOrdersValidationError(normErr error) (OrdersFile, error) {
	l.classifySchedulerMistake(
		"build.prepare_orders",
		"orders validation failed, requesting scheduler repair",
		normErr,
		SchedulerMistakeReasonOrdersNextRejected,
	)

	archivedPath := ""
	ordersData, readErr := os.ReadFile(l.deps.OrdersFile)
	if readErr == nil && len(strings.TrimSpace(string(ordersData))) > 0 {
		archivedPath = fmt.Sprintf("%s.bad.%d", l.deps.OrdersFile, l.deps.Now().UnixNano())
		if writeErr := os.WriteFile(archivedPath, ordersData, 0o644); writeErr != nil {
			l.logger.Warn("failed to archive invalid orders state", "error", writeErr)
			archivedPath = ""
		}
	}

	repairMessage := "The loop rejected scheduler-produced orders during preparation.\n" +
		"Fix this issue in your next orders-next.json output:\n" + normErr.Error()
	if archivedPath != "" {
		repairMessage += "\nInvalid orders snapshot: " + archivedPath
	}
	l.lastPromotionError = repairMessage
	l.scheduleNothingUntil = time.Time{}

	repairOrders := bootstrapScheduleOrder(l.config)
	if err := l.writeOrdersState(repairOrders); err != nil {
		return OrdersFile{}, fmt.Errorf("recover invalid orders state: %w", err)
	}

	l.logger.Warn("orders validation failed, replaced orders with schedule repair order",
		"error", normErr,
		"archived_orders", archivedPath)
	return repairOrders, nil
}

// emitSyncWarnings emits a degraded event if the sync script reported warnings.
func (l *Loop) emitSyncWarnings(warnings []string) {
	if !hasSyncWarnings(warnings) {
		return
	}
	failureMetadata := eventFailureMetadataForLoop(CycleFailureClassDegradeContinue, "", nil)
	l.logger.Warn("sync script issue, continuing with empty backlog", "warnings", strings.Join(warnings, "; "))
	_ = l.events.Emit(LoopEventSyncDegraded, SyncDegradedPayload{
		Reason:  strings.Join(warnings, "; "),
		Failure: &failureMetadata,
	})
}

func (l *Loop) planCycleSpawns(orders OrdersFile, brief mise.Brief, capacity int) []dispatchCandidate {
	if l.mergeQueue != nil {
		if l.mergeQueue.Pending()+l.mergeQueue.InFlight() > mergeBackpressureLimit {
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
