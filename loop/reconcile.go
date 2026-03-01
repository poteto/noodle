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
		return err
	}
	// Prune pending reviews for orders that no longer exist in orders.json.
	// This handles the crash window between advancing orders.json and updating
	// pending-review.json (finding #5).
	if err := l.reconcilePendingReview(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(l.runtimeDir, "sessions"), 0o755); err != nil {
		return err
	}

	l.cooks.adoptedTargets = map[string]string{}
	l.cooks.adoptedSessions = l.cooks.adoptedSessions[:0]

	// Recover sessions from all registered runtimes.
	// Build order lookup for stage index derivation during adoption.
	orders, _ := l.currentOrders()
	orderMap := make(map[string]Order, len(orders.Orders))
	for _, o := range orders.Orders {
		orderMap[o.ID] = o
	}

	for _, rt := range l.deps.Runtimes {
		recovered, err := rt.Recover(ctx)
		if err != nil {
			l.logger.Warn("runtime recovery failed", "err", err)
			continue
		}
		for _, rs := range recovered {
			if rs.OrderID != "" {
				l.cooks.adoptedTargets[rs.OrderID] = rs.SessionHandle.ID()

				// Derive stage index from order state instead of hardcoding 0.
				stageIndex := 0
				if order, ok := orderMap[rs.OrderID]; ok {
					if idx, _ := activeStageForOrder(order); idx >= 0 {
						stageIndex = idx
					}
				}

				// Emit V2 canonical state event for session adoption.
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

	// Scheduler must be present after startup reconciliation.
	if err := l.ensureScheduleOrderPresent(); err != nil {
		return err
	}
	// If any stage was left "active" by a crash but no live session was
	// recovered for that order, reset it to pending so startup can dispatch.
	if err := l.reconcileStaleActiveStages(); err != nil {
		return err
	}

	if len(l.cooks.adoptedSessions) > 0 {
		tickets := monitor.NewEventTicketMaterializer(l.runtimeDir)
		_ = tickets.Materialize(ctx, l.cooks.adoptedSessions)
	}

	// Recover stages stuck in "merging" status from a previous crash.
	// Must run after adopted session index is built.
	if err := l.reconcileMergingStages(); err != nil {
		return err
	}

	return nil
}

func (l *Loop) ensureScheduleOrderPresent() error {
	orders, err := l.currentOrders()
	if err != nil {
		return err
	}
	if hasScheduleOrder(orders) {
		return nil
	}
	orders.Orders = append(orders.Orders, scheduleOrder(l.config, ""))
	if err := l.writeOrdersState(orders); err != nil {
		return err
	}
	l.logger.Info("startup injected schedule order")
	return nil
}

func (l *Loop) reconcileStaleActiveStages() error {
	orders, err := l.currentOrders()
	if err != nil {
		return err
	}

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

	if !changed {
		return nil
	}
	return l.writeOrdersState(orders)
}

// reconcileMergingStages recovers stages stuck in "merging" status after a
// crash. For each merging stage it reads the merge metadata from Extra and
// decides:
//   - metadata missing → fail the stage
//   - branch already merged (ancestor of HEAD) → advance to completed
//   - branch exists but not merged → re-enqueue MergeRequest
//   - branch gone + no live session → fail the stage
//   - live session adopted → keep as "active"
func (l *Loop) reconcileMergingStages() error {
	orders, err := l.currentOrders()
	if err != nil {
		return err
	}

	type mergingStage struct {
		orderIdx int
		stageIdx int
		stage    Stage
		order    Order
	}
	var merging []mergingStage

	for oi, order := range orders.Orders {
		if order.Status != OrderStatusActive {
			continue
		}
		for si, s := range order.Stages {
			if s.Status == StageStatusMerging {
				merging = append(merging, mergingStage{
					orderIdx: oi,
					stageIdx: si,
					stage:    s,
					order:    order,
				})
			}
		}
	}

	if len(merging) == 0 {
		return nil
	}

	for _, ms := range merging {
		wtName := extraString(ms.stage.Extra, mergeExtraWorktree)
		mode := extraString(ms.stage.Extra, mergeExtraMode)
		branch := extraString(ms.stage.Extra, mergeExtraBranch)

		// Missing metadata — can't recover.
		if wtName == "" && branch == "" {
			l.logger.Warn("merging stage missing metadata, failing", "order", ms.order.ID, "stage", ms.stageIdx)
			if err := l.failMergingStage(ms.order.ID, ms.stageIdx, "merging stage missing merge metadata after crash"); err != nil {
				return err
			}
			continue
		}

		// Check if a live session was adopted for this order.
		if _, adopted := l.cooks.adoptedTargets[ms.order.ID]; adopted {
			l.logger.Info("merging stage has live session, resetting to active", "order", ms.order.ID, "stage", ms.stageIdx)
			if err := l.persistOrderStageStatus(ms.order.ID, ms.stageIdx, StageStatusActive); err != nil {
				return err
			}
			continue
		}

		// Determine which branch to check — remote mode uses the named branch,
		// local mode uses the worktree branch name.
		checkBranch := wtName
		if mode == "remote" && branch != "" {
			checkBranch = branch
		}

		if isBranchMerged(l.projectDir, checkBranch) {
			l.logger.Info("merging stage branch already merged, advancing", "order", ms.order.ID, "stage", ms.stageIdx, "branch", checkBranch)
			cook := &cookHandle{
				cookIdentity: cookIdentity{
					orderID:    ms.order.ID,
					stageIndex: ms.stageIdx,
					stage:      ms.stage,
					plan:       ms.order.Plan,
				},
				orderStatus: ms.order.Status,
			}
			// Reconstruct canonical state: stage was mergeable and merge completed before crash.
			l.emitEvent(ingest.EventStageCompleted, map[string]any{
				"order_id":    ms.order.ID,
				"stage_index": ms.stageIdx,
				"mergeable":   true,
			})
			l.emitEvent(ingest.EventMergeCompleted, map[string]any{
				"order_id":    ms.order.ID,
				"stage_index": ms.stageIdx,
			})
			if err := l.advanceAndPersist(context.Background(), cook); err != nil {
				return err
			}
			continue
		}

		// Branch not merged yet — check if it still exists.
		if branchExists(l.projectDir, checkBranch) {
			l.logger.Info("merging stage branch exists, re-enqueueing merge", "order", ms.order.ID, "stage", ms.stageIdx, "branch", checkBranch)
			cook := &cookHandle{
				cookIdentity: cookIdentity{
					orderID:    ms.order.ID,
					stageIndex: ms.stageIdx,
					stage:      ms.stage,
					plan:       ms.order.Plan,
				},
				orderStatus:  ms.order.Status,
				worktreeName: wtName,
				worktreePath: l.worktreePath(wtName),
				session:      &adoptedSession{id: "crash-recovery", status: "completed"},
			}
			// Emit canonical stage_completed (mergeable) before merge attempt.
			l.emitEvent(ingest.EventStageCompleted, map[string]any{
				"order_id":    ms.order.ID,
				"stage_index": ms.stageIdx,
				"mergeable":   true,
			})
			if l.mergeQueue != nil {
				// Queued path: drainMergeResults emits merge_completed.
				l.mergeQueue.Enqueue(MergeRequest{Cook: cook})
			} else {
				if err := l.mergeCookWorktree(context.Background(), cook); err != nil {
					l.logger.Warn("crash recovery merge failed, failing stage", "order", ms.order.ID, "err", err)
					if failErr := l.failMergingStage(ms.order.ID, ms.stageIdx, "crash recovery merge failed: "+err.Error()); failErr != nil {
						return failErr
					}
					continue
				}
				l.emitEvent(ingest.EventMergeCompleted, map[string]any{
					"order_id":    ms.order.ID,
					"stage_index": ms.stageIdx,
				})
				if err := l.advanceAndPersist(context.Background(), cook); err != nil {
					return err
				}
			}
			continue
		}

		// Branch gone, no live session — fail.
		l.logger.Warn("merging stage branch not found, failing", "order", ms.order.ID, "stage", ms.stageIdx, "branch", checkBranch)
		if err := l.failMergingStage(ms.order.ID, ms.stageIdx, "merge branch "+checkBranch+" not found after crash"); err != nil {
			return err
		}
	}

	return nil
}

// failMergingStage transitions a stuck merging stage to failed via failStage.
func (l *Loop) failMergingStage(orderID string, stageIdx int, reason string) error {
	orders, err := l.currentOrders()
	if err != nil {
		return err
	}
	orders, err = failStage(orders, orderID, reason)
	if err != nil {
		return err
	}
	if err := l.writeOrdersState(orders); err != nil {
		return err
	}
	_ = l.events.Emit(LoopEventStageFailed, StageFailedPayload{
		OrderID:    orderID,
		StageIndex: stageIdx,
		Reason:     reason,
	})
	_ = l.events.Emit(LoopEventOrderFailed, OrderFailedPayload{
		OrderID: orderID,
		Reason:  reason,
	})
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
func (s *adoptedSession) Kill() error         { return nil }
func (s *adoptedSession) VerdictPath() string { return "" }
func (s *adoptedSession) Controller() loopruntime.AgentController {
	return loopruntime.NoopController()
}

func (s *adoptedSession) Done() <-chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}
