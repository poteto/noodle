package loop

import "sync"

// completionBuffer groups the fields related to stage-result completion
// buffering. The channel and overflow slice work together to ensure no
// completion is lost even under high concurrency.
type completionBuffer struct {
	completions          chan StageResult
	completionOverflow   []StageResult
	completionOverflowMu sync.Mutex
	completionOverflowSaturated uint64
}
