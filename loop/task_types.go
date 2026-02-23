package loop

import (
	"strings"

	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/internal/taskreg"
)

// TaskType is the canonical registry entry for a Noodle task/session type.
type TaskType = taskreg.TaskType

const (
	taskKeyPlan       = taskreg.TaskKeyPlan
	taskKeyReview     = taskreg.TaskKeyReview
	taskKeyExecute    = taskreg.TaskKeyExecute
	taskKeyVerify     = taskreg.TaskKeyVerify
	taskKeyReflect    = taskreg.TaskKeyReflect
	taskKeyMeditate   = taskreg.TaskKeyMeditate
	taskKeyCook       = taskreg.TaskKeyCook
	taskKeyPrioritize = taskreg.TaskKeyPrioritize
	taskKeyTaster     = taskreg.TaskKeyTaster
	taskKeyOops       = taskreg.TaskKeyOops
	taskKeyRepair     = taskreg.TaskKeyRepair
	taskKeyDebate     = taskreg.TaskKeyDebate
)

func configuredTaskTypes(cfg config.Config) []TaskType {
	return taskreg.New(cfg).All()
}

func configuredTaskTypeByKey(cfg config.Config, taskKey string) (TaskType, bool) {
	return taskreg.New(cfg).ByKey(taskKey)
}

func configuredTaskSkill(cfg config.Config, taskKey, fallback string) string {
	taskType, ok := configuredTaskTypeByKey(cfg, taskKey)
	if !ok {
		return fallback
	}
	return nonEmpty(strings.TrimSpace(taskType.Skill), fallback)
}

func taskTypeForQueueItem(cfg config.Config, item QueueItem) (TaskType, bool) {
	return taskreg.New(cfg).ResolveQueueItem(taskreg.QueueItemInput{
		ID:      item.ID,
		TaskKey: item.TaskKey,
		Title:   item.Title,
		Skill:   item.Skill,
	})
}

func isBlockingQueueItem(cfg config.Config, item QueueItem) bool {
	taskType, ok := taskTypeForQueueItem(cfg, item)
	return ok && taskType.Blocking
}

func prioritizeTaskSkill(cfg config.Config) string {
	return configuredTaskSkill(cfg, taskKeyPrioritize, taskKeyPrioritize)
}

func tasterTaskSkill(cfg config.Config) string {
	return configuredTaskSkill(cfg, taskKeyTaster, taskKeyTaster)
}

func oopsTaskSkill(cfg config.Config) string {
	return configuredTaskSkill(cfg, taskKeyOops, taskKeyOops)
}

func repairTaskSkill(cfg config.Config) string {
	return configuredTaskSkill(cfg, taskKeyRepair, "debugging")
}

// RepairTaskSkill returns the configured skill for repair sessions.
func RepairTaskSkill(cfg config.Config) string {
	return repairTaskSkill(cfg)
}

func executeTaskKey() string {
	return taskKeyExecute
}

// PrioritizeTaskKey returns the canonical steer target for the scheduler.
func PrioritizeTaskKey() string {
	return taskKeyPrioritize
}

func isPrioritizeTarget(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return false
	}
	return strings.EqualFold(value, PrioritizeTaskKey())
}
