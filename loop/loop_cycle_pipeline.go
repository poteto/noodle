package loop

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/poteto/noodle/internal/dispatch"
	"github.com/poteto/noodle/mise"
)

const mergeBackpressureLimit = 128

func (l *Loop) buildCycleBrief(ctx context.Context) (mise.Brief, []string, bool, error) {
	l.refreshAdoptedTargets()
	brief, warnings, err := l.deps.Mise.Build(ctx, l.snapshotActiveSummary(), l.snapshotRecentHistory())
	if err != nil {
		return mise.Brief{}, warnings, false, err
	}
	if l.state != StateRunning && l.state != StateIdle {
		return brief, warnings, false, nil
	}
	if l.state == StateIdle {
		l.setState(StateRunning)
	}
	return brief, warnings, true, nil
}

// mergeOrdersNext reads orders-next.json, validates it, and merges into
// current orders. Returns whether promotion occurred and whether the incoming
// orders array was empty. Does NOT handle promotion side effects (decision
// memoization, canonical emission, failure classification).
func (l *Loop) mergeOrdersNext() (mergeResult, error) {
	orders, err := l.currentOrders()
	if err != nil {
		return mergeResult{}, err
	}
	return consumeOrdersNext(l.deps.OrdersNextFile, orders)
}

// handlePromotionResult processes the side effects of an orders-next
// promotion: error classification, decision memoization, and canonical
// event emission.
func (l *Loop) handlePromotionResult(result mergeResult, err error) error {
	if err != nil {
		l.handlePromotionError(err)
		return nil
	}
	if !result.Promoted {
		return nil
	}

	l.logger.Info("orders-next promoted")
	l.schedulePromoted = true
	l.lastPromotionError = ""
	// Memoize the decision against the state the schedule session was
	// dispatched with, not the state at promotion time: a backlog change
	// that landed while the session was running must still get its own
	// schedule dispatch.
	l.scheduleDecidedDigest = l.scheduleDispatchDigest
	if result.EmptyPromotion {
		l.logger.Info("schedule produced no orders, memoizing empty decision",
			"digest", l.scheduleDecidedDigest)
	}
	if err := l.writeOrdersState(result.Orders); err != nil {
		l.handlePromotionError(err)
		return nil
	}
	if err := os.Remove(l.deps.OrdersNextFile); err != nil && !os.IsNotExist(err) {
		l.handlePromotionError(fmt.Errorf("remove orders-next.json: %w", err))
		return nil
	}
	if err := l.cancelSupersededActiveCooks(result.Orders); err != nil {
		l.handlePromotionError(err)
		return nil
	}
	for _, order := range result.Orders.Orders {
		l.syncCanonicalOrderFromLegacy(order)
	}
	if err := l.persistCanonicalCheckpoint(); err != nil {
		l.handlePromotionError(err)
		return nil
	}
	l.emitPromotedOrders()
	return nil
}

func (l *Loop) cancelSupersededActiveCooks(orders OrdersFile) error {
	orderByID := make(map[string]Order, len(orders.Orders))
	for _, order := range orders.Orders {
		orderByID[order.ID] = order
	}

	for orderID, cook := range l.cooks.activeCooksByOrder {
		order, ok := orderByID[orderID]
		if !ok {
			if err := l.cancelSupersededCook(orderID, cook, false); err != nil {
				return err
			}
			continue
		}
		_, currentStage := activeStageForOrder(order)
		if currentStage == nil {
			if err := l.cancelSupersededCook(orderID, cook, false); err != nil {
				return err
			}
			continue
		}
		if sameStageDefinition(cook.stage, *currentStage) {
			continue
		}
		if err := l.cancelSupersededCook(orderID, cook, true); err != nil {
			return err
		}
	}
	return nil
}

func (l *Loop) cancelSupersededCook(orderID string, cook *cookHandle, resume bool) error {
	if cook == nil || cook.session == nil {
		delete(l.cooks.activeCooksByOrder, orderID)
		return nil
	}
	l.logger.Info("order amendment superseded active stage, cancelling cook",
		"order", orderID,
		"session", cook.session.ID(),
		"stage", cook.stage.TaskKey)
	if err := cook.session.ForceKill(); err != nil {
		l.logger.Warn("cancel superseded cook force kill failed",
			"order", orderID,
			"session", cook.session.ID(),
			"error", err)
	}
	l.trackCookCompleted(cook, StageResult{
		SessionID:   cook.session.ID(),
		Status:      StageResultCancelled,
		CompletedAt: l.deps.Now(),
	})
	delete(l.cooks.activeCooksByOrder, orderID)
	if resume {
		if err := l.ensureOrderStageStatus(orderID, cook.stageIndex, StageStatusPending); err != nil {
			return err
		}
		orders, err := l.currentOrders()
		if err != nil {
			return err
		}
		for _, order := range orders.Orders {
			if order.ID != orderID {
				continue
			}
			l.syncCanonicalOrderFromLegacy(order)
			if err := l.persistCanonicalCheckpoint(); err != nil {
				return err
			}
			break
		}
	}
	l.cleanupCookWorktree(cook)
	return nil
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
	// No decision was made — drop the memo so the scheduler is re-dispatched
	// with the repair message.
	l.scheduleDecidedDigest = ""
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
		l.trackCanonicalOrder(order)
	}
}

// ensureScheduleIfNeeded checks whether a schedule order needs to be
// bootstrapped or injected. Handles idle transition, empty-backlog bootstrap,
// and decision-state-change injection.
//
// digest is the decision-relevant state digest for this cycle and changed
// reports whether it moved since the previous cycle. While the digest equals
// the one the last schedule decision was memoized against, no new schedule
// session is spawned — that decision still holds.
func (l *Loop) ensureScheduleIfNeeded(brief mise.Brief, orders *OrdersFile, digest string, changed bool) (idle bool, err error) {
	decided := digest == l.scheduleDecidedDigest
	if len(l.cooks.activeCooksByOrder) == 0 && len(l.cooks.adoptedTargets) == 0 && !hasNonScheduleOrders(*orders) {
		idle, err = l.bootstrapScheduleIfEmpty(brief, orders, decided)
		if err != nil || idle {
			return idle, err
		}
	}
	if changed && !decided && !l.hasActiveScheduleCook() && !hasScheduleOrder(*orders) {
		orders.Orders = append(orders.Orders, scheduleOrder(l.config, ""))
		if err := l.writeOrdersState(*orders); err != nil {
			return false, err
		}
		l.logger.Info("decision state changed, injecting schedule order", "digest", digest)
	}
	return false, nil
}

// bootstrapScheduleIfEmpty handles the case where no non-schedule orders
// exist and no cooks are active. If no schedule order exists, either
// transitions to idle or bootstraps one.
func (l *Loop) bootstrapScheduleIfEmpty(brief mise.Brief, orders *OrdersFile, decided bool) (idle bool, err error) {
	if hasScheduleOrder(*orders) {
		return false, nil
	}
	if len(brief.Backlog) == 0 || decided {
		l.setState(StateIdle)
		return true, nil
	}
	*orders = bootstrapScheduleOrder(l.config)
	if err := l.writeOrdersState(*orders); err != nil {
		return false, err
	}
	if len(orders.Orders) > 0 {
		l.trackCanonicalOrder(orders.Orders[len(orders.Orders)-1])
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

func (l *Loop) prepareOrdersForCycle(brief mise.Brief, warnings []string) (OrdersFile, bool, error) {
	result, err := l.mergeOrdersNext()
	if err := l.handlePromotionResult(result, err); err != nil {
		return OrdersFile{}, false, err
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

	digest, err := scheduleDecisionDigest(brief, orders)
	if err != nil {
		return OrdersFile{}, false, err
	}
	// The first cycle only seeds the digest: startup is responsible for
	// injecting the initial schedule order, not this edge.
	digestChanged := l.decisionDigest != "" && digest != l.decisionDigest
	l.decisionDigest = digest

	idle, err := l.ensureScheduleIfNeeded(brief, &orders, digest, digestChanged)
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
	l.scheduleDecidedDigest = ""

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

func (l *Loop) planCycleSpawns(orders OrdersFile, brief mise.Brief, capacity int) ([]dispatchCandidate, error) {
	if !l.canonicalLoaded {
		if err := l.loadOrBootstrapCanonical(); err != nil {
			return nil, err
		}
	}
	if l.mergeQueue != nil {
		if l.mergeQueue.Pending()+l.mergeQueue.InFlight() > mergeBackpressureLimit {
			return nil, nil
		}
	}

	blockedOrders := make(map[string]string, len(l.cooks.pendingReview)+len(brief.Tickets))
	for targetID := range activeTicketTargetSet(brief) {
		blockedOrders[targetID] = "ticketed"
	}
	for targetID := range l.cooks.pendingReview {
		blockedOrders[targetID] = "pending_review"
	}

	plan := dispatch.PlanDispatches(l.canonical, capacity, blockedOrders)
	if len(plan.Candidates) == 0 {
		return nil, nil
	}

	orderMap := make(map[string]Order, len(orders.Orders))
	for _, order := range orders.Orders {
		orderMap[order.ID] = order
	}

	candidates := make([]dispatchCandidate, 0, len(plan.Candidates))
	for _, candidate := range plan.Candidates {
		order, ok := orderMap[candidate.OrderID]
		if !ok {
			return nil, fmt.Errorf("canonical dispatch candidate %q missing from orders state", candidate.OrderID)
		}
		if candidate.StageIndex < 0 || candidate.StageIndex >= len(order.Stages) {
			return nil, fmt.Errorf("canonical dispatch candidate %q stage %d missing from orders state", candidate.OrderID, candidate.StageIndex)
		}
		candidates = append(candidates, dispatchCandidate{
			OrderID:    candidate.OrderID,
			StageIndex: candidate.StageIndex,
			Stage:      order.Stages[candidate.StageIndex],
		})
	}
	return candidates, nil
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
