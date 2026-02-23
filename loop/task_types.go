package loop

import (
	"strings"

	"github.com/poteto/noodle/config"
)

// TaskType describes a scheduling task type exposed to the sous-chef prompt.
type TaskType struct {
	Type       string
	ConfigPath string
	Skill      string
	Blocking   bool
	Purpose    string
}

var baseTaskTypes = []TaskType{
	{
		Type:       "Plan",
		ConfigPath: "[adapters.plans]",
		Blocking:   false,
		Purpose:    "Planning and decomposition work.",
	},
	{
		Type:       "Review",
		ConfigPath: "[review]",
		Skill:      "review",
		Blocking:   true,
		Purpose:    "Blocking review gate work after planning.",
	},
	{
		Type:       "Execute",
		ConfigPath: "[adapters.backlog]",
		Blocking:   false,
		Purpose:    "Primary implementation and coding work.",
	},
	{
		Type:       "Verify",
		ConfigPath: "[review]",
		Blocking:   false,
		Purpose:    "Validation/testing/check tasks after execution.",
	},
	{
		Type:       "Reflect",
		ConfigPath: "[skills]",
		Skill:      "reflect",
		Blocking:   false,
		Purpose:    "Persist lessons and follow-up learnings.",
	},
	{
		Type:       "Meditate",
		ConfigPath: "[skills]",
		Skill:      "meditate",
		Blocking:   false,
		Purpose:    "Periodic higher-level review across multiple reflects.",
	},
	{
		Type:       "Cook",
		ConfigPath: ".noodle/queue.json item",
		Blocking:   false,
		Purpose:    "Backlog execution session spawned from queue items.",
	},
	{
		Type:       "Sous Chef",
		ConfigPath: "[sous-chef]",
		Blocking:   false,
		Purpose:    "Queue prioritization and routing generation.",
	},
	{
		Type:       "Taster",
		ConfigPath: "[review]",
		Skill:      "taster",
		Blocking:   false,
		Purpose:    "Quality review after cook completion.",
	},
	{
		Type:       "Oops",
		ConfigPath: "[phases.oops]",
		Blocking:   false,
		Purpose:    "Fix user-project infrastructure/workflow failures.",
	},
	{
		Type:       "Repair",
		ConfigPath: "[phases.debugging]",
		Blocking:   true,
		Purpose:    "Fix Noodle runtime/configuration failures.",
	},
	{
		Type:       "Debate",
		ConfigPath: "[skills]",
		Skill:      "debate",
		Blocking:   false,
		Purpose:    "Structured multi-round validation.",
	},
}

func configuredTaskTypes(cfg config.Config) []TaskType {
	out := make([]TaskType, len(baseTaskTypes))
	copy(out, baseTaskTypes)
	for i := range out {
		switch out[i].Type {
		case "Plan":
			out[i].Skill = adapterConfiguredSkill(cfg, "plans", "plans")
		case "Execute":
			out[i].Skill = adapterConfiguredSkill(cfg, "backlog", "backlog")
		case "Sous Chef":
			out[i].Skill = nonEmpty(strings.TrimSpace(cfg.SousChef.Skill), "sous-chef")
		case "Oops":
			out[i].Skill = nonEmpty(strings.TrimSpace(cfg.Phases["oops"]), "oops")
		case "Repair":
			out[i].Skill = nonEmpty(strings.TrimSpace(cfg.Phases["debugging"]), "debugging")
		}
	}
	return out
}

func configuredTaskType(cfg config.Config, taskTypeName string) (TaskType, bool) {
	for _, taskType := range configuredTaskTypes(cfg) {
		if strings.EqualFold(strings.TrimSpace(taskType.Type), strings.TrimSpace(taskTypeName)) {
			return taskType, true
		}
	}
	return TaskType{}, false
}

func configuredTaskSkill(cfg config.Config, taskTypeName, fallback string) string {
	taskType, ok := configuredTaskType(cfg, taskTypeName)
	if !ok {
		return fallback
	}
	return nonEmpty(strings.TrimSpace(taskType.Skill), fallback)
}

func isBlockingQueueItem(cfg config.Config, item QueueItem) bool {
	itemSkill := strings.ToLower(strings.TrimSpace(item.Skill))
	itemID := strings.ToLower(strings.TrimSpace(item.ID))
	itemTitle := strings.ToLower(strings.TrimSpace(item.Title))
	for _, taskType := range configuredTaskTypes(cfg) {
		if !taskType.Blocking {
			continue
		}
		typeToken := strings.ToLower(strings.TrimSpace(taskType.Type))
		typeSkill := strings.ToLower(strings.TrimSpace(taskType.Skill))
		if typeSkill != "" && itemSkill == typeSkill {
			return true
		}
		if typeToken != "" {
			if itemID == typeToken || itemTitle == typeToken {
				return true
			}
			if strings.Contains(itemID, typeToken) || strings.Contains(itemTitle, typeToken) {
				return true
			}
		}
	}
	return false
}

func planTaskSkill(cfg config.Config) string {
	return configuredTaskSkill(cfg, "Plan", "plans")
}

func executeTaskSkill(cfg config.Config) string {
	return configuredTaskSkill(cfg, "Execute", "backlog")
}

func sousChefTaskSkill(cfg config.Config) string {
	return configuredTaskSkill(cfg, "Sous Chef", "sous-chef")
}

func tasterTaskSkill(cfg config.Config) string {
	return configuredTaskSkill(cfg, "Taster", "taster")
}

func oopsTaskSkill(cfg config.Config) string {
	return configuredTaskSkill(cfg, "Oops", "oops")
}

func repairTaskSkill(cfg config.Config) string {
	return configuredTaskSkill(cfg, "Repair", "debugging")
}

// RepairTaskSkill returns the configured skill for repair sessions.
func RepairTaskSkill(cfg config.Config) string {
	return repairTaskSkill(cfg)
}

func adapterConfiguredSkill(cfg config.Config, adapterName, fallback string) string {
	adapter, ok := cfg.Adapters[adapterName]
	if !ok {
		return fallback
	}
	return nonEmpty(strings.TrimSpace(adapter.Skill), fallback)
}
