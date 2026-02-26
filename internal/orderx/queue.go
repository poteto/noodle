package orderx

import (
	"encoding/json"
	"fmt"
	"time"
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

// Stage is a unit of work within an order (serialization type).
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

// Order is a pipeline of stages (serialization type).
type Order struct {
	ID        string   `json:"id"`
	Title     string   `json:"title,omitempty"`
	Plan      []string `json:"plan,omitempty"`
	Rationale string   `json:"rationale,omitempty"`
	Stages    []Stage  `json:"stages"`
	Status    string   `json:"status"`
	OnFailure []Stage  `json:"on_failure,omitempty"`
}

// OrdersFile is the top-level orders.json structure (serialization type).
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
