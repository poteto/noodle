package loop

import (
	"context"
	"time"

	"github.com/poteto/noodle/internal/ingest"
)

func (l *Loop) drainMergeResults(ctx context.Context) error {
	if l.mergeQueue == nil {
		return nil
	}
	for {
		results := l.mergeQueue.DrainResults()
		for _, result := range results {
			cook := result.Cook
			if cook == nil {
				continue
			}
			if result.Err != nil {
				if conflictErr := l.handleMergeConflict(cook, result.Err); conflictErr != nil {
					return conflictErr
				}
				continue
			}
			// Emit V2 canonical merge completion on the main goroutine.
			if err := l.emitEventChecked(ingest.EventMergeCompleted, map[string]any{
				"order_id":    cook.orderID,
				"stage_index": cook.stageIndex,
			}); err != nil {
				return err
			}
			if err := l.advanceAndPersist(ctx, cook); err != nil {
				return err
			}
		}

		if l.mergeQueue.Pending() == 0 && l.mergeQueue.InFlight() == 0 {
			return nil
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(1 * time.Millisecond):
		}
	}
}
