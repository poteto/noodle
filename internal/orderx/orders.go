package orderx

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/internal/filex"
	"github.com/poteto/noodle/internal/stringx"
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
func ParseOrdersStrict(data []byte) (OrdersFile, error) {
	if strings.TrimSpace(string(data)) == "" {
		return OrdersFile{}, nil
	}
	var of OrdersFile
	if err := json.Unmarshal(data, &of); err != nil {
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
// defaults. Applies to both Stages and OnFailure stages. Updates fields
// in-place to preserve Stage.Extra.
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
		for j := range orders[i].OnFailure {
			if stageChanged := applyStageRoutingDefaults(&orders[i].OnFailure[j], reg, cfg); stageChanged {
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

func applyStageRoutingDefaults(stage *Stage, reg taskreg.Registry, cfg config.Config) bool {
	changed := false
	defaultProvider := strings.TrimSpace(cfg.Routing.Defaults.Provider)
	defaultModel := strings.TrimSpace(cfg.Routing.Defaults.Model)
	tagProvider := ""
	tagModel := ""

	if taskType, ok := reg.ResolveStage(taskreg.StageInput{
		TaskKey: stage.TaskKey,
		Skill:   stage.Skill,
	}); ok {
		if policy, exists := cfg.Routing.Tags[taskType.Key]; exists {
			tagProvider = strings.TrimSpace(policy.Provider)
			tagModel = strings.TrimSpace(policy.Model)
		}
	}

	if strings.TrimSpace(stage.Provider) == "" {
		provider := stringx.FirstNonEmpty(tagProvider, defaultProvider)
		if provider != "" {
			stage.Provider = provider
			changed = true
		}
	}
	if strings.TrimSpace(stage.Model) == "" {
		model := stringx.FirstNonEmpty(tagModel, defaultModel)
		if model != "" {
			stage.Model = model
			changed = true
		}
	}
	return changed
}

// NormalizeAndValidateOrders validates stage task types against registry, drops
// orders with no valid main stages, strips invalid OnFailure stages, normalizes
// IDs, and rejects orders with empty status.
//
// Duplicate order IDs within the same file are rejected as validation errors.
// For cross-file duplicate handling during promotion, see consumeOrdersNext.
//
// If a "failing" order has all its OnFailure stages stripped, it is treated as
// terminal: removed and an error annotation is appended.
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

		// Reject orders with empty status.
		if err := ValidateOrderStatus(orders[i].Status); err != nil {
			return of, false, fmt.Errorf("order %q: %w", id, err)
		}

		// Validate and filter main stages.
		validStages := make([]Stage, 0, len(orders[i].Stages))
		for j := range orders[i].Stages {
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

		// Validate and filter OnFailure stages.
		if len(orders[i].OnFailure) > 0 {
			validOnFailure := make([]Stage, 0, len(orders[i].OnFailure))
			for j := range orders[i].OnFailure {
				if isValidStageTaskType(&orders[i].OnFailure[j], reg) {
					validOnFailure = append(validOnFailure, orders[i].OnFailure[j])
				} else {
					changed = true
				}
			}
			if len(validOnFailure) == 0 {
				orders[i].OnFailure = nil
				changed = true
			} else if len(validOnFailure) != len(orders[i].OnFailure) {
				orders[i].OnFailure = validOnFailure
			}
		}

		// "failing" + empty OnFailure = terminal.
		if orders[i].Status == OrderStatusFailing && len(orders[i].OnFailure) == 0 {
			changed = true
			of.ActionNeeded = append(of.ActionNeeded,
				fmt.Sprintf("order %q: failing with no recovery stages, removed as terminal", id))
			continue
		}

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
