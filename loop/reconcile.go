package loop

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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
	for _, rt := range l.deps.Runtimes {
		recovered, err := rt.Recover(ctx)
		if err != nil {
			l.logger.Warn("runtime recovery failed", "err", err)
			continue
		}
		for _, rs := range recovered {
			if rs.OrderID != "" {
				l.cooks.adoptedTargets[rs.OrderID] = rs.SessionHandle.ID()
			}
			l.cooks.adoptedSessions = append(l.cooks.adoptedSessions, rs.SessionHandle.ID())
		}
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

	// Load pending retries AFTER reconcile builds the live-session index
	// (adoptedTargets). This ensures we don't retry orders that already
	// have a recovered session handling them.
	if err := l.loadPendingRetry(); err != nil {
		return err
	}
	if err := l.reconcilePendingRetry(); err != nil {
		return err
	}
	return nil
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
		orderIdx    int
		stageIdx    int
		isOnFailure bool
		stage       Stage
		order       Order
	}
	var merging []mergingStage

	for oi, order := range orders.Orders {
		if order.Status != OrderStatusActive && order.Status != OrderStatusFailing {
			continue
		}
		scanStages := func(stages []Stage, isOnFailure bool) {
			for si, s := range stages {
				if s.Status == StageStatusMerging {
					merging = append(merging, mergingStage{
						orderIdx:    oi,
						stageIdx:    si,
						isOnFailure: isOnFailure,
						stage:       s,
						order:       order,
					})
				}
			}
		}
		scanStages(order.Stages, false)
		if order.Status == OrderStatusFailing {
			scanStages(order.OnFailure, true)
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
			if err := l.failMergingStage(ms.order.ID, ms.stageIdx, ms.isOnFailure, "merging stage missing merge metadata after crash"); err != nil {
				return err
			}
			continue
		}

		// Check if a live session was adopted for this order.
		if _, adopted := l.cooks.adoptedTargets[ms.order.ID]; adopted {
			l.logger.Info("merging stage has live session, resetting to active", "order", ms.order.ID, "stage", ms.stageIdx)
			if err := l.persistOrderStageStatus(ms.order.ID, ms.stageIdx, ms.isOnFailure, StageStatusActive); err != nil {
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
				isOnFailure: ms.isOnFailure,
				orderStatus: ms.order.Status,
			}
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
				isOnFailure:  ms.isOnFailure,
				orderStatus:  ms.order.Status,
				worktreeName: wtName,
				worktreePath: l.worktreePath(wtName),
				session:      &adoptedSession{id: "crash-recovery", status: "completed"},
			}
			if l.mergeQueue != nil {
				l.mergeQueue.Enqueue(MergeRequest{Cook: cook})
			} else {
				if err := l.mergeCookWorktree(context.Background(), cook); err != nil {
					l.logger.Warn("crash recovery merge failed, failing stage", "order", ms.order.ID, "err", err)
					if failErr := l.failMergingStage(ms.order.ID, ms.stageIdx, ms.isOnFailure, "crash recovery merge failed: "+err.Error()); failErr != nil {
						return failErr
					}
					continue
				}
				if err := l.advanceAndPersist(context.Background(), cook); err != nil {
					return err
				}
			}
			continue
		}

		// Branch gone, no live session — fail.
		l.logger.Warn("merging stage branch not found, failing", "order", ms.order.ID, "stage", ms.stageIdx, "branch", checkBranch)
		if err := l.failMergingStage(ms.order.ID, ms.stageIdx, ms.isOnFailure, "merge branch "+checkBranch+" not found after crash"); err != nil {
			return err
		}
	}

	return nil
}

// failMergingStage transitions a stuck merging stage to failed via failStage.
func (l *Loop) failMergingStage(orderID string, stageIdx int, isOnFailure bool, reason string) error {
	orders, err := l.currentOrders()
	if err != nil {
		return err
	}
	orders, terminal, err := failStage(orders, orderID, reason)
	if err != nil {
		return err
	}
	if err := l.writeOrdersState(orders); err != nil {
		return err
	}
	if terminal {
		return l.markFailed(orderID, reason)
	}
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
	alive := map[string]struct{}{}
	for _, name := range loopruntime.ListTmuxSessions() {
		alive[name] = struct{}{}
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
		if _, ok := alive[loopruntime.TmuxSessionName(sessionID)]; !ok {
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

func (s *adoptedSession) Done() <-chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}
