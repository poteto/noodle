package loop

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/poteto/noodle/internal/ingest"
	"github.com/poteto/noodle/internal/state"
	"github.com/poteto/noodle/internal/stringx"
	"github.com/poteto/noodle/monitor"
	loopruntime "github.com/poteto/noodle/runtime"
)

func (l *Loop) reconcile(ctx context.Context) error {
	if !l.canonicalLoaded {
		if err := l.loadOrBootstrapCanonical(); err != nil {
			return l.classifySystemHard(
				"reconcile.load_canonical",
				"reconcile load canonical state failed",
				err,
			)
		}
	}
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
	if err := l.reconcileStaleDispatchStages(); err != nil {
		return l.classifySystemHard(
			"reconcile.stale_dispatch",
			"reconcile stale dispatch stages failed",
			err,
		)
	}
	if err := l.reconcileFailedOrders(); err != nil {
		return l.classifySystemHard(
			"reconcile.failed_orders",
			"reconcile failed orders archive failed",
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
				var activeStage *Stage
				if order, ok := orderMap[rs.OrderID]; ok {
					if idx, s := activeStageForOrder(order); idx >= 0 {
						stageIndex = idx
						activeStage = s
					}
				}
				l.emitEvent(ingest.EventSessionAdopted, map[string]any{
					"order_id":    rs.OrderID,
					"stage_index": stageIndex,
					"attempt_id":  "adopted-" + rs.SessionHandle.ID(),
					"session_id":  rs.SessionHandle.ID(),
				})
				l.trackAdoptedInSummary(activeStage)
			}
			l.cooks.adoptedSessions = append(l.cooks.adoptedSessions, rs.SessionHandle.ID())
		}
	}
	return nil
}

func (l *Loop) ensureScheduleOrderPresent() error {
	injectedOrderID := ""
	if err := l.mutateOrdersState(func(orders *OrdersFile) (bool, error) {
		if hasScheduleOrder(*orders) || hasScheduleBootstrapOrder(*orders) {
			return false, nil
		}
		if l.hasTaskType(scheduleOrderID) {
			orders.Orders = append(orders.Orders, scheduleOrder(l.config, ""))
			injectedOrderID = scheduleOrderID
			return true, nil
		}
		if l.hasTaskType("oops") {
			prependOrder(orders, scheduleBootstrapOopsOrder(l.config))
			injectedOrderID = scheduleBootstrapOrderID
			return true, nil
		}
		orders.Orders = append(orders.Orders, scheduleOrder(l.config, ""))
		injectedOrderID = scheduleOrderID
		return true, nil
	}); err != nil {
		return err
	}
	switch injectedOrderID {
	case scheduleOrderID:
		if orders, err := l.currentOrders(); err == nil {
			for _, order := range orders.Orders {
				if order.ID == scheduleOrderID {
					l.trackCanonicalOrder(order)
					break
				}
			}
		}
		l.logger.Info("startup injected schedule order")
	case scheduleBootstrapOrderID:
		if orders, err := l.currentOrders(); err == nil {
			for _, order := range orders.Orders {
				if order.ID == scheduleBootstrapOrderID {
					l.trackCanonicalOrder(order)
					break
				}
			}
		}
		l.logger.Info("startup injected schedule bootstrap order")
	}
	return nil
}

func (l *Loop) reconcileStaleDispatchStages() error {
	changed := false
	now := timeNowUTC(l.deps.Now)
	for orderID, order := range l.canonical.Orders {
		if order.Status.IsTerminal() {
			continue
		}
		if _, adopted := l.cooks.adoptedTargets[orderID]; adopted {
			continue
		}
		orderChanged := false
		for i := range order.Stages {
			stage := order.Stages[i]
			if stage.Status != state.StageDispatching && stage.Status != state.StageRunning {
				continue
			}
			stage.Status = state.StagePending
			if len(stage.Attempts) > 0 {
				last := len(stage.Attempts) - 1
				if stage.Attempts[last].Status == state.AttemptLaunching || stage.Attempts[last].Status == state.AttemptRunning {
					stage.Attempts[last].Status = state.AttemptCancelled
					stage.Attempts[last].CompletedAt = now
					if strings.TrimSpace(stage.Attempts[last].Error) == "" {
						stage.Attempts[last].Error = "restart cleared stale dispatch"
					}
				}
			}
			order.Stages[i] = stage
			orderChanged = true
			changed = true
			if strings.EqualFold(strings.TrimSpace(orderID), scheduleOrderID) {
				l.logger.Info("startup reset stale canonical schedule stage", "order", orderID, "stage", i)
			} else {
				l.logger.Info("startup reset stale canonical dispatch stage", "order", orderID, "stage", i)
			}
		}
		if !orderChanged {
			continue
		}
		order.Status = state.OrderPending
		order.UpdatedAt = now
		l.canonical.Orders[orderID] = order
	}
	if !changed {
		return nil
	}
	if err := l.mutateOrdersState(func(orders *OrdersFile) (bool, error) {
		ordersChanged := false
		for oi := range orders.Orders {
			order := &orders.Orders[oi]
			if _, adopted := l.cooks.adoptedTargets[order.ID]; adopted {
				continue
			}
			for si := range order.Stages {
				if order.Stages[si].Status != StageStatusActive {
					continue
				}
				order.Stages[si].Status = StageStatusPending
				ordersChanged = true
			}
		}
		return ordersChanged, nil
	}); err != nil {
		return err
	}
	return l.persistCanonicalCheckpoint()
}

// reconcileFailedOrders removes failed orders from orders.json and stores
// summaries so the scheduler can decide whether to re-queue the work.
func (l *Loop) reconcileFailedOrders() error {
	orders, err := l.currentOrders()
	if err != nil {
		return err
	}

	var failures []reconciledFailure
	for _, order := range orders.Orders {
		if order.Status != OrderStatusFailed {
			continue
		}
		if isScheduleOrder(order) {
			continue
		}
		f := reconciledFailure{
			OrderID: order.ID,
			Title:   order.Title,
		}
		for _, stage := range order.Stages {
			if stage.Status == StageStatusFailed {
				f.TaskKey = stage.TaskKey
				f.Reason = extraString(stage.Extra, "failure_reason")
				break
			}
		}
		if f.Reason == "" && f.TaskKey != "" {
			f.Reason = "stage " + f.TaskKey + " failed"
		}
		failures = append(failures, f)
	}

	if len(failures) == 0 {
		return nil
	}

	if err := l.mutateOrdersState(func(orders *OrdersFile) (bool, error) {
		kept := orders.Orders[:0]
		for _, order := range orders.Orders {
			if order.Status == OrderStatusFailed && !isScheduleOrder(order) {
				continue
			}
			kept = append(kept, order)
		}
		orders.Orders = kept
		return true, nil
	}); err != nil {
		return err
	}

	l.reconciledFailures = failures
	for _, f := range failures {
		l.logger.Info("startup archived failed order",
			"order", f.OrderID, "title", f.Title, "task_key", f.TaskKey)
	}
	return nil
}

func timeNowUTC(nowFn func() time.Time) time.Time {
	if nowFn != nil {
		return nowFn().UTC()
	}
	return time.Now().UTC()
}

// reconciledFailure holds a summary of a failed order archived during startup.
type reconciledFailure struct {
	OrderID string
	Title   string
	TaskKey string
	Reason  string
}

// mergeRecoveryStage holds the canonical recovery data for a stage stuck in
// merging during a crash window.
type mergeRecoveryStage struct {
	orderID     string
	order       state.OrderNode
	stage       state.StageNode
	checkBranch string
}

// reconcileMergingStages recovers stages stuck in "merging" status after a
// crash. For each merging stage it reads the canonical merge recovery record and
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

// collectMergingStages reads canonical state and returns recovery data for
// every stage in "merging" status.
func (l *Loop) collectMergingStages() ([]mergeRecoveryStage, error) {
	var result []mergeRecoveryStage
	for orderID, order := range l.canonical.Orders {
		if order.Status != state.OrderActive {
			continue
		}
		for _, stage := range order.Stages {
			if stage.Status != state.StageMerging {
				continue
			}
			checkBranch := ""
			if stage.Merge != nil {
				checkBranch = strings.TrimSpace(stage.Merge.WorktreeName)
				if strings.EqualFold(strings.TrimSpace(stage.Merge.Mode), "remote") && strings.TrimSpace(stage.Merge.Branch) != "" {
					checkBranch = strings.TrimSpace(stage.Merge.Branch)
				}
			}
			result = append(result, mergeRecoveryStage{
				orderID:     orderID,
				order:       order,
				stage:       stage,
				checkBranch: checkBranch,
			})
		}
	}
	return result, nil
}

// reconcileOneMergingStage applies the correct recovery action for a single
// merging stage based on its metadata and branch state.
func (l *Loop) reconcileOneMergingStage(md mergeRecoveryStage) error {
	if md.stage.Merge == nil || strings.TrimSpace(md.stage.Merge.WorktreeName) == "" {
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

// failMissingMetadataStage fails a merging stage that has no merge metadata.
func (l *Loop) failMissingMetadataStage(md mergeRecoveryStage) error {
	l.logger.Warn("merging stage missing metadata, failing",
		"order", md.orderID, "stage", md.stage.StageIndex)
	return l.failMergingStage(md.orderID, md.stage.StageIndex,
		"merging stage missing merge metadata after crash")
}

// handleAlreadyAdoptedStage resets a merging stage back to active when a live
// session was recovered for the order.
func (l *Loop) handleAlreadyAdoptedStage(md mergeRecoveryStage) error {
	l.logger.Info("merging stage has live session, resetting to active",
		"order", md.orderID, "stage", md.stage.StageIndex)
	order := l.canonical.Orders[md.orderID]
	stage := order.Stages[md.stage.StageIndex]
	stage.Status = state.StageRunning
	stage.Merge = nil
	order.Stages[md.stage.StageIndex] = stage
	order.Status = state.OrderActive
	order.UpdatedAt = timeNowUTC(l.deps.Now)
	l.canonical.Orders[md.orderID] = order
	if err := l.persistCanonicalCheckpoint(); err != nil {
		return err
	}
	return l.mirrorLegacyOrderFromCanonical(md.orderID)
}

// handleAlreadyMergedStage advances a merging stage whose branch is already
// an ancestor of HEAD (merge completed before crash).
func (l *Loop) handleAlreadyMergedStage(md mergeRecoveryStage) error {
	l.logger.Info("merging stage branch already merged, advancing",
		"order", md.orderID, "stage", md.stage.StageIndex, "branch", md.checkBranch)
	cook := &cookHandle{
		cookIdentity: cookIdentity{
				orderID:    md.orderID,
				stageIndex: md.stage.StageIndex,
				stage: Stage{
					TaskKey: md.stage.TaskKey,
					Skill:   md.stage.Skill,
					Runtime: md.stage.Runtime,
				},
		},
		worktreeName: md.stage.Merge.WorktreeName,
		worktreePath: l.worktreePath(md.stage.Merge.WorktreeName),
	}
	if err := l.emitEventChecked(ingest.EventMergeCompleted, map[string]any{
		"order_id":    md.orderID,
		"stage_index": md.stage.StageIndex,
	}); err != nil {
		return err
	}
	return l.advanceAndPersist(context.Background(), cook)
}

// requeueStaleMerge re-enqueues a merge for a stage whose branch still exists
// but was not merged before the crash.
func (l *Loop) requeueStaleMerge(md mergeRecoveryStage) error {
	l.logger.Info("merging stage branch exists, re-enqueueing merge",
		"order", md.orderID, "stage", md.stage.StageIndex, "branch", md.checkBranch)
	cook := &cookHandle{
		cookIdentity: cookIdentity{
				orderID:    md.orderID,
				stageIndex: md.stage.StageIndex,
				stage: Stage{
					TaskKey: md.stage.TaskKey,
					Skill:   md.stage.Skill,
					Runtime: md.stage.Runtime,
				},
		},
		worktreeName: md.stage.Merge.WorktreeName,
		worktreePath: l.worktreePath(md.stage.Merge.WorktreeName),
		session:      &adoptedSession{id: "crash-recovery", status: "completed"},
	}
	if l.mergeQueue != nil {
		l.mergeQueue.Enqueue(MergeRequest{Cook: cook})
		return nil
	}
	if err := l.mergeCookWorktree(context.Background(), cook); err != nil {
		if conflictErr := l.handleMergeConflict(cook, err); conflictErr != nil {
			return conflictErr
		}
		return nil
	}
	if err := l.emitEventChecked(ingest.EventMergeCompleted, map[string]any{
		"order_id":    md.orderID,
		"stage_index": md.stage.StageIndex,
	}); err != nil {
		return err
	}
	return l.advanceAndPersist(context.Background(), cook)
}

// failMissingBranchStage fails a merging stage whose branch no longer exists
// and has no live session.
func (l *Loop) failMissingBranchStage(md mergeRecoveryStage) error {
	l.logger.Warn("merging stage branch not found, failing",
		"order", md.orderID, "stage", md.stage.StageIndex, "branch", md.checkBranch)
	return l.failMergingStage(md.orderID, md.stage.StageIndex,
		"merge branch "+md.checkBranch+" not found after crash")
}

// failMergingStage transitions a stuck merging stage to failed through the
// canonical reducer and mirrors the legacy projection afterward.
func (l *Loop) failMergingStage(orderID string, stageIdx int, reason string) error {
	order, ok := l.canonical.Orders[orderID]
	if !ok {
		return fmt.Errorf("canonical order %q missing for merging-stage failure", orderID)
	}
	if stageIdx < 0 || stageIdx >= len(order.Stages) {
		return fmt.Errorf("canonical stage %d missing for order %q", stageIdx, orderID)
	}
	stage := order.Stages[stageIdx]
	cook := &cookHandle{
		cookIdentity: cookIdentity{
				orderID:    orderID,
				stageIndex: stageIdx,
				stage: Stage{
					TaskKey: stage.TaskKey,
					Skill:   stage.Skill,
					Runtime: stage.Runtime,
				},
		},
		worktreeName: latestStageMergeWorktree(stage),
	}
	if err := l.emitEventChecked(ingest.EventStageFailed, map[string]any{
		"order_id":      orderID,
		"stage_index":   stageIdx,
		"attempt_id":    latestAttemptID(stage),
		"session_id":    latestSessionID(stage),
		"worktree_name": latestStageWorktree(stage),
		"error":         reason,
	}); err != nil {
		return err
	}
	if err := l.mirrorLegacyOrderFromCanonical(orderID); err != nil {
		return err
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

func latestAttemptID(stage state.StageNode) string {
	for i := len(stage.Attempts) - 1; i >= 0; i-- {
		if id := strings.TrimSpace(stage.Attempts[i].AttemptID); id != "" {
			return id
		}
	}
	return ""
}

func latestSessionID(stage state.StageNode) string {
	for i := len(stage.Attempts) - 1; i >= 0; i-- {
		if id := strings.TrimSpace(stage.Attempts[i].SessionID); id != "" {
			return id
		}
	}
	return ""
}

func latestStageWorktree(stage state.StageNode) string {
	if stage.Merge != nil && strings.TrimSpace(stage.Merge.WorktreeName) != "" {
		return strings.TrimSpace(stage.Merge.WorktreeName)
	}
	for i := len(stage.Attempts) - 1; i >= 0; i-- {
		if name := strings.TrimSpace(stage.Attempts[i].WorktreeName); name != "" {
			return name
		}
	}
	return ""
}

func latestStageMergeWorktree(stage state.StageNode) string {
	if stage.Merge == nil {
		return latestStageWorktree(stage)
	}
	return strings.TrimSpace(stage.Merge.WorktreeName)
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

// branchExists checks if a local or remote branch ref exists.
func branchExists(projectDir string, branch string) bool {
	branch = strings.TrimSpace(branch)
	if branch == "" {
		return false
	}
	cmd := exec.Command("git", "-C", projectDir, "for-each-ref", "--format=%(refname)",
		"refs/heads/"+branch,
		"refs/remotes/origin/"+branch,
		"refs/remotes/"+branch,
	)
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) != ""
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
		var meta struct {
			Status string `json:"status"`
		}
		if err := json.Unmarshal(data, &meta); err != nil {
			continue
		}
		if stringx.Normalize(meta.Status) != "running" {
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

func (s *adoptedSession) ID() string     { return s.id }
func (s *adoptedSession) Status() string { return s.status }
func (s *adoptedSession) Outcome() loopruntime.SessionOutcome {
	return loopruntime.SessionOutcome{Status: loopruntime.SessionStatus(s.status)}
}
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
