package loop

import (
	"fmt"
	"sync"
)

// MergeRequest is enqueued when a cook completes and needs its worktree merged.
type MergeRequest struct {
	OrderID        string
	StageIndex     int
	WorktreeName   string
	IsRemoteBranch bool
	BranchName     string
	Generation     uint64
}

// MergeResult is produced by the merge queue worker after processing a request.
type MergeResult struct {
	OrderID    string
	StageIndex int
	Generation uint64
	Success    bool
	Error      error
}

// MergeQueue processes worktree merges serially in a background goroutine.
// The loop enqueues requests and drains results each cycle.
type MergeQueue struct {
	worktree WorktreeManager

	mu       sync.Mutex
	pending  []MergeRequest
	results  []MergeResult
	inFlight int

	wake chan struct{}
	done chan struct{}
}

func NewMergeQueue(wt WorktreeManager) *MergeQueue {
	return &MergeQueue{
		worktree: wt,
		wake:     make(chan struct{}, 1),
		done:     make(chan struct{}),
	}
}

// Start launches the background merge worker.
func (q *MergeQueue) Start() {
	go q.worker()
}

// Close signals the worker to stop and waits for it to finish.
func (q *MergeQueue) Close() {
	close(q.done)
}

// Enqueue adds a merge request. Non-blocking.
func (q *MergeQueue) Enqueue(req MergeRequest) {
	q.mu.Lock()
	q.pending = append(q.pending, req)
	q.mu.Unlock()
	select {
	case q.wake <- struct{}{}:
	default:
	}
}

// DrainResults returns all completed merge results and clears the buffer.
func (q *MergeQueue) DrainResults() []MergeResult {
	q.mu.Lock()
	defer q.mu.Unlock()
	results := q.results
	q.results = nil
	return results
}

// Pending returns the number of enqueued + in-flight merges.
func (q *MergeQueue) Pending() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.pending) + q.inFlight
}

func (q *MergeQueue) worker() {
	for {
		select {
		case <-q.done:
			return
		case <-q.wake:
		}
		for {
			req, ok := q.dequeue()
			if !ok {
				break
			}
			result := q.processOne(req)
			q.mu.Lock()
			q.results = append(q.results, result)
			q.inFlight--
			q.mu.Unlock()
		}
	}
}

func (q *MergeQueue) dequeue() (MergeRequest, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.pending) == 0 {
		return MergeRequest{}, false
	}
	req := q.pending[0]
	q.pending = q.pending[1:]
	q.inFlight++
	return req, true
}

func (q *MergeQueue) processOne(req MergeRequest) MergeResult {
	var err error
	if req.IsRemoteBranch {
		err = q.worktree.MergeRemoteBranch(req.BranchName)
	} else {
		err = q.worktree.Merge(req.WorktreeName)
	}
	if err != nil {
		return MergeResult{
			OrderID:    req.OrderID,
			StageIndex: req.StageIndex,
			Generation: req.Generation,
			Success:    false,
			Error:      fmt.Errorf("merge %s: %w", req.WorktreeName, err),
		}
	}
	return MergeResult{
		OrderID:    req.OrderID,
		StageIndex: req.StageIndex,
		Generation: req.Generation,
		Success:    true,
	}
}
