package loop

import (
	"strings"

	"github.com/poteto/noodle/internal/taskreg"
	"github.com/poteto/noodle/mise"
)

// TaskType is the canonical registry entry for a Noodle task/session type.
type TaskType = taskreg.TaskType

func taskSkill(reg taskreg.Registry, taskKey, fallback string) string {
	if _, ok := reg.ByKey(taskKey); ok {
		return taskKey
	}
	return fallback
}

// ScheduleTaskKey returns the canonical steer target for the scheduler.
func ScheduleTaskKey() string {
	return "schedule"
}

func isScheduleTarget(value string) bool {
	return strings.EqualFold(strings.TrimSpace(value), ScheduleTaskKey())
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
			CanMerge: tt.CanMerge,
			Schedule: tt.Schedule,
		}
	}
	return summaries
}
