package loop

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/poteto/noodle/internal/ingest"
	"github.com/poteto/noodle/monitor"
	loopruntime "github.com/poteto/noodle/runtime"
)

func (l *Loop) reconcile(ctx context.Context) error {
	if err := l.loadPendingReview(); err != nil {
		return l.classifySystemHard(
			"reconcile.load_pending_review",
			"reconcile load pending review failed",
			err,
		)
	}
	if err := l.reconcilePendingReview(); err != nil {
		return l.classifySystemHard(
			"reconcile.pending_review_prune",
			"reconcile pending review prune failed",
			err,
		)
	}
	if err := os.MkdirAll(filepath.Join(l.runtimeDir, "sessions"), 0o755); err != nil {
		return l.classifySystemHard(
			"reconcile.sessions_dir",
			"reconcile create sessions directory failed",
			err,
		)
	}

	if err := l.recoverAdoptedSessions(ctx); err != nil {
		return err
	}

	if err := l.ensureScheduleOrderPresent(); err != nil {
		return l.classifySystemHard(
			"reconcile.ensure_schedule",
			"reconcile ensure schedule order failed",
			err,
		)
	}
	if err := l.reconcileStaleActiveStages(); err != nil {
		return l.classifySystemHard(
			"reconcile.stale_active",
			"reconcile stale active stages failed",
			err,
		)
	}

	if len(l.cooks.adoptedSessions) > 0 {
		tickets := monitor.NewEventTicketMaterializer(l.runtimeDir)
		_ = tickets.Materialize(ctx, l.cooks.adoptedSessions)
	}

	// Recover stages stuck in "merging" status from a previous crash.
	// Must run after adopted session index is built.
	if err := l.reconcileMergingStages(); err != nil {
		return l.classifySystemHard(
			"reconcile.merging",
			"reconcile merging stages failed",
			err,
		)
	}

	return nil
}

// recoverAdoptedSessions recovers live sessions from all registered runtimes
// and builds the adopted targets/sessions index.
func (l *Loop) recoverAdoptedSessions(ctx context.Context) error {
	l.cooks.adoptedTargets = map[string]string{}
	l.cooks.adoptedSessions = l.cooks.adoptedSessions[:0]

	orders, err := l.currentOrders()
	if err != nil {
		return l.classifySystemHard(
			"reconcile.orders_for_adoption",
			"reconcile load orders for session adoption failed",
			err,
		)
	}
	orderMap := make(map[string]Order, len(orders.Orders))
	for _, o := range orders.Orders {
		orderMap[o.ID] = o
	}

	for _, rt := range l.deps.Runtimes {
		recovered, err := rt.Recover(ctx)
		if err != nil {
			l.classifyDegrade(
				"reconcile.runtime_recover",
				"runtime recovery degraded",
				err,
			)
			l.logger.Warn("runtime recovery failed", "err", err)
			continue
		}
		for _, rs := range recovered {
			if rs.OrderID != "" {
				l.cooks.adoptedTargets[rs.OrderID] = rs.SessionHandle.ID()
				stageIndex := 0
				if order, ok := orderMap[rs.OrderID]; ok {
					if idx, _ := activeStageForOrder(order); idx >= 0 {
						stageIndex = idx
					}
				}
				l.emitEvent(ingest.EventSessionAdopted, map[string]any{
					"order_id":    rs.OrderID,
					"stage_index": stageIndex,
					"attempt_id":  "adopted-" + rs.SessionHandle.ID(),
					"session_id":  rs.SessionHandle.ID(),
				})
			}
			l.cooks.adoptedSessions = append(l.cooks.adoptedSessions, rs.SessionHandle.ID())
		}
	}
	return nil
}

func (l *Loop) ensureScheduleOrderPresent() error {
	injected := false
	if err := l.mutateOrdersState(func(orders *OrdersFile) (bool, error) {
		if hasScheduleOrder(*orders) {
			return false, nil
		}
		orders.Orders = append(orders.Orders, scheduleOrder(l.config, ""))
		injected = true
		return true, nil
	}); err != nil {
		return err
	}
	if injected {
		l.logger.Info("startup injected schedule order")
	}
	return nil
}

func (l *Loop) reconcileStaleActiveStages() error {
	return l.mutateOrdersState(func(orders *OrdersFile) (bool, error) {
		changed := false
		for oi := range orders.Orders {
			order := &orders.Orders[oi]
			if order.Status != OrderStatusActive {
				continue
			}
			if _, adopted := l.cooks.adoptedTargets[order.ID]; adopted {
				continue
			}
			for si := range order.Stages {
				if order.Stages[si].Status == StageStatusActive {
					order.Stages[si].Status = StageStatusPending
					changed = true
					if isScheduleOrder(*order) {
						l.logger.Info("startup reset stale active schedule stage", "order", order.ID, "stage", si)
					} else {
						l.logger.Info("startup reset stale active stage", "order", order.ID, "stage", si)
					}
				}
			}
		}
		return changed, nil
	})
}

// mergeMetadata holds the fields extracted from a merging stage's Extra map.
type mergeMetadata struct {
	orderID     string
	stageIdx    int
	stage       Stage
	order       Order
	wtName      string
	mode        string
	branch      string
	checkBranch string
}

// reconcileMergingStages recovers stages stuck in "merging" status after a
// crash. For each merging stage it reads the merge metadata from Extra and
// decides:
//   - metadata missing -> fail the stage
//   - live session adopted -> keep as "active"
//   - branch already merged (ancestor of HEAD) -> advance to completed
//   - branch exists but not merged -> re-enqueue MergeRequest
//   - branch gone + no live session -> fail the stage
func (l *Loop) reconcileMergingStages() error {
	merging, err := l.collectMergingStages()
	if err != nil {
		return err
	}
	for _, md := range merging {
		if err := l.reconcileOneMergingStage(md); err != nil {
			return err
		}
	}
	return nil
}

// collectMergingStages reads orders and returns metadata for every stage in
// "merging" status.
func (l *Loop) collectMergingStages() ([]mergeMetadata, error) {
	orders, err := l.currentOrders()
	if err != nil {
		return nil, err
	}
	var result []mergeMetadata
	for _, order := range orders.Orders {
		if order.Status != OrderStatusActive {
			continue
		}
		for si, s := range order.Stages {
			if s.Status == StageStatusMerging {
				result = append(result, extractMergeMetadata(order, si, s))
			}
		}
	}
	return result, nil
}

// reconcileOneMergingStage applies the correct recovery action for a single
// merging stage based on its metadata and branch state.
func (l *Loop) reconcileOneMergingStage(md mergeMetadata) error {
	if md.wtName == "" && md.branch == "" {
		return l.failMissingMetadataStage(md)
	}
	if _, adopted := l.cooks.adoptedTargets[md.orderID]; adopted {
		return l.handleAlreadyAdoptedStage(md)
	}
	if isBranchMerged(l.projectDir, md.checkBranch) {
		return l.handleAlreadyMergedStage(md)
	}
	if branchExists(l.projectDir, md.checkBranch) {
		return l.requeueStaleMerge(md)
	}
	return l.failMissingBranchStage(md)
}

// extractMergeMetadata reads merge-related fields from a stage's Extra map
// and derives the branch to check.
func extractMergeMetadata(order Order, stageIdx int, stage Stage) mergeMetadata {
	wtName := extraString(stage.Extra, mergeExtraWorktree)
	mode := extraString(stage.Extra, mergeExtraMode)
	branch := extraString(stage.Extra, mergeExtraBranch)

	checkBranch := wtName
	if mode == "remote" && branch != "" {
		checkBranch = branch
	}

	return mergeMetadata{
		orderID:     order.ID,
		stageIdx:    stageIdx,
		stage:       stage,
		order:       order,
		wtName:      wtName,
		mode:        mode,
		branch:      branch,
		checkBranch: checkBranch,
	}
}

// failMissingMetadataStage fails a merging stage that has no merge metadata.
func (l *Loop) failMissingMetadataStage(md mergeMetadata) error {
	l.logger.Warn("merging stage missing metadata, failing",
		"order", md.orderID, "stage", md.stageIdx)
	return l.failMergingStage(md.orderID, md.stageIdx,
		"merging stage missing merge metadata after crash")
}

// handleAlreadyAdoptedStage resets a merging stage back to active when a live
// session was recovered for the order.
func (l *Loop) handleAlreadyAdoptedStage(md mergeMetadata) error {
	l.logger.Info("merging stage has live session, resetting to active",
		"order", md.orderID, "stage", md.stageIdx)
	return l.persistOrderStageStatus(md.orderID, md.stageIdx, StageStatusActive)
}

// handleAlreadyMergedStage advances a merging stage whose branch is already
// an ancestor of HEAD (merge completed before crash).
func (l *Loop) handleAlreadyMergedStage(md mergeMetadata) error {
	l.logger.Info("merging stage branch already merged, advancing",
		"order", md.orderID, "stage", md.stageIdx, "branch", md.checkBranch)
	cook := &cookHandle{
		cookIdentity: cookIdentity{
			orderID:    md.orderID,
			stageIndex: md.stageIdx,
			stage:      md.stage,
			plan:       md.order.Plan,
		},
		orderStatus: md.order.Status,
	}
	l.emitEvent(ingest.EventStageCompleted, map[string]any{
		"order_id":    md.orderID,
		"stage_index": md.stageIdx,
		"mergeable":   true,
	})
	l.emitEvent(ingest.EventMergeCompleted, map[string]any{
		"order_id":    md.orderID,
		"stage_index": md.stageIdx,
	})
	return l.advanceAndPersist(context.Background(), cook)
}

// requeueStaleMerge re-enqueues a merge for a stage whose branch still exists
// but was not merged before the crash.
func (l *Loop) requeueStaleMerge(md mergeMetadata) error {
	l.logger.Info("merging stage branch exists, re-enqueueing merge",
		"order", md.orderID, "stage", md.stageIdx, "branch", md.checkBranch)
	cook := &cookHandle{
		cookIdentity: cookIdentity{
			orderID:    md.orderID,
			stageIndex: md.stageIdx,
			stage:      md.stage,
			plan:       md.order.Plan,
		},
		orderStatus:  md.order.Status,
		worktreeName: md.wtName,
		worktreePath: l.worktreePath(md.wtName),
		session:      &adoptedSession{id: "crash-recovery", status: "completed"},
	}
	l.emitEvent(ingest.EventStageCompleted, map[string]any{
		"order_id":    md.orderID,
		"stage_index": md.stageIdx,
		"mergeable":   true,
	})
	if l.mergeQueue != nil {
		l.mergeQueue.Enqueue(MergeRequest{Cook: cook})
		return nil
	}
	if err := l.mergeCookWorktree(context.Background(), cook); err != nil {
		l.logger.Warn("crash recovery merge failed, failing stage",
			"order", md.orderID, "err", err)
		return l.failMergingStage(md.orderID, md.stageIdx,
			"crash recovery merge failed: "+err.Error())
	}
	l.emitEvent(ingest.EventMergeCompleted, map[string]any{
		"order_id":    md.orderID,
		"stage_index": md.stageIdx,
	})
	return l.advanceAndPersist(context.Background(), cook)
}

// failMissingBranchStage fails a merging stage whose branch no longer exists
// and has no live session.
func (l *Loop) failMissingBranchStage(md mergeMetadata) error {
	l.logger.Warn("merging stage branch not found, failing",
		"order", md.orderID, "stage", md.stageIdx, "branch", md.checkBranch)
	return l.failMergingStage(md.orderID, md.stageIdx,
		"merge branch "+md.checkBranch+" not found after crash")
}

// failMergingStage transitions a stuck merging stage to failed via failStage.
func (l *Loop) failMergingStage(orderID string, stageIdx int, reason string) error {
	if err := l.mutateOrdersState(func(orders *OrdersFile) (bool, error) {
		updated, err := failStage(*orders, orderID, reason)
		if err != nil {
			return false, &mergingStageMarkError{cause: err}
		}
		*orders = updated
		return true, nil
	}); err != nil {
		if markErr, ok := err.(*mergingStageMarkError); ok {
			return l.classifySystemHard(
				"reconcile.fail_merging_stage_mark",
				"reconcile mark merging stage failed",
				markErr.cause,
			)
		}
		return l.classifySystemHard(
			"reconcile.fail_merging_stage_persist",
			"reconcile persist merging-stage failure",
			err,
		)
	}
	cook := &cookHandle{
		cookIdentity: cookIdentity{
			orderID:    orderID,
			stageIndex: stageIdx,
		},
	}
	l.recordStageFailure(cook, reason, OrderFailureClassStageTerminal, nil)
	l.classifyOrderHard(
		"reconcile.merging_stage_terminal",
		OrderFailureClassStageTerminal,
		orderID,
		stageIdx,
		reason,
		nil,
	)
	return nil
}

type mergingStageMarkError struct {
	cause error
}

func (e *mergingStageMarkError) Error() string {
	return e.cause.Error()
}

// extraString reads a string value from a stage's Extra map.
func extraString(extra map[string]json.RawMessage, key string) string {
	if extra == nil {
		return ""
	}
	raw, ok := extra[key]
	if !ok {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return ""
	}
	return s
}

// isBranchMerged checks if a branch is an ancestor of HEAD (already merged).
func isBranchMerged(projectDir string, branch string) bool {
	branch = strings.TrimSpace(branch)
	if branch == "" {
		return false
	}
	cmd := exec.Command("git", "-C", projectDir, "merge-base", "--is-ancestor", branch, "HEAD")
	return cmd.Run() == nil
}

// branchExists checks if a local branch ref exists.
func branchExists(projectDir string, branch string) bool {
	branch = strings.TrimSpace(branch)
	if branch == "" {
		return false
	}
	cmd := exec.Command("git", "-C", projectDir, "show-ref", "--verify", "--quiet", "refs/heads/"+branch)
	return cmd.Run() == nil
}

func (l *Loop) refreshAdoptedTargets() {
	if len(l.cooks.adoptedTargets) == 0 {
		return
	}
	nextTargets := make(map[string]string, len(l.cooks.adoptedTargets))
	nextSessions := make([]string, 0, len(l.cooks.adoptedTargets))
	for target, sessionID := range l.cooks.adoptedTargets {
		metaPath := filepath.Join(l.runtimeDir, "sessions", sessionID, "meta.json")
		data, err := os.ReadFile(metaPath)
		if err != nil {
			continue
		}
		if !strings.Contains(string(data), `"status":"running"`) {
			continue
		}
		if !monitor.SessionPIDAlive(l.runtimeDir, sessionID) {
			continue
		}
		nextTargets[target] = sessionID
		nextSessions = append(nextSessions, sessionID)
	}
	l.cooks.adoptedTargets = nextTargets
	l.cooks.adoptedSessions = nextSessions
}

// adoptedSession is a minimal SessionHandle for crash-recovery merge scenarios
// where no live dispatcher session exists.
type adoptedSession struct {
	id     string
	status string
}

func (s *adoptedSession) ID() string          { return s.id }
func (s *adoptedSession) Status() string      { return s.status }
func (s *adoptedSession) TotalCost() float64  { return 0 }
func (s *adoptedSession) Terminate() error    { return nil }
func (s *adoptedSession) ForceKill() error    { return nil }
func (s *adoptedSession) VerdictPath() string { return "" }
func (s *adoptedSession) Controller() loopruntime.AgentController {
	return loopruntime.NoopController()
}

func (s *adoptedSession) Done() <-chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}
