package loop

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/poteto/noodle/internal/ingest"
	"github.com/poteto/noodle/internal/orderx"
	"github.com/poteto/noodle/internal/reducer"
	"github.com/poteto/noodle/internal/state"
	"github.com/poteto/noodle/internal/statever"
)

func (l *Loop) canonicalSnapshotPath() string {
	if strings.TrimSpace(l.runtimeDir) == "" {
		return ""
	}
	return filepath.Join(l.runtimeDir, "state.snapshot.json")
}

func (l *Loop) loadOrBootstrapCanonical() error {
	loaded, err := l.loadCanonicalSnapshot()
	if err != nil {
		return err
	}
	if loaded {
		return nil
	}
	return l.bootstrapCanonicalFromLegacy()
}

func (l *Loop) loadCanonicalSnapshot() (bool, error) {
	path := l.canonicalSnapshotPath()
	if path == "" {
		return false, nil
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("stat canonical snapshot: %w", err)
	}

	snapshot, err := reducer.ReadSnapshot(path)
	if err != nil {
		return false, err
	}
	l.canonical = snapshot.State.Clone()
	l.effectLedger = restoreEffectLedger(snapshot.EffectLedger)

	lastEventID, err := parseLastEventID(l.canonical.LastEventID)
	if err != nil {
		return false, err
	}
	l.eventCounter.Store(lastEventID)
	l.canonicalLoaded = true
	return true, nil
}

func (l *Loop) bootstrapCanonicalFromLegacy() error {
	orders, err := l.currentOrders()
	if err != nil {
		return fmt.Errorf("bootstrap canonical from legacy orders: %w", err)
	}
	pendingReview, err := ReadPendingReview(l.runtimeDir)
	if err != nil {
		return fmt.Errorf("bootstrap canonical from legacy pending review: %w", err)
	}

	now := time.Now().UTC()
	if l.deps.Now != nil {
		now = l.deps.Now().UTC()
	}

	l.canonical = synthesizeCanonicalState(orders, pendingReview, state.RunMode(l.config.Mode), now)
	l.effectLedger = reducer.NewEffectLedger()
	l.eventCounter.Store(0)
	l.canonicalLoaded = true
	return l.persistCanonicalCheckpoint()
}

func (l *Loop) persistCanonicalCheckpoint() error {
	path := l.canonicalSnapshotPath()
	if path == "" {
		return nil
	}
	if l.effectLedger == nil {
		l.effectLedger = reducer.NewEffectLedger()
	}
	now := time.Now().UTC()
	if l.deps.Now != nil {
		now = l.deps.Now().UTC()
	}
	snapshot := reducer.BuildSnapshot(l.canonical, l.effectLedger, now)
	return reducer.WriteSnapshotAtomic(path, snapshot)
}

func synthesizeCanonicalState(
	orders OrdersFile,
	pendingReview []PendingReviewItem,
	mode state.RunMode,
	now time.Time,
) state.State {
	reviewByOrder := make(map[string]PendingReviewItem, len(pendingReview))
	for _, item := range pendingReview {
		orderID := strings.TrimSpace(item.OrderID)
		if orderID == "" {
			continue
		}
		reviewByOrder[orderID] = item
	}

	canonicalOrders := make(map[string]state.OrderNode, len(orders.Orders))
	canonicalReviews := make(map[string]state.PendingReviewNode, len(pendingReview))
	for orderIndex, order := range orders.Orders {
		stageNodes := make([]state.StageNode, 0, len(order.Stages))
		reviewItem, hasReview := reviewByOrder[order.ID]
		for i, stage := range order.Stages {
			stageStatus := legacyStageStatusToCanonical(stage.Status)
			if hasReview && reviewItem.StageIndex == i {
				stageStatus = state.StageReview
			}

			stageNode := state.StageNode{
				StageIndex:  i,
				TaskKey:     strings.TrimSpace(stage.TaskKey),
				Prompt:      strings.TrimSpace(stage.Prompt),
				Status:      stageStatus,
				Skill:       strings.TrimSpace(stage.Skill),
				Provider:    strings.TrimSpace(stage.Provider),
				Model:       strings.TrimSpace(stage.Model),
				Runtime:     strings.TrimSpace(stage.Runtime),
				Group:       stage.Group,
				Extra:       cloneLegacyExtra(stage.Extra),
				ExtraPrompt: strings.TrimSpace(stage.ExtraPrompt),
			}
			stageNode.Merge = canonicalMergeRecoveryFromLegacyStage(stage, stageStatus)
			if stageNode.Runtime == "" {
				stageNode.Runtime = "process"
			}
			if hasReview && reviewItem.StageIndex == i {
				attempt := state.AttemptNode{
					AttemptID:    "legacy-review-" + order.ID,
					SessionID:    strings.TrimSpace(reviewItem.SessionID),
					Status:       state.AttemptCompleted,
					CompletedAt:  now,
					WorktreeName: strings.TrimSpace(reviewItem.WorktreeName),
				}
				stageNode.Attempts = []state.AttemptNode{attempt}
			}
			stageNodes = append(stageNodes, stageNode)
		}
		if hasReview {
			stage := Stage{}
			if reviewItem.StageIndex >= 0 && reviewItem.StageIndex < len(order.Stages) {
				stage = order.Stages[reviewItem.StageIndex]
			}
			canonicalReviews[order.ID] = state.PendingReviewNode{
				OrderID:      strings.TrimSpace(reviewItem.OrderID),
				StageIndex:   reviewItem.StageIndex,
				TaskKey:      strings.TrimSpace(stage.TaskKey),
				Prompt:       strings.TrimSpace(stage.Prompt),
				Provider:     strings.TrimSpace(stage.Provider),
				Model:        strings.TrimSpace(stage.Model),
				Runtime:      strings.TrimSpace(stage.Runtime),
				Skill:        strings.TrimSpace(stage.Skill),
				Plan:         slices.Clone(order.Plan),
				WorktreeName: strings.TrimSpace(reviewItem.WorktreeName),
				WorktreePath: strings.TrimSpace(reviewItem.WorktreePath),
				SessionID:    strings.TrimSpace(reviewItem.SessionID),
				Reason:       strings.TrimSpace(reviewItem.Reason),
			}
		}
		canonicalOrders[order.ID] = state.OrderNode{
			OrderID:   strings.TrimSpace(order.ID),
			Sequence:  orderIndex,
			Title:     strings.TrimSpace(order.Title),
			Plan:      slices.Clone(order.Plan),
			Rationale: strings.TrimSpace(order.Rationale),
			Status:    legacyOrderStatusToCanonical(order.Status),
			Stages:    stageNodes,
			CreatedAt: now,
			UpdatedAt: now,
		}
	}

	return state.State{
		Orders:         canonicalOrders,
		PendingReviews: canonicalReviews,
		ActionNeeded:   slices.Clone(orders.ActionNeeded),
		Mode:           mode,
		SchemaVersion:  statever.Current,
		LastEventID:    "0",
	}
}

func legacyOrderStatusToCanonical(status orderx.OrderStatus) state.OrderLifecycleStatus {
	switch status {
	case OrderStatusFailed:
		return state.OrderFailed
	case OrderStatusCompleted:
		return state.OrderCompleted
	default:
		return state.OrderActive
	}
}

func legacyStageStatusToCanonical(status orderx.StageStatus) state.StageLifecycleStatus {
	switch status {
	case StageStatusActive:
		return state.StageRunning
	case StageStatusMerging:
		return state.StageMerging
	case StageStatusCompleted:
		return state.StageCompleted
	case StageStatusFailed:
		return state.StageFailed
	case StageStatusCancelled:
		return state.StageCancelled
	default:
		return state.StagePending
	}
}

func restoreEffectLedger(records []reducer.EffectLedgerRecord) *reducer.EffectLedger {
	ledger := reducer.NewEffectLedger()
	for _, rec := range records {
		ledger.Record(rec.Effect)

		switch rec.Status {
		case reducer.EffectLedgerPending:
			continue
		case reducer.EffectLedgerRunning:
			_ = ledger.MarkRunning(rec.EffectID)
		case reducer.EffectLedgerDone:
			_ = ledger.MarkRunning(rec.EffectID)
			result := rec.Result
			if result == nil {
				result = &reducer.EffectResult{EffectID: rec.EffectID, Status: reducer.EffectResultCompleted}
			}
			_ = ledger.MarkDone(rec.EffectID, *result)
		case reducer.EffectLedgerFailed:
			_ = ledger.MarkRunning(rec.EffectID)
			result := rec.Result
			if result == nil {
				result = &reducer.EffectResult{EffectID: rec.EffectID, Status: reducer.EffectResultFailed}
			}
			_ = ledger.MarkFailed(rec.EffectID, *result)
		}
	}
	return ledger
}

func parseLastEventID(raw string) (uint64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	n, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse last event id %q: %w", raw, err)
	}
	return n, nil
}

func (l *Loop) trackCanonicalOrder(order Order) {
	l.syncCanonicalOrderFromLegacy(order)
}

func (l *Loop) syncCanonicalOrderFromLegacy(order Order) {
	l.syncCanonicalOrderFromLegacyAt(order, -1)
}

func (l *Loop) syncCanonicalOrderFromLegacyAt(order Order, sequence int) {
	if _, exists := l.canonical.Orders[order.ID]; !exists {
		stages := make([]map[string]any, 0, len(order.Stages))
		for i, stage := range order.Stages {
			stages = append(stages, map[string]any{
				"stage_index": i,
				"status":      legacyStageStatusToCanonical(stage.Status),
				"task_key":    strings.TrimSpace(stage.TaskKey),
				"prompt":      strings.TrimSpace(stage.Prompt),
				"skill":       stage.Skill,
				"provider":    strings.TrimSpace(stage.Provider),
				"model":       strings.TrimSpace(stage.Model),
				"runtime":     nonEmpty(stage.Runtime, "process"),
				"group":       stage.Group,
			})
		}
		l.emitEvent(ingest.EventSchedulePromoted, map[string]any{
			"order_id": order.ID,
			"stages":   stages,
			"metadata": map[string]string{
				"title":     strings.TrimSpace(order.Title),
				"plan":      strings.Join(order.Plan, "\n"),
				"rationale": strings.TrimSpace(order.Rationale),
			},
		})
	}

	node := l.canonical.Orders[order.ID]
	node.OrderID = strings.TrimSpace(order.ID)
	if sequence >= 0 {
		node.Sequence = sequence
	}
	node.Title = strings.TrimSpace(order.Title)
	node.Plan = slices.Clone(order.Plan)
	node.Rationale = strings.TrimSpace(order.Rationale)
	node.Status = legacyOrderStatusToCanonical(order.Status)
	now := timeNowUTC(l.deps.Now)
	node.UpdatedAt = now

	stages := make([]state.StageNode, 0, len(order.Stages))
	for i, stage := range order.Stages {
		stageNode := state.StageNode{
			StageIndex:  i,
			TaskKey:     strings.TrimSpace(stage.TaskKey),
			Prompt:      strings.TrimSpace(stage.Prompt),
			Status:      legacyStageStatusToCanonical(stage.Status),
			Skill:       strings.TrimSpace(stage.Skill),
			Provider:    strings.TrimSpace(stage.Provider),
			Model:       strings.TrimSpace(stage.Model),
			Runtime:     nonEmpty(stage.Runtime, "process"),
			Group:       stage.Group,
			Extra:       cloneLegacyExtra(stage.Extra),
			ExtraPrompt: strings.TrimSpace(stage.ExtraPrompt),
		}
		if i < len(node.Stages) {
			stageNode.Attempts = slices.Clone(node.Stages[i].Attempts)
			if node.Stages[i].Merge != nil {
				mergeCopy := *node.Stages[i].Merge
				stageNode.Merge = &mergeCopy
			}
		}
		if mergeRecovery := canonicalMergeRecoveryFromLegacyStage(stage, stageNode.Status); mergeRecovery != nil {
			stageNode.Merge = mergeRecovery
		}
		stages = append(stages, stageNode)
	}
	node.Stages = stages
	l.canonical.Orders[order.ID] = node
}

func (l *Loop) syncCanonicalStateFromOrders(orders OrdersFile) {
	if l.canonical.Orders == nil {
		l.canonical.Orders = make(map[string]state.OrderNode)
	}
	present := make(map[string]struct{}, len(orders.Orders))
	for index, order := range orders.Orders {
		present[order.ID] = struct{}{}
		l.syncCanonicalOrderFromLegacyAt(order, index)
	}
	for orderID := range l.canonical.Orders {
		if _, ok := present[orderID]; ok {
			continue
		}
		delete(l.canonical.Orders, orderID)
		delete(l.canonical.PendingReviews, orderID)
	}
	l.canonical.ActionNeeded = slices.Clone(orders.ActionNeeded)
}

func canonicalMergeRecoveryFromLegacyStage(stage Stage, status state.StageLifecycleStatus) *state.MergeRecoveryNode {
	if status != state.StageMerging {
		return nil
	}
	worktreeName := strings.TrimSpace(legacyExtraString(stage.Extra, "merge_worktree"))
	if worktreeName == "" {
		return nil
	}
	mergeMode := strings.TrimSpace(legacyExtraString(stage.Extra, "merge_mode"))
	mergeBranch := strings.TrimSpace(legacyExtraString(stage.Extra, "merge_branch"))
	switch mergeMode {
	case "":
		if mergeBranch != "" {
			mergeMode = "remote"
		} else {
			mergeMode = "local"
		}
	case "local":
		mergeBranch = ""
	case "remote":
		if mergeBranch == "" {
			return nil
		}
	default:
		return nil
	}
	return &state.MergeRecoveryNode{
		WorktreeName: worktreeName,
		Mode:         mergeMode,
		Branch:       mergeBranch,
	}
}

func legacyExtraString(extra map[string]json.RawMessage, key string) string {
	if extra == nil {
		return ""
	}
	raw, ok := extra[key]
	if !ok {
		return ""
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return ""
	}
	return value
}

func cloneLegacyExtra(src map[string]json.RawMessage) map[string]json.RawMessage {
	if len(src) == 0 {
		return nil
	}
	cloned := make(map[string]json.RawMessage, len(src))
	for key, value := range src {
		cloned[key] = append(json.RawMessage(nil), value...)
	}
	return cloned
}
