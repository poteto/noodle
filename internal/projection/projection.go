package projection

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/poteto/noodle/internal/filex"
	"github.com/poteto/noodle/internal/mode"
	"github.com/poteto/noodle/internal/state"
	"github.com/poteto/noodle/internal/statever"
)

type projectedOrdersFile struct {
	GeneratedAt time.Time         `json:"generated_at"`
	Orders      []OrderProjection `json:"orders"`
}

type hashPayload struct {
	OrdersProjection []OrderProjection    `json:"orders_projection"`
	StateMarker      statever.StateMarker `json:"state_marker"`
	SnapshotView     SnapshotView         `json:"snapshot_view"`
	Version          ProjectionVersion    `json:"version"`
	GeneratedAt      time.Time            `json:"generated_at"`
}

// Project deterministically projects canonical state into all external views.
func Project(s state.State, modeState mode.ModeState) ProjectionBundle {
	generatedAt := projectionGeneratedAt(s, modeState)
	orders := projectOrders(s)
	version := versionFromLastEventID(s.LastEventID)
	effectiveMode := modeState.EffectiveMode
	if effectiveMode == "" {
		effectiveMode = s.Mode
	}

	snapshot := SnapshotView{
		Orders:        cloneOrderProjections(orders),
		Mode:          string(effectiveMode),
		ModeEpoch:     uint64(modeState.Epoch),
		SchemaVersion: int(s.SchemaVersion),
		LastEventID:   s.LastEventID,
		GeneratedAt:   generatedAt,
	}

	bundle := ProjectionBundle{
		OrdersProjection: orders,
		StateMarker: statever.StateMarker{
			SchemaVersion: s.SchemaVersion,
			GeneratedAt:   generatedAt,
		},
		SnapshotView: snapshot,
		Version:      version,
		GeneratedAt:  generatedAt,
	}
	bundle.Hash = ComputeHash(bundle)
	return bundle
}

// WriteProjectionFiles atomically writes orders.json and state.json.
func WriteProjectionFiles(dir string, bundle ProjectionBundle) error {
	ordersPath := filepath.Join(dir, ordersFileName)
	statePath := filepath.Join(dir, stateFileName)

	ordersData, err := json.MarshalIndent(projectedOrdersFile{
		GeneratedAt: normalizeTime(bundle.GeneratedAt),
		Orders:      cloneOrderProjections(bundle.OrdersProjection),
	}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode orders projection: %w", err)
	}
	if err := filex.WriteFileAtomic(ordersPath, append(ordersData, '\n')); err != nil {
		return fmt.Errorf("write projected orders file: %w", err)
	}

	stateData, err := json.MarshalIndent(statever.StateMarker{
		SchemaVersion: bundle.StateMarker.SchemaVersion,
		GeneratedAt:   normalizeTime(bundle.StateMarker.GeneratedAt),
	}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode state marker projection: %w", err)
	}
	if err := filex.WriteFileAtomic(statePath, append(stateData, '\n')); err != nil {
		return fmt.Errorf("write projected state marker file: %w", err)
	}

	return nil
}

// ComputeHash returns a deterministic SHA-256 hash of projection content.
func ComputeHash(bundle ProjectionBundle) ProjectionHash {
	payload := hashPayload{
		OrdersProjection: cloneOrderProjections(bundle.OrdersProjection),
		StateMarker: statever.StateMarker{
			SchemaVersion: bundle.StateMarker.SchemaVersion,
			GeneratedAt:   normalizeTime(bundle.StateMarker.GeneratedAt),
		},
		SnapshotView: SnapshotView{
			Orders:        cloneOrderProjections(bundle.SnapshotView.Orders),
			Mode:          bundle.SnapshotView.Mode,
			ModeEpoch:     bundle.SnapshotView.ModeEpoch,
			SchemaVersion: bundle.SnapshotView.SchemaVersion,
			LastEventID:   bundle.SnapshotView.LastEventID,
			GeneratedAt:   normalizeTime(bundle.SnapshotView.GeneratedAt),
		},
		Version:     bundle.Version,
		GeneratedAt: normalizeTime(bundle.GeneratedAt),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		panic(fmt.Sprintf("serialize projection hash payload: %v", err))
	}
	sum := sha256.Sum256(data)
	return ProjectionHash(hex.EncodeToString(sum[:]))
}

func versionFromLastEventID(lastEventID string) ProjectionVersion {
	raw := strings.TrimSpace(lastEventID)
	if raw == "" {
		return 0
	}
	n, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0
	}
	return ProjectionVersion(n)
}

func projectOrders(s state.State) []OrderProjection {
	ids := make([]string, 0, len(s.Orders))
	for orderID := range s.Orders {
		ids = append(ids, orderID)
	}
	sort.Strings(ids)

	orders := make([]OrderProjection, 0, len(ids))
	for _, orderID := range ids {
		order := s.Orders[orderID]
		orders = append(orders, OrderProjection{
			ID:     order.OrderID,
			Status: string(order.Status),
			Stages: projectStages(order.Stages),
		})
	}
	return orders
}

func projectStages(stages []state.StageNode) []StageProjection {
	if len(stages) == 0 {
		return []StageProjection{}
	}
	projected := make([]StageProjection, 0, len(stages))
	for _, stage := range stages {
		projected = append(projected, StageProjection{
			Skill:   stage.Skill,
			Runtime: stage.Runtime,
			Status:  string(stage.Status),
			Group:   stage.Group,
		})
	}
	return projected
}

func projectionGeneratedAt(s state.State, modeState mode.ModeState) time.Time {
	latest := time.Time{}
	updateLatest := func(t time.Time) {
		t = normalizeTime(t)
		if t.IsZero() {
			return
		}
		if latest.IsZero() || t.After(latest) {
			latest = t
		}
	}

	for _, order := range s.Orders {
		updateLatest(order.CreatedAt)
		updateLatest(order.UpdatedAt)
		for _, stage := range order.Stages {
			for _, attempt := range stage.Attempts {
				updateLatest(attempt.StartedAt)
				updateLatest(attempt.CompletedAt)
			}
		}
	}
	for _, transition := range modeState.Transitions {
		updateLatest(transition.AppliedAt)
	}

	return latest
}

func normalizeTime(t time.Time) time.Time {
	if t.IsZero() {
		return time.Time{}
	}
	return t.UTC().Round(0)
}

func cloneOrderProjections(orders []OrderProjection) []OrderProjection {
	if len(orders) == 0 {
		return []OrderProjection{}
	}
	cloned := make([]OrderProjection, 0, len(orders))
	for _, order := range orders {
		stageCopy := make([]StageProjection, len(order.Stages))
		copy(stageCopy, order.Stages)
		cloned = append(cloned, OrderProjection{
			ID:     order.ID,
			Status: order.Status,
			Stages: stageCopy,
		})
	}
	return cloned
}
