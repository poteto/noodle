package orderx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/internal/filex"
	"github.com/poteto/noodle/internal/taskreg"
)

// ReadOrders reads and parses an orders.json file.
// Returns an empty OrdersFile for missing or empty files.
func ReadOrders(path string) (OrdersFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return OrdersFile{}, nil
		}
		return OrdersFile{}, fmt.Errorf("read orders: %w", err)
	}
	if strings.TrimSpace(string(data)) == "" {
		return OrdersFile{}, nil
	}
	return ParseOrdersStrict(data)
}

// ParseOrdersStrict validates orders bytes without disk I/O.
// Rejects unknown fields to prevent silent data loss on round-trip.
func ParseOrdersStrict(data []byte) (OrdersFile, error) {
	if strings.TrimSpace(string(data)) == "" {
		return OrdersFile{}, nil
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var of OrdersFile
	if err := dec.Decode(&of); err != nil {
		return OrdersFile{}, fmt.Errorf("parse orders: %w", err)
	}
	if of.Orders == nil {
		of.Orders = []Order{}
	}
	return of, nil
}

// WriteOrdersAtomic writes an OrdersFile to path via temp file + rename.
func WriteOrdersAtomic(path string, of OrdersFile) error {
	data, err := json.MarshalIndent(of, "", "  ")
	if err != nil {
		return fmt.Errorf("encode orders: %w", err)
	}
	if err := filex.WriteFileAtomic(path, append(data, '\n')); err != nil {
		return fmt.Errorf("write orders file: %w", err)
	}
	return nil
}

// ApplyOrderRoutingDefaults fills missing provider/model on stages from config
// defaults. Updates fields in-place to preserve Stage.Extra.
func ApplyOrderRoutingDefaults(of OrdersFile, reg taskreg.Registry, cfg config.Config) (OrdersFile, bool) {
	orders := make([]Order, len(of.Orders))
	copy(orders, of.Orders)
	changed := false
	for i := range orders {
		for j := range orders[i].Stages {
			if stageChanged := applyStageRoutingDefaults(&orders[i].Stages[j], reg, cfg); stageChanged {
				changed = true
			}
		}
	}
	if !changed {
		return of, false
	}
	of.Orders = orders
	return of, true
}

func applyStageRoutingDefaults(stage *Stage, _ taskreg.Registry, cfg config.Config) bool {
	changed := false
	defaultProvider := strings.TrimSpace(cfg.Routing.Defaults.Provider)
	defaultModel := strings.TrimSpace(cfg.Routing.Defaults.Model)

	if strings.TrimSpace(stage.Provider) == "" {
		if defaultProvider != "" {
			stage.Provider = defaultProvider
			changed = true
		}
	}
	if strings.TrimSpace(stage.Model) == "" {
		if defaultModel != "" {
			stage.Model = defaultModel
			changed = true
		}
	}
	return changed
}

// NormalizeAndValidateOrders validates stage task types against registry, drops
// orders with no valid main stages, normalizes IDs, and rejects orders with
// empty status.
//
// Duplicate order IDs within the same file are rejected as validation errors.
// For cross-file duplicate handling during promotion, see consumeOrdersNext.
func NormalizeAndValidateOrders(
	of OrdersFile,
	reg taskreg.Registry,
	cfg config.Config,
) (OrdersFile, bool, error) {
	orders := make([]Order, len(of.Orders))
	copy(orders, of.Orders)
	changed := false
	seenIDs := make(map[string]struct{}, len(orders))
	kept := make([]Order, 0, len(orders))

	for i := range orders {
		id := strings.TrimSpace(orders[i].ID)
		if id == "" {
			return of, false, fmt.Errorf("order id is required")
		}
		if _, exists := seenIDs[id]; exists {
			return of, false, fmt.Errorf("order %q appears more than once", id)
		}
		seenIDs[id] = struct{}{}

		// Write back trimmed ID (finding #8 — prevent dedupe bypass).
		orders[i].ID = id

		// Reject orders with invalid status.
		if err := ValidateOrderStatus(orders[i].Status); err != nil {
			return of, false, fmt.Errorf("order %q: %w", id, err)
		}
		// Completed orders should never persist — the loop removes them.
		if orders[i].Status == OrderStatusCompleted {
			return of, false, fmt.Errorf("order %q has terminal status %q", id, orders[i].Status)
		}

		// Truncate extra_prompt, validate, and filter stages in a single pass.
		validStages := make([]Stage, 0, len(orders[i].Stages))
		for j := range orders[i].Stages {
			if truncateExtraPrompt(&orders[i].Stages[j]) {
				changed = true
			}
			if err := ValidateStageStatus(orders[i].Stages[j].Status); err != nil {
				return of, false, fmt.Errorf("order %q stage %d: %w", id, j, err)
			}
			if isValidStageTaskType(&orders[i].Stages[j], reg) {
				validStages = append(validStages, orders[i].Stages[j])
			} else {
				changed = true
			}
		}

		// Drop order if no valid main stages.
		if len(validStages) == 0 {
			changed = true
			continue
		}
		orders[i].Stages = validStages

		kept = append(kept, orders[i])
	}

	if !changed {
		return of, false, nil
	}
	of.Orders = kept
	return of, true, nil
}

func isValidStageTaskType(stage *Stage, reg taskreg.Registry) bool {
	_, ok := reg.ResolveStage(taskreg.StageInput{
		TaskKey: stage.TaskKey,
		Skill:   stage.Skill,
	})
	return ok
}

const extraPromptMaxRunes = 1000

// truncateExtraPrompt caps ExtraPrompt at extraPromptMaxRunes runes.
// Prefers a word boundary (last space); falls back to hard rune truncation
// when the last space is before 80% of the limit.
func truncateExtraPrompt(stage *Stage) bool {
	runes := []rune(stage.ExtraPrompt)
	if len(runes) <= extraPromptMaxRunes {
		return false
	}
	truncated := runes[:extraPromptMaxRunes]
	// Try word boundary: find last space in truncated slice.
	lastSpace := -1
	for i := len(truncated) - 1; i >= 0; i-- {
		if truncated[i] == ' ' {
			lastSpace = i
			break
		}
	}
	threshold := extraPromptMaxRunes * 80 / 100
	if lastSpace >= threshold {
		stage.ExtraPrompt = string(truncated[:lastSpace])
	} else {
		stage.ExtraPrompt = string(truncated)
	}
	return true
}
