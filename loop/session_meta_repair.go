package loop

import (
	"context"

	"github.com/poteto/noodle/internal/stringx"
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
			l.forwardToScheduler(cook, "session_repaired", "session emitted terminal result but stayed alive; auto-closing and advancing", nil)
		}
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
		_ = cook.session.ForceKill()
	}
	if l.bootstrapInFlight != nil {
		cook := l.bootstrapInFlight
		metaStatus, ok, err := l.readSessionStatus(cook.session.ID())
		if err != nil {
			return err
		}
		if ok {
			resultStatus, terminal := stageResultFromSessionMetaStatus(metaStatus)
			if terminal {
				l.logger.Info("bootstrap session meta indicates terminal state; enqueueing completion",
					"session", cook.session.ID(),
					"status", metaStatus,
				)
				l.enqueueCompletion(ctx, StageResult{
					OrderID:      cook.orderID,
					StageIndex:   cook.stageIndex,
					Attempt:      cook.attempt,
					Status:       resultStatus,
					SessionID:    cook.session.ID(),
					Generation:   cook.generation,
					IsSchedule:   isScheduleStage(cook.stage),
					IsBootstrap:  true,
					WorktreeName: cook.worktreeName,
					WorktreePath: cook.worktreePath,
					CompletedAt:  l.deps.Now(),
				})
				_ = cook.session.ForceKill()
			}
		}
	}
	return nil
}

func stageResultFromSessionMetaStatus(raw string) (StageResultStatus, bool) {
	switch stringx.Normalize(raw) {
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
