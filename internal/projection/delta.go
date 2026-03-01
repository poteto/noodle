package projection

import (
	"encoding/json"
	"fmt"
	"sort"
)

// ComputeDelta compares two projection bundles and returns a deterministic delta.
func ComputeDelta(previous, current ProjectionBundle) (ProjectionDelta, error) {
	changes := make([]DeltaChange, 0)
	orderChanges, err := computeOrderChanges(previous.OrdersProjection, current.OrdersProjection)
	if err != nil {
		return ProjectionDelta{}, fmt.Errorf("compute order changes: %w", err)
	}
	changes = append(changes, orderChanges...)

	if previous.SnapshotView.Mode != current.SnapshotView.Mode {
		c, err := deltaSet("mode", current.SnapshotView.Mode)
		if err != nil {
			return ProjectionDelta{}, err
		}
		changes = append(changes, c)
	}
	if previous.SnapshotView.ModeEpoch != current.SnapshotView.ModeEpoch {
		c, err := deltaSet("mode_epoch", current.SnapshotView.ModeEpoch)
		if err != nil {
			return ProjectionDelta{}, err
		}
		changes = append(changes, c)
	}
	if previous.SnapshotView.SchemaVersion != current.SnapshotView.SchemaVersion {
		c, err := deltaSet("schema_version", current.SnapshotView.SchemaVersion)
		if err != nil {
			return ProjectionDelta{}, err
		}
		changes = append(changes, c)
	}
	if previous.SnapshotView.LastEventID != current.SnapshotView.LastEventID {
		c, err := deltaSet("last_event_id", current.SnapshotView.LastEventID)
		if err != nil {
			return ProjectionDelta{}, err
		}
		changes = append(changes, c)
	}
	if !previous.SnapshotView.GeneratedAt.Equal(current.SnapshotView.GeneratedAt) {
		c, err := deltaSet("generated_at", normalizeTime(current.SnapshotView.GeneratedAt))
		if err != nil {
			return ProjectionDelta{}, err
		}
		changes = append(changes, c)
	}

	sort.SliceStable(changes, func(i, j int) bool {
		if changes[i].Path == changes[j].Path {
			return changes[i].Op < changes[j].Op
		}
		return changes[i].Path < changes[j].Path
	})

	return ProjectionDelta{
		Version:         current.Version,
		PreviousVersion: previous.Version,
		Changes:         changes,
		Timestamp:       normalizeTime(current.GeneratedAt),
	}, nil
}

func computeOrderChanges(previous, current []OrderProjection) ([]DeltaChange, error) {
	changes := make([]DeltaChange, 0)

	previousByID := make(map[string]OrderProjection, len(previous))
	currentByID := make(map[string]OrderProjection, len(current))
	seenIDs := make(map[string]struct{}, len(previous)+len(current))

	for _, order := range previous {
		previousByID[order.ID] = order
		seenIDs[order.ID] = struct{}{}
	}
	for _, order := range current {
		currentByID[order.ID] = order
		seenIDs[order.ID] = struct{}{}
	}

	ids := make([]string, 0, len(seenIDs))
	for id := range seenIDs {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	for _, id := range ids {
		prevOrder, hadPrev := previousByID[id]
		currOrder, hasCurr := currentByID[id]
		path := fmt.Sprintf("orders.%s", id)

		switch {
		case !hadPrev && hasCurr:
			c, err := deltaSet(path, currOrder)
			if err != nil {
				return nil, err
			}
			changes = append(changes, c)
		case hadPrev && !hasCurr:
			changes = append(changes, deltaDelete(path))
		default:
			orderChanges, err := diffOrder(path, prevOrder, currOrder)
			if err != nil {
				return nil, err
			}
			changes = append(changes, orderChanges...)
		}
	}

	return changes, nil
}

func diffOrder(basePath string, previous, current OrderProjection) ([]DeltaChange, error) {
	changes := make([]DeltaChange, 0)

	if previous.Status != current.Status {
		c, err := deltaSet(basePath+".status", current.Status)
		if err != nil {
			return nil, err
		}
		changes = append(changes, c)
	}

	maxStages := len(previous.Stages)
	if len(current.Stages) > maxStages {
		maxStages = len(current.Stages)
	}

	for i := 0; i < maxStages; i++ {
		stagePath := fmt.Sprintf("%s.stages.%d", basePath, i)
		if i >= len(previous.Stages) {
			c, err := deltaSet(stagePath, current.Stages[i])
			if err != nil {
				return nil, err
			}
			changes = append(changes, c)
			continue
		}
		if i >= len(current.Stages) {
			changes = append(changes, deltaDelete(stagePath))
			continue
		}
		stageChanges, err := diffStage(stagePath, previous.Stages[i], current.Stages[i])
		if err != nil {
			return nil, err
		}
		changes = append(changes, stageChanges...)
	}

	return changes, nil
}

func diffStage(basePath string, previous, current StageProjection) ([]DeltaChange, error) {
	changes := make([]DeltaChange, 0, 4)
	if previous.Skill != current.Skill {
		c, err := deltaSet(basePath+".skill", current.Skill)
		if err != nil {
			return nil, err
		}
		changes = append(changes, c)
	}
	if previous.Runtime != current.Runtime {
		c, err := deltaSet(basePath+".runtime", current.Runtime)
		if err != nil {
			return nil, err
		}
		changes = append(changes, c)
	}
	if previous.Status != current.Status {
		c, err := deltaSet(basePath+".status", current.Status)
		if err != nil {
			return nil, err
		}
		changes = append(changes, c)
	}
	if previous.Group != current.Group {
		c, err := deltaSet(basePath+".group", current.Group)
		if err != nil {
			return nil, err
		}
		changes = append(changes, c)
	}
	return changes, nil
}

func deltaSet(path string, value any) (DeltaChange, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return DeltaChange{}, fmt.Errorf("delta set value encoding failed for path %q: %v", path, err)
	}
	return DeltaChange{
		Path:  path,
		Op:    string(DeltaOpSet),
		Value: json.RawMessage(data),
	}, nil
}

func deltaDelete(path string) DeltaChange {
	return DeltaChange{
		Path: path,
		Op:   string(DeltaOpDelete),
	}
}
