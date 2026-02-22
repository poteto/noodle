package loop

import (
	"context"
	"time"

	"github.com/poteto/noodle/adapter"
	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/mise"
	"github.com/poteto/noodle/monitor"
	"github.com/poteto/noodle/spawner"
)

type State string

const (
	StateRunning  State = "running"
	StatePaused   State = "paused"
	StateDraining State = "draining"
)

type Queue struct {
	GeneratedAt time.Time   `json:"generated_at"`
	Items       []QueueItem `json:"items"`
}

type QueueItem struct {
	ID        string `json:"id"`
	Provider  string `json:"provider"`
	Model     string `json:"model"`
	Skill     string `json:"skill,omitempty"`
	Review    *bool  `json:"review,omitempty"`
	Rationale string `json:"rationale,omitempty"`
}

type ControlCommand struct {
	ID     string    `json:"id"`
	Action string    `json:"action"`
	Item   string    `json:"item,omitempty"`
	Name   string    `json:"name,omitempty"`
	Target string    `json:"target,omitempty"`
	Prompt string    `json:"prompt,omitempty"`
	At     time.Time `json:"at,omitempty"`
}

type ControlAck struct {
	ID      string    `json:"id"`
	Action  string    `json:"action"`
	Status  string    `json:"status"`
	Message string    `json:"message,omitempty"`
	At      time.Time `json:"at"`
}

type activeCook struct {
	queueItem     QueueItem
	session       spawner.Session
	worktreeName  string
	worktreePath  string
	attempt       int
	reviewEnabled bool
}

type Spawner interface {
	Spawn(ctx context.Context, req spawner.SpawnRequest) (spawner.Session, error)
}

type WorktreeManager interface {
	Create(name string) error
	Merge(name string) error
	Cleanup(name string, force bool) error
}

type AdapterRunner interface {
	Run(ctx context.Context, adapterName, action string, options adapter.RunOptions) (string, error)
}

type MiseBuilder interface {
	Build(ctx context.Context) (mise.Brief, []string, error)
}

type Monitor interface {
	RunOnce(ctx context.Context) ([]monitor.SessionMeta, error)
}

type Dependencies struct {
	Spawner   Spawner
	Worktree  WorktreeManager
	Adapter   AdapterRunner
	Mise      MiseBuilder
	Monitor   Monitor
	Now       func() time.Time
	QueueFile string
}

type Loop struct {
	projectDir string
	runtimeDir string
	config     config.Config
	deps       Dependencies

	state State

	activeByTarget  map[string]*activeCook
	activeByID      map[string]*activeCook
	adoptedTargets  map[string]string
	adoptedSessions []string
	failedTargets   map[string]string
	processedIDs    map[string]struct{}
}
