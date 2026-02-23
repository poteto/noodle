package taskreg

import (
	"strings"

	"github.com/poteto/noodle/config"
)

// TaskType describes one canonical noodle task kind.
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

// QueueItemInput contains queue item fields needed for task resolution.
type QueueItemInput struct {
	ID      string
	TaskKey string
	Title   string
	Skill   string
}

// Registry indexes task types for fast lookup by key, skill, or aliases.
type Registry struct {
	types   []TaskType
	byKey   map[string]TaskType
	bySkill map[string]TaskType
	byAlias map[string]TaskType
}

const (
	TaskKeyPlan       = "plan"
	TaskKeyReview     = "review"
	TaskKeyExecute    = "execute"
	TaskKeyVerify     = "verify"
	TaskKeyReflect    = "reflect"
	TaskKeyMeditate   = "meditate"
	TaskKeyCook       = "cook"
	TaskKeyPrioritize = "prioritize"
	TaskKeyTaster     = "taster"
	TaskKeyOops       = "oops"
	TaskKeyRepair     = "repair"
	TaskKeyDebate     = "debate"
)

var baseTaskTypes = []TaskType{
	{
		Key:        TaskKeyPlan,
		Type:       "Plan",
		ConfigPath: "[adapters.plans]",
		Blocking:   false,
		Synthetic:  false,
		Aliases:    []string{"plan", "planning", "roadmap", "decompose"},
		Purpose:    "Planning and decomposition work.",
	},
	{
		Key:        TaskKeyReview,
		Type:       "Review",
		ConfigPath: "[review]",
		Skill:      "review",
		Blocking:   true,
		Synthetic:  true,
		Aliases:    []string{"review", "chef review", "approval", "sign-off", "signoff"},
		Purpose:    "Blocking review gate work after planning.",
	},
	{
		Key:        TaskKeyExecute,
		Type:       "Execute",
		ConfigPath: "[adapters.backlog]",
		Blocking:   false,
		Synthetic:  false,
		Aliases:    []string{"execute", "implement", "build", "code", "fix"},
		Purpose:    "Primary implementation and coding work.",
	},
	{
		Key:        TaskKeyVerify,
		Type:       "Verify",
		ConfigPath: "[review]",
		Skill:      "verify",
		Blocking:   false,
		Synthetic:  true,
		Aliases:    []string{"verify", "validation", "validate", "test", "qa"},
		Purpose:    "Validation/testing/check tasks after execution.",
	},
	{
		Key:        TaskKeyReflect,
		Type:       "Reflect",
		ConfigPath: "[skills]",
		Skill:      "reflect",
		Blocking:   false,
		Synthetic:  true,
		Aliases:    []string{"reflect", "retrospective", "lessons", "postmortem"},
		Purpose:    "Persist lessons and follow-up learnings.",
	},
	{
		Key:        TaskKeyMeditate,
		Type:       "Meditate",
		ConfigPath: "[skills]",
		Skill:      "meditate",
		Blocking:   false,
		Synthetic:  true,
		Aliases:    []string{"meditate", "audit", "synthesis", "cleanup brain"},
		Purpose:    "Periodic higher-level review across multiple reflects.",
	},
	{
		Key:        TaskKeyCook,
		Type:       "Cook",
		ConfigPath: ".noodle/queue.json item",
		Blocking:   false,
		Synthetic:  false,
		Aliases:    []string{"cook", "task", "ticket"},
		Purpose:    "Backlog execution session spawned from queue items.",
	},
	{
		Key:        TaskKeyPrioritize,
		Type:       "Prioritize",
		ConfigPath: "[prioritize]",
		Blocking:   false,
		Synthetic:  true,
		Aliases:    []string{"prioritize", "scheduler"},
		Purpose:    "Queue prioritization and routing generation.",
	},
	{
		Key:        TaskKeyTaster,
		Type:       "Taster",
		ConfigPath: "[review]",
		Skill:      "taster",
		Blocking:   false,
		Synthetic:  true,
		Aliases:    []string{"taster", "taste", "quality review"},
		Purpose:    "Quality review after cook completion.",
	},
	{
		Key:        TaskKeyOops,
		Type:       "Oops",
		ConfigPath: "[phases.oops]",
		Blocking:   false,
		Synthetic:  true,
		Aliases:    []string{"oops", "infra fix", "workflow fix"},
		Purpose:    "Fix user-project infrastructure/workflow failures.",
	},
	{
		Key:        TaskKeyRepair,
		Type:       "Repair",
		ConfigPath: "[phases.debugging]",
		Blocking:   true,
		Synthetic:  true,
		Aliases:    []string{"repair", "runtime repair", "self-heal"},
		Purpose:    "Fix Noodle runtime/configuration failures.",
	},
	{
		Key:        TaskKeyDebate,
		Type:       "Debate",
		ConfigPath: "[skills]",
		Skill:      "debate",
		Blocking:   false,
		Synthetic:  true,
		Aliases:    []string{"debate", "adjudicate", "argue"},
		Purpose:    "Structured multi-round validation.",
	},
}

// New builds a canonical task registry using current config wiring.
func New(cfg config.Config) Registry {
	types := configuredTaskTypes(cfg)
	reg := Registry{
		types:   types,
		byKey:   make(map[string]TaskType, len(types)),
		bySkill: make(map[string]TaskType, len(types)),
		byAlias: make(map[string]TaskType, len(types)*2),
	}
	for _, taskType := range types {
		key := normalize(taskType.Key)
		if key != "" {
			reg.byKey[key] = taskType
		}
		skill := normalize(taskType.Skill)
		if skill != "" {
			// Preserve first-match semantics for ambiguous skill mappings.
			if _, exists := reg.bySkill[skill]; !exists {
				reg.bySkill[skill] = taskType
			}
		}
		for _, alias := range taskType.Aliases {
			alias = normalize(alias)
			if alias == "" {
				continue
			}
			if _, exists := reg.byAlias[alias]; !exists {
				reg.byAlias[alias] = taskType
			}
		}
	}
	return reg
}

func (r Registry) All() []TaskType {
	out := make([]TaskType, len(r.types))
	copy(out, r.types)
	return out
}

func (r Registry) ByKey(taskKey string) (TaskType, bool) {
	taskType, ok := r.byKey[normalize(taskKey)]
	return taskType, ok
}

func (r Registry) PrioritizeTarget() string {
	if taskType, ok := r.ByKey(TaskKeyPrioritize); ok {
		return taskType.Key
	}
	return TaskKeyPrioritize
}

func (r Registry) ResolveQueueItem(item QueueItemInput) (TaskType, bool) {
	taskKey := normalize(item.TaskKey)
	if taskKey != "" {
		if taskType, ok := r.byKey[taskKey]; ok {
			return taskType, true
		}
		if taskType, ok := r.byAlias[taskKey]; ok {
			return taskType, true
		}
		return TaskType{}, false
	}

	skill := normalize(item.Skill)
	if skill != "" {
		if taskType, ok := r.bySkill[skill]; ok {
			return taskType, true
		}
		if taskType, ok := r.byAlias[skill]; ok {
			return taskType, true
		}
		return TaskType{}, false
	}

	id := normalize(item.ID)
	if id == "" {
		return TaskType{}, false
	}
	if taskType, ok := r.byKey[id]; ok {
		return taskType, true
	}
	if taskType, ok := r.byAlias[id]; ok {
		return taskType, true
	}
	if head, _, ok := strings.Cut(id, "-"); ok {
		if taskType, exists := r.byKey[head]; exists {
			return taskType, true
		}
		if taskType, exists := r.byAlias[head]; exists {
			return taskType, true
		}
	}
	return TaskType{}, false
}

func configuredTaskTypes(cfg config.Config) []TaskType {
	out := make([]TaskType, len(baseTaskTypes))
	copy(out, baseTaskTypes)
	for i := range out {
		switch out[i].Key {
		case TaskKeyPlan:
			out[i].Skill = adapterConfiguredSkill(cfg, "plans", "plans")
		case TaskKeyExecute:
			out[i].Skill = adapterConfiguredSkill(cfg, "backlog", "backlog")
		case TaskKeyPrioritize:
			out[i].Skill = nonEmpty(strings.TrimSpace(cfg.Prioritize.Skill), TaskKeyPrioritize)
		case TaskKeyOops:
			out[i].Skill = nonEmpty(strings.TrimSpace(cfg.Phases["oops"]), "oops")
		case TaskKeyRepair:
			out[i].Skill = nonEmpty(strings.TrimSpace(cfg.Phases["debugging"]), "debugging")
		}
	}
	return out
}

func adapterConfiguredSkill(cfg config.Config, adapterName, fallback string) string {
	adapter, ok := cfg.Adapters[adapterName]
	if !ok {
		return fallback
	}
	return nonEmpty(strings.TrimSpace(adapter.Skill), fallback)
}

func nonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}

func normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
