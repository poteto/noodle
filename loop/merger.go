package loop

import (
	"context"
	"sync"
)

type MergeRequest struct {
	Cook *cookHandle
}

type MergeResult struct {
	Cook *cookHandle
	Err  error
}

type MergeQueue struct {
	mergeFn func(context.Context, MergeRequest) error

	mu       sync.Mutex
	requests []MergeRequest
	results  []MergeResult
	pending  int
	inFlight int

	wakeCh    chan struct{}
	doneCh    chan struct{}
	ctx       context.Context
	cancel    context.CancelFunc
	closeOnce sync.Once
}

func NewMergeQueue(parent context.Context, mergeFn func(context.Context, MergeRequest) error) *MergeQueue {
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithCancel(parent)
	q := &MergeQueue{
		mergeFn:  mergeFn,
		requests: make([]MergeRequest, 0),
		results:  make([]MergeResult, 0),
		wakeCh:   make(chan struct{}, 1),
		doneCh:   make(chan struct{}),
		ctx:      ctx,
		cancel:   cancel,
	}
	go q.run()
	return q
}

func (q *MergeQueue) run() {
	defer close(q.doneCh)
	for {
		req, ok := q.nextRequest()
		if !ok {
			select {
			case <-q.ctx.Done():
				return
			case <-q.wakeCh:
				continue
			}
		}
		err := q.mergeFn(q.ctx, req)
		q.completeRequest(req, err)
	}
}

func (q *MergeQueue) nextRequest() (MergeRequest, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.requests) == 0 {
		return MergeRequest{}, false
	}
	req := q.requests[0]
	q.requests = q.requests[1:]
	if q.pending > 0 {
		q.pending--
	}
	q.inFlight++
	return req, true
}

func (q *MergeQueue) completeRequest(req MergeRequest, err error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.inFlight > 0 {
		q.inFlight--
	}
	q.results = append(q.results, MergeResult{Cook: req.Cook, Err: err})
}

func (q *MergeQueue) Enqueue(req MergeRequest) {
	q.mu.Lock()
	q.requests = append(q.requests, req)
	q.pending++
	q.mu.Unlock()
	select {
	case q.wakeCh <- struct{}{}:
	default:
	}
}

func (q *MergeQueue) DrainResults() []MergeResult {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.results) == 0 {
		return nil
	}
	out := append([]MergeResult(nil), q.results...)
	q.results = q.results[:0]
	return out
}

func (q *MergeQueue) Pending() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.pending
}

func (q *MergeQueue) InFlight() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.inFlight
}

func (q *MergeQueue) Close() {
	q.closeOnce.Do(func() {
		q.cancel()
		<-q.doneCh
	})
}
