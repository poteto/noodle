package taskreg

import (
	"strings"

	"github.com/poteto/noodle/skill"
)

// TaskType is one schedulable task kind, discovered from skill frontmatter.
type TaskType struct {
	Key       string // skill name (e.g., "schedule", "execute", "deploy")
	CanMerge  bool
	Schedule  string // one-line guidance for schedule skill
	SkillPath string // absolute path to skill directory
}

// QueueItemInput contains queue item fields needed for task resolution.
type QueueItemInput struct {
	ID      string
	TaskKey string
	Title   string
	Skill   string
}

// Registry indexes task types for fast lookup by key.
type Registry struct {
	types []TaskType
	byKey map[string]TaskType
}

// NewFromSkills builds a registry from discovered skill metadata.
// Only skills with noodle: frontmatter are included.
func NewFromSkills(skills []skill.SkillMeta) Registry {
	types := make([]TaskType, 0, len(skills))
	for _, s := range skills {
		if !s.Frontmatter.IsTaskType() {
			continue
		}
		types = append(types, TaskType{
			Key:       s.Name,
			CanMerge:  s.Frontmatter.Noodle.Permissions.CanMerge(),
			Schedule:  s.Frontmatter.Noodle.Schedule,
			SkillPath: s.Path,
		})
	}
	reg := Registry{
		types: types,
		byKey: make(map[string]TaskType, len(types)),
	}
	for _, tt := range types {
		key := normalize(tt.Key)
		if key != "" {
			reg.byKey[key] = tt
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
	tt, ok := r.byKey[normalize(taskKey)]
	return tt, ok
}

// ResolveQueueItem resolves a queue item to its task type.
// Resolution order: task_key → skill name → exact id match.
func (r Registry) ResolveQueueItem(item QueueItemInput) (TaskType, bool) {
	if taskKey := normalize(item.TaskKey); taskKey != "" {
		if tt, ok := r.byKey[taskKey]; ok {
			return tt, true
		}
		return TaskType{}, false
	}

	if sk := normalize(item.Skill); sk != "" {
		if tt, ok := r.byKey[sk]; ok {
			return tt, true
		}
		return TaskType{}, false
	}

	id := normalize(item.ID)
	if id == "" {
		return TaskType{}, false
	}
	if tt, ok := r.byKey[id]; ok {
		return tt, true
	}
	return TaskType{}, false
}

func normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
