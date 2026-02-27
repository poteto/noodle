package loop

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/poteto/noodle/adapter"
	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/dispatcher"
	"github.com/poteto/noodle/internal/statusfile"
	"github.com/poteto/noodle/internal/taskreg"
	"github.com/poteto/noodle/mise"
	"github.com/poteto/noodle/monitor"
	nrt "github.com/poteto/noodle/runtime"
)

type State string

const (
	StateRunning  State = "running"
	StatePaused   State = "paused"
	StateDraining State = "draining"
	StateIdle     State = "idle"
)

// Stage status constants.
const (
	StageStatusPending   = "pending"
	StageStatusActive    = "active"
	StageStatusCompleted = "completed"
	StageStatusFailed    = "failed"
	StageStatusCancelled = "cancelled"
)

// Order status constants.
const (
	OrderStatusActive    = "active"
	OrderStatusCompleted = "completed"
	OrderStatusFailed    = "failed"
	OrderStatusFailing   = "failing"
)

// Stage is a unit of work within an order.
type Stage struct {
	TaskKey  string                     `json:"task_key,omitempty"`
	Prompt   string                     `json:"prompt,omitempty"`
	Skill    string                     `json:"skill,omitempty"`
	Provider string                     `json:"provider"`
	Model    string                     `json:"model"`
	Runtime  string                     `json:"runtime,omitempty"`
	Status   string                     `json:"status"`
	Extra    map[string]json.RawMessage `json:"extra,omitempty"`
}

// Order is a pipeline of stages.
type Order struct {
	ID        string  `json:"id"`
	Title     string  `json:"title,omitempty"`
	Plan      []string `json:"plan,omitempty"`
	Rationale string  `json:"rationale,omitempty"`
	Stages    []Stage `json:"stages"`
	Status    string  `json:"status"`
	OnFailure []Stage `json:"on_failure,omitempty"`
}

// OrdersFile is the top-level orders.json structure.
type OrdersFile struct {
	GeneratedAt  time.Time `json:"generated_at"`
	Orders       []Order   `json:"orders"`
	ActionNeeded []string  `json:"action_needed,omitempty"`
}

// ValidateOrderStatus returns an error if the order status is not valid.
func ValidateOrderStatus(status string) error {
	switch status {
	case OrderStatusActive, OrderStatusCompleted, OrderStatusFailed, OrderStatusFailing:
		return nil
	case "":
		return fmt.Errorf("order status is required")
	default:
		return fmt.Errorf("unknown order status %q", status)
	}
}

// ValidateStageStatus returns an error if the stage status is not valid.
func ValidateStageStatus(status string) error {
	switch status {
	case StageStatusPending, StageStatusActive, StageStatusCompleted, StageStatusFailed, StageStatusCancelled:
		return nil
	case "":
		return fmt.Errorf("stage status is required")
	default:
		return fmt.Errorf("unknown stage status %q", status)
	}
}

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

// StageResultStatus indicates how a stage completed.
type StageResultStatus string

const (
	StageResultCompleted StageResultStatus = "completed"
	StageResultFailed    StageResultStatus = "failed"
)

// StageResult is pushed by a watcher goroutine when a session finishes.
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
}

type cookHandle struct {
	orderID      string
	stageIndex   int
	stage        Stage
	isOnFailure  bool
	orderStatus  string
	plan         []string
	session      nrt.SessionHandle
	done         <-chan struct{}
	generation   uint64
	worktreeName string
	worktreePath string
	attempt      int
	displayName  string // stable kitchen name, preserved across retries
}

// pendingReviewCook is a completed cook waiting for human merge/reject.
type pendingReviewCook struct {
	orderID      string
	stageIndex   int
	stage        Stage
	plan         []string
	worktreeName string
	worktreePath string
	sessionID    string
	reason       string
}

// pendingRetryCook is a stage whose retry dispatch failed, waiting for
// runtime repair before retrying.
type pendingRetryCook struct {
	orderID     string
	stageIndex  int
	stage       Stage
	isOnFailure bool
	orderStatus string
	plan        []string
	attempt     int    // the next attempt to use (already incremented)
	displayName string // stable kitchen name from original spawn
}

// Dispatcher dispatches sessions. Satisfied by runtime.RuntimeMap.
type Dispatcher interface {
	Dispatch(ctx context.Context, req dispatcher.DispatchRequest) (nrt.SessionHandle, error)
}

// legacyDispatcherAdapter wraps a dispatcher.Dispatcher to satisfy the
// Dispatcher interface by converting Session → SessionHandle.
type legacyDispatcherAdapter struct {
	inner dispatcher.Dispatcher
}

func (a *legacyDispatcherAdapter) Dispatch(ctx context.Context, req dispatcher.DispatchRequest) (nrt.SessionHandle, error) {
	return a.inner.Dispatch(ctx, req)
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
	Build(ctx context.Context) (mise.Brief, []string, error)
}

type Monitor interface {
	RunOnce(ctx context.Context) ([]monitor.SessionMeta, error)
}

type Dependencies struct {
	Dispatcher Dispatcher
	Worktree   WorktreeManager
	Adapter    AdapterRunner
	Mise       MiseBuilder
	Monitor    Monitor
	Registry   taskreg.Registry
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

	state            State
	registryStale    atomic.Bool
	registryFailCount int

	activeCooksByOrder map[string]*cookHandle
	adoptedTargets  map[string]string
	adoptedSessions []string
	failedTargets   map[string]string
	pendingReview   map[string]*pendingReviewCook
	pendingRetry    map[string]*pendingRetryCook
	processedIDs    map[string]struct{}

	completions    chan StageResult
	completionsMu  sync.Mutex
	completionOver []StageResult
	nextGeneration uint64
	watcherWG      sync.WaitGroup

	bootstrapAttempts  int
	bootstrapExhausted bool
	bootstrapInFlight  *cookHandle

	lastStatus statusfile.Status
}
