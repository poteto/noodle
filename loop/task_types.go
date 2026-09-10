package loop

import (
	"strings"

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
	summaries := make([]mise.TaskTypeSummary, 0, len(all))
	for _, tt := range all {
		if strings.EqualFold(strings.TrimSpace(tt.Key), scheduleOrderID) {
			continue
		}
		summaries = append(summaries, mise.TaskTypeSummary{
			Key:      tt.Key,
			Schedule: tt.Schedule,
		})
	}
	return summaries
}
