package loop

import (
	"strings"

	"github.com/poteto/noodle/config"
)

// TaskType is the canonical registry entry for a Noodle task/session type.
type TaskType struct {
	Key        string
	Type       string
	ConfigPath string
	Skill      string
	Blocking   bool
	Synthetic  bool
	Aliases    []string
	Purpose    string
}

const (
	taskKeyPlan       = "plan"
	taskKeyReview     = "review"
	taskKeyExecute    = "execute"
	taskKeyVerify     = "verify"
	taskKeyReflect    = "reflect"
	taskKeyMeditate   = "meditate"
	taskKeyCook       = "cook"
	taskKeyPrioritize = "prioritize"
	taskKeyTaster     = "taster"
	taskKeyOops       = "oops"
	taskKeyRepair     = "repair"
	taskKeyDebate     = "debate"
)

var baseTaskTypes = []TaskType{
	{
		Key:        taskKeyPlan,
		Type:       "Plan",
		ConfigPath: "[adapters.plans]",
		Blocking:   false,
		Synthetic:  false,
		Aliases:    []string{"plan", "planning", "roadmap", "decompose"},
		Purpose:    "Planning and decomposition work.",
	},
	{
		Key:        taskKeyReview,
		Type:       "Review",
		ConfigPath: "[review]",
		Skill:      "review",
		Blocking:   true,
		Synthetic:  true,
		Aliases:    []string{"review", "chef review", "approval", "sign-off", "signoff"},
		Purpose:    "Blocking review gate work after planning.",
	},
	{
		Key:        taskKeyExecute,
		Type:       "Execute",
		ConfigPath: "[adapters.backlog]",
		Blocking:   false,
		Synthetic:  false,
		Aliases:    []string{"execute", "implement", "build", "code", "fix"},
		Purpose:    "Primary implementation and coding work.",
	},
	{
		Key:        taskKeyVerify,
		Type:       "Verify",
		ConfigPath: "[review]",
		Skill:      "verify",
		Blocking:   false,
		Synthetic:  true,
		Aliases:    []string{"verify", "validation", "validate", "test", "qa"},
		Purpose:    "Validation/testing/check tasks after execution.",
	},
	{
		Key:        taskKeyReflect,
		Type:       "Reflect",
		ConfigPath: "[skills]",
		Skill:      "reflect",
		Blocking:   false,
		Synthetic:  true,
		Aliases:    []string{"reflect", "retrospective", "lessons", "postmortem"},
		Purpose:    "Persist lessons and follow-up learnings.",
	},
	{
		Key:        taskKeyMeditate,
		Type:       "Meditate",
		ConfigPath: "[skills]",
		Skill:      "meditate",
		Blocking:   false,
		Synthetic:  true,
		Aliases:    []string{"meditate", "audit", "synthesis", "cleanup brain"},
		Purpose:    "Periodic higher-level review across multiple reflects.",
	},
	{
		Key:        taskKeyCook,
		Type:       "Cook",
		ConfigPath: ".noodle/queue.json item",
		Blocking:   false,
		Synthetic:  false,
		Aliases:    []string{"cook", "task", "ticket"},
		Purpose:    "Backlog execution session spawned from queue items.",
	},
	{
		Key:        taskKeyPrioritize,
		Type:       "Prioritize",
		ConfigPath: "[prioritize]",
		Blocking:   false,
		Synthetic:  true,
		Aliases:    []string{"prioritize", "scheduler"},
		Purpose:    "Queue prioritization and routing generation.",
	},
	{
		Key:        taskKeyTaster,
		Type:       "Taster",
		ConfigPath: "[review]",
		Skill:      "taster",
		Blocking:   false,
		Synthetic:  true,
		Aliases:    []string{"taster", "taste", "quality review"},
		Purpose:    "Quality review after cook completion.",
	},
	{
		Key:        taskKeyOops,
		Type:       "Oops",
		ConfigPath: "[phases.oops]",
		Blocking:   false,
		Synthetic:  true,
		Aliases:    []string{"oops", "infra fix", "workflow fix"},
		Purpose:    "Fix user-project infrastructure/workflow failures.",
	},
	{
		Key:        taskKeyRepair,
		Type:       "Repair",
		ConfigPath: "[phases.debugging]",
		Blocking:   true,
		Synthetic:  true,
		Aliases:    []string{"repair", "runtime repair", "self-heal"},
		Purpose:    "Fix Noodle runtime/configuration failures.",
	},
	{
		Key:        taskKeyDebate,
		Type:       "Debate",
		ConfigPath: "[skills]",
		Skill:      "debate",
		Blocking:   false,
		Synthetic:  true,
		Aliases:    []string{"debate", "adjudicate", "argue"},
		Purpose:    "Structured multi-round validation.",
	},
}

func configuredTaskTypes(cfg config.Config) []TaskType {
	out := make([]TaskType, len(baseTaskTypes))
	copy(out, baseTaskTypes)
	for i := range out {
		switch out[i].Key {
		case taskKeyPlan:
			out[i].Skill = adapterConfiguredSkill(cfg, "plans", "plans")
		case taskKeyExecute:
			out[i].Skill = adapterConfiguredSkill(cfg, "backlog", "backlog")
		case taskKeyPrioritize:
			out[i].Skill = nonEmpty(strings.TrimSpace(cfg.Prioritize.Skill), "prioritize")
		case taskKeyOops:
			out[i].Skill = nonEmpty(strings.TrimSpace(cfg.Phases["oops"]), "oops")
		case taskKeyRepair:
			out[i].Skill = nonEmpty(strings.TrimSpace(cfg.Phases["debugging"]), "debugging")
		}
	}
	return out
}

func configuredTaskTypeByKey(cfg config.Config, taskKey string) (TaskType, bool) {
	taskKey = strings.ToLower(strings.TrimSpace(taskKey))
	if taskKey == "" {
		return TaskType{}, false
	}
	for _, taskType := range configuredTaskTypes(cfg) {
		if strings.EqualFold(taskType.Key, taskKey) {
			return taskType, true
		}
	}
	return TaskType{}, false
}

func configuredTaskSkill(cfg config.Config, taskKey, fallback string) string {
	taskType, ok := configuredTaskTypeByKey(cfg, taskKey)
	if !ok {
		return fallback
	}
	return nonEmpty(strings.TrimSpace(taskType.Skill), fallback)
}

func taskTypeForQueueItem(cfg config.Config, item QueueItem) (TaskType, bool) {
	if taskType, ok := configuredTaskTypeByKey(cfg, item.TaskKey); ok {
		return taskType, true
	}

	skillName := strings.ToLower(strings.TrimSpace(item.Skill))
	if skillName != "" {
		for _, taskType := range configuredTaskTypes(cfg) {
			if strings.EqualFold(strings.TrimSpace(taskType.Skill), skillName) {
				return taskType, true
			}
		}
	}

	text := strings.ToLower(strings.TrimSpace(item.ID + " " + item.Title))
	if text == "" {
		return TaskType{}, false
	}
	for _, taskType := range configuredTaskTypes(cfg) {
		for _, alias := range taskType.Aliases {
			alias = strings.ToLower(strings.TrimSpace(alias))
			if alias == "" {
				continue
			}
			if strings.Contains(text, alias) {
				return taskType, true
			}
		}
	}

	return TaskType{}, false
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

func adapterConfiguredSkill(cfg config.Config, adapterName, fallback string) string {
	adapter, ok := cfg.Adapters[adapterName]
	if !ok {
		return fallback
	}
	return nonEmpty(strings.TrimSpace(adapter.Skill), fallback)
}
