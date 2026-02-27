package loop

import (
	"context"
	"time"
)

func (l *Loop) drainMergeResults(ctx context.Context) error {
	if l.mergeQueue == nil {
		return nil
	}
	deadline := time.Now().Add(20 * time.Millisecond)
	for {
		results := l.mergeQueue.DrainResults()
		for _, result := range results {
			cook := result.Cook
			if cook == nil {
				continue
			}
			if result.Err != nil {
				if conflictErr := l.handleMergeConflict(cook, result.Err); conflictErr != nil {
					l.pendingRetry[cook.orderID] = &pendingRetryCook{
						orderID:     cook.orderID,
						stageIndex:  cook.stageIndex,
						stage:       cook.stage,
						isOnFailure: cook.isOnFailure,
						orderStatus: cook.orderStatus,
						plan:        cook.plan,
						attempt:     cook.attempt + 1,
						displayName: cook.displayName,
					}
					_ = l.writePendingRetry()
					return conflictErr
				}
				continue
			}
			if err := l.advanceAndPersist(ctx, cook); err != nil {
				return err
			}
		}

		if l.mergeQueue.Pending() == 0 && l.mergeQueue.InFlight() == 0 {
			return nil
		}
		if time.Now().After(deadline) {
			return nil
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(1 * time.Millisecond):
		}
	}
}
