package loop

import (
	"strings"

	"github.com/poteto/noodle/internal/taskreg"
	"github.com/poteto/noodle/mise"
)

// TaskType is the canonical registry entry for a Noodle task/session type.
type TaskType = taskreg.TaskType

func isBlockingQueueItem(reg taskreg.Registry, item QueueItem) bool {
	taskType, ok := reg.ResolveQueueItem(taskreg.QueueItemInput{
		ID:      item.ID,
		TaskKey: item.TaskKey,
		Title:   item.Title,
		Skill:   item.Skill,
	})
	return ok && taskType.Blocking
}

func taskSkill(reg taskreg.Registry, taskKey, fallback string) string {
	if _, ok := reg.ByKey(taskKey); ok {
		return taskKey
	}
	return fallback
}

// PrioritizeTaskKey returns the canonical steer target for the scheduler.
func PrioritizeTaskKey() string {
	return "prioritize"
}

func isPrioritizeTarget(value string) bool {
	return strings.EqualFold(strings.TrimSpace(value), PrioritizeTaskKey())
}

// RepairTaskSkill returns the skill name for runtime repair sessions.
func RepairTaskSkill() string {
	return "debugging"
}

func registryToTaskTypeSummaries(reg taskreg.Registry) []mise.TaskTypeSummary {
	all := reg.All()
	summaries := make([]mise.TaskTypeSummary, len(all))
	for i, tt := range all {
		summaries[i] = mise.TaskTypeSummary{
			Key:      tt.Key,
			Blocking: tt.Blocking,
			Schedule: tt.Schedule,
		}
	}
	return summaries
}
