package loop

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/poteto/noodle/adapter"
	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/internal/orderx"
	"github.com/poteto/noodle/internal/statusfile"
	"github.com/poteto/noodle/internal/taskreg"
	"github.com/poteto/noodle/mise"
	"github.com/poteto/noodle/monitor"
	loopruntime "github.com/poteto/noodle/runtime"
)

type State string

const (
	StateRunning  State = "running"
	StatePaused   State = "paused"
	StateDraining State = "draining"
	StateIdle     State = "idle"
)

// Type aliases — orderx is canonical.
type Stage = orderx.Stage
type Order = orderx.Order
type OrdersFile = orderx.OrdersFile

// Re-export orderx status constants for in-package use.
const (
	StageStatusPending   = orderx.StageStatusPending
	StageStatusActive    = orderx.StageStatusActive
	StageStatusMerging   = orderx.StageStatusMerging
	StageStatusCompleted = orderx.StageStatusCompleted
	StageStatusFailed    = orderx.StageStatusFailed
	StageStatusCancelled = orderx.StageStatusCancelled
)

const (
	OrderStatusActive    = orderx.OrderStatusActive
	OrderStatusCompleted = orderx.OrderStatusCompleted
	OrderStatusFailed    = orderx.OrderStatusFailed
	OrderStatusFailing   = orderx.OrderStatusFailing
)

type StageResultStatus string

const (
	StageResultCompleted StageResultStatus = "completed"
	StageResultFailed    StageResultStatus = "failed"
	StageResultCancelled StageResultStatus = "cancelled"
)

type ControlCommand struct {
	ID       string    `json:"id"`
	Action   string    `json:"action"`
	OrderID  string    `json:"order_id,omitempty"`
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

// QualityVerdict is the minimal struct for reading verdict files at the merge boundary.
type QualityVerdict struct {
	Accept   bool   `json:"accept"`
	Feedback string `json:"feedback,omitempty"`
}

type StageResult struct {
	OrderID      string
	StageIndex   int
	Attempt      int
	IsOnFailure  bool
	Status       StageResultStatus
	SessionID    string
	Generation   uint64
	IsSchedule   bool
	IsBootstrap  bool
	WorktreeName string
	WorktreePath string
	Error        error
	CompletedAt  time.Time
}

// cookIdentity holds the fields shared across all cook handle types.
type cookIdentity struct {
	orderID    string
	stageIndex int
	stage      Stage
	plan       []string
}

type cookHandle struct {
	cookIdentity
	isOnFailure  bool
	orderStatus  orderx.OrderStatus
	session      loopruntime.SessionHandle
	worktreeName string
	worktreePath string
	attempt      int
	generation   uint64
	startedAt    time.Time
	displayName  string // stable kitchen name, preserved across retries
}

// pendingReviewCook is a completed cook waiting for human merge/reject.
type pendingReviewCook struct {
	cookIdentity
	worktreeName string
	worktreePath string
	sessionID    string
	reason       string
}

// pendingRetryCook is a stage whose retry dispatch failed, waiting for
// runtime repair before retrying.
type pendingRetryCook struct {
	cookIdentity
	isOnFailure bool
	orderStatus orderx.OrderStatus
	attempt     int    // the next attempt to use (already incremented)
	displayName string // stable kitchen name from original spawn
}

type WorktreeManager interface {
	Create(name string) error
	Merge(name string) error
	MergeRemoteBranch(branch string) error
	Cleanup(name string, force bool) error
}

type AdapterRunner interface {
	Run(ctx context.Context, adapterName, action string, options adapter.RunOptions) (string, error)
}

type MiseBuilder interface {
	Build(ctx context.Context, activeSummary mise.ActiveSummary, recentHistory []mise.HistoryItem) (mise.Brief, []string, error)
}

type Monitor interface {
	RunOnce(ctx context.Context) ([]monitor.SessionMeta, error)
}

type Dependencies struct {
	Runtimes       map[string]loopruntime.Runtime
	Worktree       WorktreeManager
	Adapter        AdapterRunner
	Mise           MiseBuilder
	Monitor        Monitor
	Registry       taskreg.Registry
	Logger         *slog.Logger
	Now            func() time.Time
	OrdersFile     string
	OrdersNextFile string
	StatusFile     string
}

type Loop struct {
	projectDir  string
	runtimeDir  string
	config      config.Config
	registry    taskreg.Registry
	registryErr error
	deps        Dependencies
	logger      *slog.Logger

	// Components — field-grouping sub-structs.
	cooks         cookTracker
	cmds          cmdProcessor
	completionBuf completionBuffer

	state             State
	registryStale     atomic.Bool
	registryFailCount int

	watcherWG          sync.WaitGroup
	watcherCount       atomic.Int64
	dispatchGeneration atomic.Uint64

	bootstrapAttempts  int
	bootstrapExhausted bool
	bootstrapInFlight  *cookHandle

	orders       OrdersFile
	ordersLoaded bool

	activeSummary  mise.ActiveSummary
	recentHistory  []mise.HistoryItem
	mergeQueue     *MergeQueue
	publishedState atomic.Pointer[LoopState]

	lastStatus statusfile.Status

	// Test hooks — nil in production. These allow tests to simulate crashes
	// at specific points in the state-persistence pipeline.
	TestFlushBarrier      func() // called between file writes in flushState()
	TestControlAckBarrier func() // called between command processing and ack write in processControlCommands()
}
