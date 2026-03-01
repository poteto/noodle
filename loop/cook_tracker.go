package loop

// cookTracker groups the fields that track active, adopted, and pending-review cooks.
type cookTracker struct {
	activeCooksByOrder map[string]*cookHandle
	adoptedTargets     map[string]string
	adoptedSessions    []string
	pendingReview      map[string]*pendingReviewCook
}
