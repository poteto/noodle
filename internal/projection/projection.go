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
	"github.com/poteto/noodle/internal/orderx"
	"github.com/poteto/noodle/internal/state"
	"github.com/poteto/noodle/internal/statever"
)

type hashPayload struct {
	OrdersProjection []OrderProjection    `json:"orders_projection"`
	StateMarker      statever.StateMarker `json:"state_marker"`
	SnapshotView     SnapshotView         `json:"snapshot_view"`
	Version          ProjectionVersion    `json:"version"`
	GeneratedAt      time.Time            `json:"generated_at"`
}

// Project deterministically projects canonical state into all external views.
func Project(s state.State, modeState mode.ModeState) (ProjectionBundle, error) {
	generatedAt := projectionGeneratedAt(s, modeState)
	orders := projectOrders(s)
	activeOrderIDs := projectActiveOrderIDs(s)
	pendingReviews := projectPendingReviews(s)
	version := versionFromLastEventID(s.LastEventID)
	effectiveMode := modeState.EffectiveMode
	if effectiveMode == "" {
		effectiveMode = s.Mode
	}

	snapshot := SnapshotView{
		Orders:             cloneOrderProjections(orders),
		ActiveOrderIDs:     append([]string(nil), activeOrderIDs...),
		ActionNeeded:       append([]string(nil), s.ActionNeeded...),
		PendingReviews:     clonePendingReviewProjections(pendingReviews),
		PendingReviewCount: len(pendingReviews),
		Mode:               string(effectiveMode),
		ModeEpoch:          uint64(modeState.Epoch),
		SchemaVersion:      int(s.SchemaVersion),
		LastEventID:        s.LastEventID,
		GeneratedAt:        generatedAt,
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
	hash, err := ComputeHash(bundle)
	if err != nil {
		return ProjectionBundle{}, fmt.Errorf("project compute hash: %w", err)
	}
	bundle.Hash = hash
	return bundle, nil
}

// WriteProjectionFiles atomically writes orders.json and state.json.
func WriteProjectionFiles(dir string, bundle ProjectionBundle) error {
	ordersPath := filepath.Join(dir, ordersFileName)
	statePath := filepath.Join(dir, stateFileName)

	ordersData, err := json.MarshalIndent(legacyOrdersFileProjection(bundle), "", "  ")
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
func ComputeHash(bundle ProjectionBundle) (ProjectionHash, error) {
	payload := hashPayload{
		OrdersProjection: cloneOrderProjections(bundle.OrdersProjection),
		StateMarker: statever.StateMarker{
			SchemaVersion: bundle.StateMarker.SchemaVersion,
			GeneratedAt:   normalizeTime(bundle.StateMarker.GeneratedAt),
		},
		SnapshotView: SnapshotView{
			Orders:             cloneOrderProjections(bundle.SnapshotView.Orders),
			ActiveOrderIDs:     append([]string(nil), bundle.SnapshotView.ActiveOrderIDs...),
			ActionNeeded:       append([]string(nil), bundle.SnapshotView.ActionNeeded...),
			PendingReviews:     clonePendingReviewProjections(bundle.SnapshotView.PendingReviews),
			PendingReviewCount: bundle.SnapshotView.PendingReviewCount,
			Mode:               bundle.SnapshotView.Mode,
			ModeEpoch:          bundle.SnapshotView.ModeEpoch,
			SchemaVersion:      bundle.SnapshotView.SchemaVersion,
			LastEventID:        bundle.SnapshotView.LastEventID,
			GeneratedAt:        normalizeTime(bundle.SnapshotView.GeneratedAt),
		},
		Version:     bundle.Version,
		GeneratedAt: normalizeTime(bundle.GeneratedAt),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("projection hash payload encoding failed: %v", err)
	}
	sum := sha256.Sum256(data)
	return ProjectionHash(hex.EncodeToString(sum[:])), nil
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
	ordered := make([]state.OrderNode, 0, len(s.Orders))
	for _, order := range s.Orders {
		ordered = append(ordered, order)
	}
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].Sequence == ordered[j].Sequence {
			return ordered[i].OrderID < ordered[j].OrderID
		}
		return ordered[i].Sequence < ordered[j].Sequence
	})

	orders := make([]OrderProjection, 0, len(ordered))
	for _, order := range ordered {
		orders = append(orders, OrderProjection{
			ID:        order.OrderID,
			Title:     order.Title,
			Plan:      append([]string(nil), order.Plan...),
			Rationale: order.Rationale,
			Stages:    projectStages(order.Stages),
			Status:    projectOrderStatus(order.Status),
		})
	}
	return orders
}

func projectActiveOrderIDs(s state.State) []string {
	ordered := make([]state.OrderNode, 0, len(s.Orders))
	for _, order := range s.Orders {
		if order.Status.IsTerminal() {
			continue
		}
		ordered = append(ordered, order)
	}
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].Sequence == ordered[j].Sequence {
			return ordered[i].OrderID < ordered[j].OrderID
		}
		return ordered[i].Sequence < ordered[j].Sequence
	})
	activeOrderIDs := make([]string, 0, len(ordered))
	for _, order := range ordered {
		activeOrderIDs = append(activeOrderIDs, order.OrderID)
	}
	return activeOrderIDs
}

func projectStages(stages []state.StageNode) []StageProjection {
	if len(stages) == 0 {
		return []StageProjection{}
	}
	projected := make([]StageProjection, 0, len(stages))
	for _, stage := range stages {
		extra := make(map[string]json.RawMessage, len(stage.Extra))
		for key, value := range stage.Extra {
			extra[key] = append(json.RawMessage(nil), value...)
		}
		if len(extra) == 0 {
			extra = nil
		}
		projected = append(projected, StageProjection{
			TaskKey:     stage.TaskKey,
			Prompt:      stage.Prompt,
			Skill:       stage.Skill,
			Provider:    stage.Provider,
			Model:       stage.Model,
			Runtime:     stage.Runtime,
			Group:       stage.Group,
			Status:      projectStageStatus(stage.Status),
			Extra:       extra,
			ExtraPrompt: stage.ExtraPrompt,
		})
	}
	return projected
}

func projectPendingReviews(s state.State) []PendingReviewProjection {
	if len(s.PendingReviews) == 0 {
		return []PendingReviewProjection{}
	}
	orderIDs := make([]string, 0, len(s.PendingReviews))
	for orderID := range s.PendingReviews {
		orderIDs = append(orderIDs, orderID)
	}
	sort.Strings(orderIDs)

	reviews := make([]PendingReviewProjection, 0, len(orderIDs))
	for _, orderID := range orderIDs {
		review := s.PendingReviews[orderID]
		reviews = append(reviews, PendingReviewProjection{
			OrderID:      review.OrderID,
			StageIndex:   review.StageIndex,
			TaskKey:      review.TaskKey,
			Prompt:       review.Prompt,
			Provider:     review.Provider,
			Model:        review.Model,
			Runtime:      review.Runtime,
			Skill:        review.Skill,
			Plan:         append([]string(nil), review.Plan...),
			WorktreeName: review.WorktreeName,
			WorktreePath: review.WorktreePath,
			SessionID:    review.SessionID,
			Reason:       review.Reason,
		})
	}
	return reviews
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

func projectOrderStatus(status state.OrderLifecycleStatus) string {
	return string(status)
}

func projectStageStatus(status state.StageLifecycleStatus) string {
	return string(status)
}

func cloneOrderProjections(orders []OrderProjection) []OrderProjection {
	if len(orders) == 0 {
		return []OrderProjection{}
	}
	cloned := make([]OrderProjection, 0, len(orders))
	for _, order := range orders {
		stageCopy := make([]StageProjection, 0, len(order.Stages))
		for _, stage := range order.Stages {
			stageClone := stage
			if stage.Extra != nil {
				stageClone.Extra = make(map[string]json.RawMessage, len(stage.Extra))
				for key, value := range stage.Extra {
					stageClone.Extra[key] = append(json.RawMessage(nil), value...)
				}
			}
			stageCopy = append(stageCopy, stageClone)
		}
		cloned = append(cloned, OrderProjection{
			ID:        order.ID,
			Title:     order.Title,
			Plan:      append([]string(nil), order.Plan...),
			Rationale: order.Rationale,
			Stages:    stageCopy,
			Status:    order.Status,
		})
	}
	return cloned
}

func clonePendingReviewProjections(reviews []PendingReviewProjection) []PendingReviewProjection {
	if len(reviews) == 0 {
		return []PendingReviewProjection{}
	}
	cloned := make([]PendingReviewProjection, 0, len(reviews))
	for _, review := range reviews {
		cloned = append(cloned, PendingReviewProjection{
			OrderID:      review.OrderID,
			StageIndex:   review.StageIndex,
			TaskKey:      review.TaskKey,
			Prompt:       review.Prompt,
			Provider:     review.Provider,
			Model:        review.Model,
			Runtime:      review.Runtime,
			Skill:        review.Skill,
			Plan:         append([]string(nil), review.Plan...),
			WorktreeName: review.WorktreeName,
			WorktreePath: review.WorktreePath,
			SessionID:    review.SessionID,
			Reason:       review.Reason,
		})
	}
	return cloned
}

func legacyOrdersFileProjection(bundle ProjectionBundle) orderx.OrdersFile {
	orders := make([]orderx.Order, 0, len(bundle.OrdersProjection))
	for _, order := range bundle.OrdersProjection {
		if legacyOrderRemoves(order.Status) {
			continue
		}
		stages := make([]orderx.Stage, 0, len(order.Stages))
		for _, stage := range order.Stages {
			stages = append(stages, orderx.Stage{
				TaskKey:     stage.TaskKey,
				Prompt:      stage.Prompt,
				Skill:       stage.Skill,
				Provider:    stage.Provider,
				Model:       stage.Model,
				Runtime:     stage.Runtime,
				Group:       stage.Group,
				Status:      legacyStageStatus(stage.Status),
				Extra:       cloneRawMessageMap(stage.Extra),
				ExtraPrompt: stage.ExtraPrompt,
			})
		}
		orders = append(orders, orderx.Order{
			ID:        order.ID,
			Title:     order.Title,
			Plan:      append([]string(nil), order.Plan...),
			Rationale: order.Rationale,
			Stages:    stages,
			Status:    legacyOrderStatus(order.Status),
		})
	}
	return orderx.OrdersFile{
		GeneratedAt:  normalizeTime(bundle.GeneratedAt),
		Orders:       orders,
		ActionNeeded: append([]string(nil), bundle.SnapshotView.ActionNeeded...),
	}
}

func legacyOrderRemoves(status string) bool {
	switch strings.TrimSpace(status) {
	case "completed", "cancelled":
		return true
	default:
		return false
	}
}

func legacyOrderStatus(status string) orderx.OrderStatus {
	switch strings.TrimSpace(status) {
	case "failed":
		return orderx.OrderStatusFailed
	case "completed", "cancelled":
		return orderx.OrderStatusCompleted
	default:
		return orderx.OrderStatusActive
	}
}

func legacyStageStatus(status string) orderx.StageStatus {
	switch strings.TrimSpace(status) {
	case "dispatching", "running", "review", "active":
		return orderx.StageStatusActive
	case "merging":
		return orderx.StageStatusMerging
	case "completed":
		return orderx.StageStatusCompleted
	case "failed":
		return orderx.StageStatusFailed
	case "skipped", "cancelled":
		return orderx.StageStatusCancelled
	default:
		return orderx.StageStatusPending
	}
}

func cloneRawMessageMap(src map[string]json.RawMessage) map[string]json.RawMessage {
	if len(src) == 0 {
		return nil
	}
	cloned := make(map[string]json.RawMessage, len(src))
	for key, value := range src {
		cloned[key] = append(json.RawMessage(nil), value...)
	}
	return cloned
}
