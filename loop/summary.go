package loop

import (
	"strings"

	"github.com/poteto/noodle/internal/stringx"
	"github.com/poteto/noodle/mise"
)

const recentHistoryLimit = 20

// trackAdoptedInSummary updates activeSummary for a recovered adopted session
// so mise.json reflects orders that survived a restart.
func (l *Loop) trackAdoptedInSummary(stage *Stage) {
	if l.activeSummary.ByTaskKey == nil {
		l.activeSummary.ByTaskKey = map[string]int{}
	}
	if l.activeSummary.ByStatus == nil {
		l.activeSummary.ByStatus = map[string]int{}
	}
	if l.activeSummary.ByRuntime == nil {
		l.activeSummary.ByRuntime = map[string]int{}
	}

	taskKey := "unknown"
	runtimeName := "process"
	if stage != nil {
		if tk := strings.TrimSpace(stage.TaskKey); tk != "" {
			taskKey = tk
		}
		if rn := stringx.Normalize(stage.Runtime); rn != "" {
			runtimeName = rn
		}
	}

	l.activeSummary.Total++
	l.activeSummary.ByTaskKey[taskKey]++
	l.activeSummary.ByStatus["active"]++
	l.activeSummary.ByRuntime[runtimeName]++
}

func (l *Loop) trackCookStarted(cook *cookHandle) {
	if cook == nil {
		return
	}
	if l.activeSummary.ByTaskKey == nil {
		l.activeSummary.ByTaskKey = map[string]int{}
	}
	if l.activeSummary.ByStatus == nil {
		l.activeSummary.ByStatus = map[string]int{}
	}
	if l.activeSummary.ByRuntime == nil {
		l.activeSummary.ByRuntime = map[string]int{}
	}

	taskKey := strings.TrimSpace(cook.stage.TaskKey)
	if taskKey == "" {
		taskKey = "unknown"
	}
	runtimeName := stringx.Normalize(cook.dispatchedRuntime)
	if runtimeName == "" {
		runtimeName = stringx.Normalize(cook.stage.Runtime)
	}
	if runtimeName == "" {
		runtimeName = stringx.Normalize(l.config.Runtime.Default)
	}
	if runtimeName == "" {
		runtimeName = "process"
	}

	l.activeSummary.Total++
	l.activeSummary.ByTaskKey[taskKey]++
	l.activeSummary.ByStatus["active"]++
	l.activeSummary.ByRuntime[runtimeName]++
	cook.startedAt = l.deps.Now()
}

func (l *Loop) trackCookCompleted(cook *cookHandle, result StageResult) {
	if cook == nil {
		return
	}
	taskKey := strings.TrimSpace(cook.stage.TaskKey)
	if taskKey == "" {
		taskKey = "unknown"
	}
	runtimeName := stringx.Normalize(cook.dispatchedRuntime)
	if runtimeName == "" {
		runtimeName = stringx.Normalize(cook.stage.Runtime)
	}
	if runtimeName == "" {
		runtimeName = stringx.Normalize(l.config.Runtime.Default)
	}
	if runtimeName == "" {
		runtimeName = "process"
	}

	if l.activeSummary.Total > 0 {
		l.activeSummary.Total--
	}
	decrementSummaryBucket(l.activeSummary.ByTaskKey, taskKey)
	decrementSummaryBucket(l.activeSummary.ByStatus, "active")
	decrementSummaryBucket(l.activeSummary.ByRuntime, runtimeName)

	completedAt := result.CompletedAt
	if completedAt.IsZero() {
		completedAt = l.deps.Now()
	}
	duration := int64(0)
	if !cook.startedAt.IsZero() && completedAt.After(cook.startedAt) {
		duration = int64(completedAt.Sub(cook.startedAt).Seconds())
	}
	item := mise.HistoryItem{
		SessionID:   strings.TrimSpace(result.SessionID),
		OrderID:     cook.orderID,
		TaskKey:     taskKey,
		Status:      string(result.Status),
		DurationS:   duration,
		CompletedAt: completedAt.UTC(),
	}
	l.recentHistory = append(l.recentHistory, item)
	if len(l.recentHistory) > recentHistoryLimit {
		l.recentHistory = append([]mise.HistoryItem(nil), l.recentHistory[len(l.recentHistory)-recentHistoryLimit:]...)
	}
}

func decrementSummaryBucket(bucket map[string]int, key string) {
	if bucket == nil {
		return
	}
	current := bucket[key]
	if current <= 1 {
		delete(bucket, key)
		return
	}
	bucket[key] = current - 1
}

func (l *Loop) snapshotActiveSummary() mise.ActiveSummary {
	summary := mise.ActiveSummary{
		Total:     l.activeSummary.Total,
		ByTaskKey: make(map[string]int, len(l.activeSummary.ByTaskKey)),
		ByStatus:  make(map[string]int, len(l.activeSummary.ByStatus)),
		ByRuntime: make(map[string]int, len(l.activeSummary.ByRuntime)),
	}
	for k, v := range l.activeSummary.ByTaskKey {
		summary.ByTaskKey[k] = v
	}
	for k, v := range l.activeSummary.ByStatus {
		summary.ByStatus[k] = v
	}
	for k, v := range l.activeSummary.ByRuntime {
		summary.ByRuntime[k] = v
	}
	return summary
}

func (l *Loop) snapshotRecentHistory() []mise.HistoryItem {
	if len(l.recentHistory) == 0 {
		return []mise.HistoryItem{}
	}
	copied := make([]mise.HistoryItem, len(l.recentHistory))
	copy(copied, l.recentHistory)
	return copied
}
