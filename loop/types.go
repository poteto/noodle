package loop

import (
	"context"
	"time"

	"github.com/poteto/noodle/adapter"
	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/dispatcher"
	"github.com/poteto/noodle/internal/taskreg"
	"github.com/poteto/noodle/mise"
	"github.com/poteto/noodle/monitor"
)

type State string

const (
	StateRunning  State = "running"
	StatePaused   State = "paused"
	StateDraining State = "draining"
)

type Queue struct {
	GeneratedAt  time.Time   `json:"generated_at"`
	Items        []QueueItem `json:"items"`
	Active       []string    `json:"active,omitempty"`
	ActionNeeded []string    `json:"action_needed,omitempty"`
	Autonomy     string      `json:"autonomy,omitempty"`
}

type QueueItem struct {
	ID        string   `json:"id"`
	TaskKey   string   `json:"task_key,omitempty"`
	Title     string   `json:"title,omitempty"`
	Provider  string   `json:"provider"`
	Model     string   `json:"model"`
	Runtime   string   `json:"runtime,omitempty"`
	Skill     string   `json:"skill,omitempty"`
	Plan      []string `json:"plan,omitempty"`
	Review    *bool    `json:"review,omitempty"`
	Rationale string   `json:"rationale,omitempty"`
}

type ControlCommand struct {
	ID       string    `json:"id"`
	Action   string    `json:"action"`
	Item     string    `json:"item,omitempty"`
	Name     string    `json:"name,omitempty"`
	Target   string    `json:"target,omitempty"`
	Prompt   string    `json:"prompt,omitempty"`
	Value    string    `json:"value,omitempty"`
	TaskKey  string    `json:"task_key,omitempty"`
	Provider string    `json:"provider,omitempty"`
	Model    string    `json:"model,omitempty"`
	Skill    string    `json:"skill,omitempty"`
	At       time.Time `json:"at,omitempty"`
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
	session       dispatcher.Session
	worktreeName  string
	worktreePath  string
	attempt       int
	reviewEnabled bool
}

// pendingReviewCook is a completed cook waiting for human merge/reject.
type pendingReviewCook struct {
	queueItem    QueueItem
	worktreeName string
	worktreePath string
	sessionID    string
}

type Dispatcher interface {
	Dispatch(ctx context.Context, req dispatcher.DispatchRequest) (dispatcher.Session, error)
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
	Dispatcher Dispatcher
	Worktree   WorktreeManager
	Adapter   AdapterRunner
	Mise      MiseBuilder
	Monitor   Monitor
	Registry  taskreg.Registry
	Now       func() time.Time
	QueueFile string
}

type Loop struct {
	projectDir string
	runtimeDir string
	config     config.Config
	registry    taskreg.Registry
	registryErr error
	deps        Dependencies

	state State

	activeByTarget  map[string]*activeCook
	activeByID      map[string]*activeCook
	adoptedTargets  map[string]string
	adoptedSessions []string
	failedTargets   map[string]string
	pendingReview   map[string]*pendingReviewCook
	processedIDs    map[string]struct{}

	runtimeRepairAttempts map[string]int
	runtimeRepairInFlight *runtimeRepairState
}

type runtimeIssue struct {
	Scope    string
	Message  string
	Warnings []string
	Stack    string
}

type runtimeRepairState struct {
	Fingerprint string
	Issue       runtimeIssue
	Attempt     int
	SessionID   string
	Session     dispatcher.Session
	StateBefore State
}
