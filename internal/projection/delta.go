package projection

import (
	"encoding/json"
	"fmt"
	"sort"
)

// ComputeDelta compares two projection bundles and returns a deterministic delta.
func ComputeDelta(previous, current ProjectionBundle) ProjectionDelta {
	changes := make([]DeltaChange, 0)
	changes = append(changes, computeOrderChanges(previous.OrdersProjection, current.OrdersProjection)...)

	if previous.SnapshotView.Mode != current.SnapshotView.Mode {
		changes = append(changes, deltaSet("mode", current.SnapshotView.Mode))
	}
	if previous.SnapshotView.ModeEpoch != current.SnapshotView.ModeEpoch {
		changes = append(changes, deltaSet("mode_epoch", current.SnapshotView.ModeEpoch))
	}
	if previous.SnapshotView.SchemaVersion != current.SnapshotView.SchemaVersion {
		changes = append(changes, deltaSet("schema_version", current.SnapshotView.SchemaVersion))
	}
	if previous.SnapshotView.LastEventID != current.SnapshotView.LastEventID {
		changes = append(changes, deltaSet("last_event_id", current.SnapshotView.LastEventID))
	}
	if !previous.SnapshotView.GeneratedAt.Equal(current.SnapshotView.GeneratedAt) {
		changes = append(changes, deltaSet("generated_at", normalizeTime(current.SnapshotView.GeneratedAt)))
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
	}
}

func computeOrderChanges(previous, current []OrderProjection) []DeltaChange {
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
			changes = append(changes, deltaSet(path, currOrder))
		case hadPrev && !hasCurr:
			changes = append(changes, deltaDelete(path))
		default:
			changes = append(changes, diffOrder(path, prevOrder, currOrder)...)
		}
	}

	return changes
}

func diffOrder(basePath string, previous, current OrderProjection) []DeltaChange {
	changes := make([]DeltaChange, 0)

	if previous.Status != current.Status {
		changes = append(changes, deltaSet(basePath+".status", current.Status))
	}

	maxStages := len(previous.Stages)
	if len(current.Stages) > maxStages {
		maxStages = len(current.Stages)
	}

	for i := 0; i < maxStages; i++ {
		stagePath := fmt.Sprintf("%s.stages.%d", basePath, i)
		if i >= len(previous.Stages) {
			changes = append(changes, deltaSet(stagePath, current.Stages[i]))
			continue
		}
		if i >= len(current.Stages) {
			changes = append(changes, deltaDelete(stagePath))
			continue
		}
		changes = append(changes, diffStage(stagePath, previous.Stages[i], current.Stages[i])...)
	}

	return changes
}

func diffStage(basePath string, previous, current StageProjection) []DeltaChange {
	changes := make([]DeltaChange, 0, 4)
	if previous.Skill != current.Skill {
		changes = append(changes, deltaSet(basePath+".skill", current.Skill))
	}
	if previous.Runtime != current.Runtime {
		changes = append(changes, deltaSet(basePath+".runtime", current.Runtime))
	}
	if previous.Status != current.Status {
		changes = append(changes, deltaSet(basePath+".status", current.Status))
	}
	if previous.Group != current.Group {
		changes = append(changes, deltaSet(basePath+".group", current.Group))
	}
	return changes
}

func deltaSet(path string, value any) DeltaChange {
	data, err := json.Marshal(value)
	if err != nil {
		panic(fmt.Sprintf("serialize delta set value: %v", err))
	}
	return DeltaChange{
		Path:  path,
		Op:    string(DeltaOpSet),
		Value: json.RawMessage(data),
	}
}

func deltaDelete(path string) DeltaChange {
	return DeltaChange{
		Path: path,
		Op:   string(DeltaOpDelete),
	}
}
