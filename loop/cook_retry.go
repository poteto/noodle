package loop

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/poteto/noodle/internal/recover"
)

func (l *Loop) readSessionStatus(sessionID string) (string, bool, error) {
	metaPath := filepath.Join(l.runtimeDir, "sessions", sessionID, "meta.json")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, err
	}
	var payload struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return "", false, err
	}
	return strings.ToLower(strings.TrimSpace(payload.Status)), true, nil
}

func (l *Loop) buildAdoptedCook(targetID string, sessionID string, status string) (*cookHandle, bool, error) {
	// Try orders-based lookup first.
	orders, err := l.currentOrders()
	if err != nil {
		return nil, false, err
	}
	for _, order := range orders.Orders {
		if order.ID != targetID {
			continue
		}
		idx, stg := activeStageForOrder(order)
		if idx < 0 || stg == nil {
			return nil, false, nil
		}
		name := cookBaseName(order.ID, idx, stg.TaskKey)
		worktreePath := l.worktreePath(name)
		return &cookHandle{
			cookIdentity: cookIdentity{
				orderID:    order.ID,
				stageIndex: idx,
				stage:      *stg,
				plan:       order.Plan,
			},
			isOnFailure: order.Status == OrderStatusFailing,
			orderStatus: order.Status,
			session: &adoptedSession{
				id:     sessionID,
				status: status,
			},
			worktreeName: name,
			worktreePath: worktreePath,
			attempt:      recover.RecoveryChainLength(name),
		}, true, nil
	}

	return nil, false, nil
}

func (l *Loop) dropAdoptedTarget(targetID string, sessionID string) {
	delete(l.cooks.adoptedTargets, targetID)
	filtered := l.cooks.adoptedSessions[:0]
	for _, id := range l.cooks.adoptedSessions {
		if id == sessionID {
			continue
		}
		filtered = append(filtered, id)
	}
	l.cooks.adoptedSessions = filtered
}

func (l *Loop) processPendingRetries(ctx context.Context) error {
	if len(l.cooks.pendingRetry) == 0 {
		return nil
	}
	pending := l.cooks.pendingRetry
	l.cooks.pendingRetry = map[string]*pendingRetryCook{}
	for _, p := range pending {
		if l.atMaxConcurrency() {
			l.cooks.pendingRetry[p.orderID] = p
			continue
		}
		cand := dispatchCandidate{
			OrderID:     p.orderID,
			StageIndex:  p.stageIndex,
			Stage:       p.stage,
			IsOnFailure: p.isOnFailure,
		}
		order := Order{
			ID:     p.orderID,
			Status: p.orderStatus,
			Plan:   p.plan,
			Stages: []Stage{p.stage},
		}
		if err := l.spawnCook(ctx, cand, order, spawnOptions{
			attempt:     p.attempt,
			displayName: p.displayName,
		}); err != nil {
			if p.attempt >= l.config.Recovery.MaxRetries {
				fmt.Fprintf(os.Stderr, "loop.pending-retry: %s exhausted retries: %v\n", p.orderID, err)
				if markErr := l.markFailed(p.orderID, err.Error()); markErr != nil {
					fmt.Fprintf(os.Stderr, "loop.pending-retry: mark failed %s: %v\n", p.orderID, markErr)
				}
				continue
			}
			l.cooks.pendingRetry[p.orderID] = &pendingRetryCook{
				cookIdentity: p.cookIdentity,
				isOnFailure:  p.isOnFailure,
				orderStatus:  p.orderStatus,
				attempt:      p.attempt + 1,
				displayName:  p.displayName,
			}
			continue
		}
	}
	// Persist after processing — captures both removals (dispatched) and
	// re-additions (still deferred or bumped attempt).
	_ = l.writePendingRetry()
	return nil
}

func (l *Loop) retryCook(ctx context.Context, cook *cookHandle, reason string) error {
	nextAttempt := cook.attempt + 1
	info, err := recover.CollectRecoveryInfo(ctx, l.runtimeDir, cook.session.ID())
	if err != nil {
		info = recover.RecoveryInfo{SessionID: cook.session.ID(), ExitReason: reason}
	}
	resolvedReason := retryFailureReason(reason, info)
	if nextAttempt > l.config.Recovery.MaxRetries {
		if isScheduleStage(cook.stage) {
			return fmt.Errorf("schedule failed after retries: %s", resolvedReason)
		}
		l.logger.Warn("cook failed permanently", "order", cook.orderID, "session", cook.session.ID(), "reason", resolvedReason)
		if err := l.failAndPersist(cook, resolvedReason); err != nil {
			return err
		}
		return nil
	}
	l.logger.Info("cook retrying", "order", cook.orderID, "session", cook.session.ID(), "attempt", nextAttempt, "reason", resolvedReason)

	if l.atMaxConcurrency() {
		l.cooks.pendingRetry[cook.orderID] = &pendingRetryCook{
			cookIdentity: cook.cookIdentity,
			isOnFailure:  cook.isOnFailure,
			orderStatus:  cook.orderStatus,
			attempt:      nextAttempt,
			displayName:  cook.displayName,
		}
		_ = l.writePendingRetry()
		l.logger.Info("retry deferred: at max concurrency", "order", cook.orderID, "attempt", nextAttempt)
		return nil
	}

	if strings.TrimSpace(info.ExitReason) == "" {
		info.ExitReason = resolvedReason
	}
	resume := recover.BuildResumeContext(info, nextAttempt, l.config.Recovery.MaxRetries)
	cand := dispatchCandidate{
		OrderID:     cook.orderID,
		StageIndex:  cook.stageIndex,
		Stage:       cook.stage,
		IsOnFailure: cook.isOnFailure,
	}
	order := Order{
		ID:        cook.orderID,
		Status:    cook.orderStatus,
		Plan:      cook.plan,
		Rationale: "",
		Stages:    []Stage{cook.stage},
	}
	return l.spawnCook(ctx, cand, order, spawnOptions{
		attempt:     nextAttempt,
		resume:      resume.Summary,
		displayName: cook.displayName,
	})
}

func retryFailureReason(base string, info recover.RecoveryInfo) string {
	base = strings.TrimSpace(base)
	if base == "" {
		base = "cook failed"
	}

	exitReason := strings.TrimSpace(info.ExitReason)
	if exitReason == "" {
		return base
	}
	if strings.EqualFold(exitReason, "session exited without explicit reason") {
		return base
	}

	if strings.HasPrefix(strings.ToLower(base), "cook exited with status") {
		return exitReason
	}
	return base
}
