package loop

import (
	"github.com/poteto/noodle/internal/taskreg"
	"github.com/poteto/noodle/mise"
)

// TaskType is the canonical registry entry for a Noodle task/session type.
type TaskType = taskreg.TaskType

// ScheduleTaskKey returns the canonical steer target for the scheduler.
func ScheduleTaskKey() string {
	return "schedule"
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
			Schedule: tt.Schedule,
		}
	}
	return summaries
}
