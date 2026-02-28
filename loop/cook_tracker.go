package loop

// cookTracker groups the fields that track active, adopted, failed,
// and pending-review cooks.
type cookTracker struct {
	activeCooksByOrder map[string]*cookHandle
	adoptedTargets     map[string]string
	adoptedSessions    []string
	failedTargets      map[string]string
	pendingReview      map[string]*pendingReviewCook
}
