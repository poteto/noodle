package loop

import (
	"context"
	"strings"
)

// runMonitorPass refreshes session meta.json files and applies monitor-side
// repairs. Best-effort: monitoring issues should not stop scheduling.
func (l *Loop) runMonitorPass(ctx context.Context) {
	if l.deps.Monitor == nil {
		return
	}
	if _, err := l.deps.Monitor.RunOnce(ctx); err != nil {
		l.logger.Warn("monitor run failed", "error", err)
	}
}

// enqueueTerminalActiveCompletions bridges monitor-derived terminal statuses
// back into loop stage completion handling when a process remains alive after
// writing a terminal canonical event.
func (l *Loop) enqueueTerminalActiveCompletions(ctx context.Context) error {
	for _, cook := range l.cooks.activeCooksByOrder {
		metaStatus, ok, err := l.readSessionStatus(cook.session.ID())
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		resultStatus, terminal := stageResultFromSessionMetaStatus(metaStatus)
		if !terminal {
			continue
		}
		l.logger.Info("session meta indicates terminal state; enqueueing completion",
			"order", cook.orderID,
			"session", cook.session.ID(),
			"status", metaStatus,
		)
		if resultStatus == StageResultCompleted && !isScheduleStage(cook.stage) {
			l.forwardToScheduler(cook, "session_repaired", "session emitted terminal result but stayed alive; auto-closing and advancing")
		}
		_ = cook.session.Kill()
		l.enqueueCompletion(ctx, StageResult{
			OrderID:      cook.orderID,
			StageIndex:   cook.stageIndex,
			Attempt:      cook.attempt,
			Status:       resultStatus,
			SessionID:    cook.session.ID(),
			Generation:   cook.generation,
			IsSchedule:   isScheduleStage(cook.stage),
			WorktreeName: cook.worktreeName,
			WorktreePath: cook.worktreePath,
			CompletedAt:  l.deps.Now(),
		})
	}
	return nil
}

func stageResultFromSessionMetaStatus(raw string) (StageResultStatus, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "completed", "exited":
		return StageResultCompleted, true
	case "failed":
		return StageResultFailed, true
	case "killed", "cancelled", "canceled", "stopped":
		return StageResultCancelled, true
	default:
		return "", false
	}
}
