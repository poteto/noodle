package loop

import (
	"context"
	"sync"
)

// completionBuffer groups the fields related to stage-result completion
// buffering. The channel and overflow slice work together to ensure no
// completion is lost even under high concurrency.
type completionBuffer struct {
	completions                 chan StageResult
	completionOverflow          []StageResult
	completionOverflowMu        sync.Mutex
	completionOverflowSaturated uint64
}

func (l *Loop) enqueueCompletion(ctx context.Context, result StageResult) {
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case l.completionBuf.completions <- result:
		return
	default:
	}

	overflowCap := cap(l.completionBuf.completionOverflow)
	if overflowCap <= 0 {
		overflowCap = completionBufferSize
	}

	l.completionBuf.completionOverflowMu.Lock()
	if len(l.completionBuf.completionOverflow) < overflowCap {
		l.completionBuf.completionOverflow = append(l.completionBuf.completionOverflow, result)
		l.completionBuf.completionOverflowMu.Unlock()
		return
	}
	l.completionBuf.completionOverflowSaturated++
	l.completionBuf.completionOverflowMu.Unlock()

	select {
	case l.completionBuf.completions <- result:
	case <-ctx.Done():
	}
}

func (l *Loop) takeCompletionOverflow() []StageResult {
	l.completionBuf.completionOverflowMu.Lock()
	defer l.completionBuf.completionOverflowMu.Unlock()
	if len(l.completionBuf.completionOverflow) == 0 {
		return nil
	}
	drained := append([]StageResult(nil), l.completionBuf.completionOverflow...)
	l.completionBuf.completionOverflow = l.completionBuf.completionOverflow[:0]
	return drained
}
