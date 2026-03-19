package projection

import (
	"encoding/json"
	"time"

	"github.com/poteto/noodle/internal/statever"
)

// ProjectionVersion is a monotonic version derived from the last applied event ID.
type ProjectionVersion uint64

// ProjectionHash is the deterministic SHA-256 hash of serialized projection content.
type ProjectionHash string

// DeltaOperation is the operation type for a delta change.
type DeltaOperation string

const (
	DeltaOpSet    DeltaOperation = "set"
	DeltaOpDelete DeltaOperation = "delete"

	ordersFileName = "orders.json"
	stateFileName  = "state.json"
)

// ProjectionBundle is the complete set of external projected outputs.
type ProjectionBundle struct {
	OrdersProjection []OrderProjection    `json:"orders_projection"`
	StateMarker      statever.StateMarker `json:"state_marker"`
	SnapshotView     SnapshotView         `json:"snapshot_view"`
	Version          ProjectionVersion    `json:"version"`
	Hash             ProjectionHash       `json:"hash"`
	GeneratedAt      time.Time            `json:"generated_at"`
}

// OrderProjection is the projected view of an order in orders.json format.
type OrderProjection struct {
	ID        string            `json:"id"`
	Title     string            `json:"title,omitempty"`
	Plan      []string          `json:"plan,omitempty"`
	Rationale string            `json:"rationale,omitempty"`
	Stages    []StageProjection `json:"stages"`
	Status    string            `json:"status"`
}

// StageProjection is the projected view of a stage.
type StageProjection struct {
	TaskKey     string                     `json:"task_key,omitempty"`
	Prompt      string                     `json:"prompt,omitempty"`
	Skill       string                     `json:"skill,omitempty"`
	Provider    string                     `json:"provider"`
	Model       string                     `json:"model"`
	Runtime     string                     `json:"runtime,omitempty"`
	Group       int                        `json:"group,omitempty"`
	Status      string                     `json:"status"`
	Extra       map[string]json.RawMessage `json:"extra,omitempty"`
	ExtraPrompt string                     `json:"extra_prompt,omitempty"`
}

// SnapshotView is the projected payload used by the snapshot API.
type SnapshotView struct {
	Orders             []OrderProjection         `json:"orders"`
	ActiveOrderIDs     []string                  `json:"active_order_ids"`
	ActionNeeded       []string                  `json:"action_needed"`
	PendingReviews     []PendingReviewProjection `json:"pending_reviews"`
	PendingReviewCount int                       `json:"pending_review_count"`
	Mode               string                    `json:"mode"`
	ModeEpoch          uint64                    `json:"mode_epoch"`
	SchemaVersion      int                       `json:"schema_version"`
	LastEventID        string                    `json:"last_event_id"`
	GeneratedAt        time.Time                 `json:"generated_at"`
}

// PendingReviewProjection is the projected view of a pending review item.
type PendingReviewProjection struct {
	OrderID      string   `json:"order_id"`
	StageIndex   int      `json:"stage_index"`
	TaskKey      string   `json:"task_key,omitempty"`
	Prompt       string   `json:"prompt,omitempty"`
	Provider     string   `json:"provider,omitempty"`
	Model        string   `json:"model,omitempty"`
	Runtime      string   `json:"runtime,omitempty"`
	Skill        string   `json:"skill,omitempty"`
	Plan         []string `json:"plan,omitempty"`
	WorktreeName string   `json:"worktree_name,omitempty"`
	WorktreePath string   `json:"worktree_path,omitempty"`
	SessionID    string   `json:"session_id,omitempty"`
	Reason       string   `json:"reason,omitempty"`
}

// ProjectionDelta is an incremental websocket update between projection versions.
type ProjectionDelta struct {
	Version         ProjectionVersion `json:"version"`
	PreviousVersion ProjectionVersion `json:"previous_version"`
	Changes         []DeltaChange     `json:"changes"`
	Timestamp       time.Time         `json:"timestamp"`
}

// DeltaChange is one changed path in a projection delta.
type DeltaChange struct {
	Path  string          `json:"path"`
	Op    string          `json:"op"`
	Value json.RawMessage `json:"value"`
}

// VersionCursor tracks a client's last seen projection version.
type VersionCursor struct {
	LastSeen ProjectionVersion `json:"last_seen"`
}

// NeedsBackfill reports whether the client missed one or more intermediate versions.
func (c VersionCursor) NeedsBackfill(current ProjectionVersion) bool {
	if current == c.LastSeen {
		return false
	}
	if current > c.LastSeen && current-c.LastSeen == 1 {
		return false
	}
	return true
}
