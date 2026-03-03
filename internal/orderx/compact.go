package orderx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// CompactStage is the scheduler wire format for a stage.
type CompactStage struct {
	Do          string                     `json:"do,omitempty"`
	With        string                     `json:"with,omitempty"`
	Model       string                     `json:"model,omitempty"`
	Runtime     string                     `json:"runtime,omitempty"`
	Prompt      string                     `json:"prompt,omitempty"`
	ExtraPrompt string                     `json:"extra_prompt,omitempty"`
	Extra       map[string]json.RawMessage `json:"extra,omitempty"`
	Group       int                        `json:"group,omitempty"`
}

// CompactOrder is the scheduler wire format for an order.
type CompactOrder struct {
	ID        string         `json:"id"`
	Title     string         `json:"title,omitempty"`
	Plan      []string       `json:"plan,omitempty"`
	Rationale string         `json:"rationale,omitempty"`
	Stages    []CompactStage `json:"stages"`
}

// CompactOrdersFile is the top-level scheduler wire format.
type CompactOrdersFile struct {
	Orders       []CompactOrder `json:"orders"`
	ActionNeeded []string       `json:"action_needed,omitempty"`
}

// ParseCompactOrders parses compact orders JSON with strict unknown-field
// handling.
func ParseCompactOrders(data []byte) (CompactOrdersFile, error) {
	if strings.TrimSpace(string(data)) == "" {
		return CompactOrdersFile{}, nil
	}

	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()

	var compact CompactOrdersFile
	if err := dec.Decode(&compact); err != nil {
		return CompactOrdersFile{}, fmt.Errorf("parse compact orders: %w", err)
	}

	for i, order := range compact.Orders {
		for j, stage := range order.Stages {
			if err := validateCompactStage(stage, i, j); err != nil {
				return CompactOrdersFile{}, err
			}
		}
	}

	if compact.Orders == nil {
		compact.Orders = []CompactOrder{}
	}

	return compact, nil
}

// ExpandCompactOrders expands compact wire-format orders into internal
// OrdersFile types.
func ExpandCompactOrders(compact CompactOrdersFile) (OrdersFile, error) {
	orders := make([]Order, 0, len(compact.Orders))
	for i, order := range compact.Orders {
		stages := make([]Stage, 0, len(order.Stages))
		for j, stage := range order.Stages {
			if err := validateCompactStage(stage, i, j); err != nil {
				return OrdersFile{}, err
			}
			do := strings.TrimSpace(stage.Do)
			stages = append(stages, Stage{
				TaskKey:     do,
				Prompt:      stage.Prompt,
				Skill:       do,
				Provider:    stage.With,
				Model:       stage.Model,
				Runtime:     stage.Runtime,
				Group:       stage.Group,
				Status:      StageStatusPending,
				Extra:       stage.Extra,
				ExtraPrompt: stage.ExtraPrompt,
			})
		}
		orders = append(orders, Order{
			ID:        order.ID,
			Title:     order.Title,
			Plan:      order.Plan,
			Rationale: order.Rationale,
			Stages:    stages,
			Status:    OrderStatusActive,
		})
	}

	return OrdersFile{
		Orders:       orders,
		ActionNeeded: compact.ActionNeeded,
	}, nil
}

func validateCompactStage(stage CompactStage, orderIndex, stageIndex int) error {
	if strings.TrimSpace(stage.Do) == "" && strings.TrimSpace(stage.Prompt) == "" {
		return fmt.Errorf("order %d stage %d task key and prompt are both empty", orderIndex, stageIndex)
	}
	return nil
}
