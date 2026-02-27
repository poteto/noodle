package loop

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/poteto/noodle/dispatcher"
	"github.com/poteto/noodle/monitor"
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
	entries, err := os.ReadDir(filepath.Join(l.runtimeDir, "sessions"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	alive := map[string]struct{}{}
	for _, name := range listTmuxSessions() {
		alive[name] = struct{}{}
	}
	knownTmux := map[string]struct{}{}
	l.adoptedTargets = map[string]string{}
	l.adoptedSessions = l.adoptedSessions[:0]

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		sessionID := entry.Name()
		sessionName := tmuxSessionName(sessionID)
		knownTmux[sessionName] = struct{}{}

		metaPath := filepath.Join(l.runtimeDir, "sessions", sessionID, "meta.json")
		data, err := os.ReadFile(metaPath)
		if err != nil {
			continue
		}
		if strings.Contains(string(data), `"status":"running"`) {
			if _, ok := alive[sessionName]; !ok {
				updated := strings.Replace(string(data), `"status":"running"`, `"status":"exited"`, 1)
				_ = os.WriteFile(metaPath, []byte(updated), 0o644)
				continue
			}
			target := readSessionTarget(filepath.Join(l.runtimeDir, "sessions", sessionID, "prompt.txt"))
			if target != "" {
				l.adoptedTargets[target] = sessionID
			}
			l.adoptedSessions = append(l.adoptedSessions, sessionID)
		}
	}

	for name := range alive {
		if !strings.HasPrefix(name, "noodle-") {
			continue
		}
		if _, ok := knownTmux[name]; ok {
			continue
		}
		_ = killTmuxSession(name)
	}

	if len(l.adoptedSessions) > 0 {
		tickets := monitor.NewEventTicketMaterializer(l.runtimeDir)
		_ = tickets.Materialize(ctx, l.adoptedSessions)
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
		if _, adopted := l.adoptedTargets[ms.order.ID]; adopted {
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
				orderID:     ms.order.ID,
				stageIndex:  ms.stageIdx,
				stage:       ms.stage,
				isOnFailure: ms.isOnFailure,
				orderStatus: ms.order.Status,
				plan:        ms.order.Plan,
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
				orderID:      ms.order.ID,
				stageIndex:   ms.stageIdx,
				stage:        ms.stage,
				isOnFailure:  ms.isOnFailure,
				orderStatus:  ms.order.Status,
				plan:         ms.order.Plan,
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

func listTmuxSessions() []string {
	out, err := exec.Command("tmux", "list-sessions", "-F", "#{session_name}").Output()
	if err != nil {
		return nil
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	outList := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		outList = append(outList, line)
	}
	return outList
}

// New format: [order:ID] Work on plan: ... or [order:ID] Work backlog item ID
var promptOrderRegexp = regexp.MustCompile(`(?im)^\[order:([^\]]+)\]`)

// Old format: Work backlog item <id>
var promptItemRegexp = regexp.MustCompile(`(?im)^work backlog item\s+([^\r\n]+)$`)
var schedulePromptRegexp = regexp.MustCompile(`(?im)^\s*use skill\([^)]+\)\s+to refresh .+from \.noodle/mise\.json\.`)

func readSessionTarget(promptPath string) string {
	data, err := os.ReadFile(promptPath)
	if err != nil {
		return ""
	}

	// Try new format first: [order:ID]
	orderMatches := promptOrderRegexp.FindStringSubmatch(string(data))
	if len(orderMatches) == 2 {
		return strings.TrimSpace(orderMatches[1])
	}

	// Old format: Work backlog item <id>
	matches := promptItemRegexp.FindStringSubmatch(string(data))
	if len(matches) == 2 {
		return strings.TrimSpace(matches[1])
	}

	if schedulePromptRegexp.Match(data) {
		return scheduleOrderID
	}
	return ""
}

func (l *Loop) refreshAdoptedTargets() {
	if len(l.adoptedTargets) == 0 {
		return
	}
	alive := map[string]struct{}{}
	for _, name := range listTmuxSessions() {
		alive[name] = struct{}{}
	}
	nextTargets := make(map[string]string, len(l.adoptedTargets))
	nextSessions := make([]string, 0, len(l.adoptedTargets))
	for target, sessionID := range l.adoptedTargets {
		metaPath := filepath.Join(l.runtimeDir, "sessions", sessionID, "meta.json")
		data, err := os.ReadFile(metaPath)
		if err != nil {
			continue
		}
		if !strings.Contains(string(data), `"status":"running"`) {
			continue
		}
		if _, ok := alive[tmuxSessionName(sessionID)]; !ok {
			continue
		}
		nextTargets[target] = sessionID
		nextSessions = append(nextSessions, sessionID)
	}
	l.adoptedTargets = nextTargets
	l.adoptedSessions = nextSessions
}

func tmuxSessionName(sessionID string) string {
	return "noodle-" + sanitizeSessionToken(sessionID, "cook")
}

type adoptedSession struct {
	id     string
	status string
}

func (s *adoptedSession) ID() string {
	return s.id
}

func (s *adoptedSession) Status() string {
	return s.status
}

func (s *adoptedSession) Events() <-chan dispatcher.SessionEvent {
	ch := make(chan dispatcher.SessionEvent)
	close(ch)
	return ch
}

func (s *adoptedSession) Done() <-chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}

func (s *adoptedSession) TotalCost() float64 {
	return 0
}

func (s *adoptedSession) Kill() error {
	return nil
}

func (s *adoptedSession) VerdictPath() string {
	return ""
}

// reconcileMergingStages resets stages stuck in "merging" status from a
// previous crash back to their prior state so they can be re-dispatched.
func (l *Loop) reconcileMergingStages() error {
	return nil
}

func killTmuxSession(name string) error {
	return exec.Command("tmux", "kill-session", "-t", name).Run()
}

func sanitizeSessionToken(value string, fallback string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		value = fallback
	}
	var out strings.Builder
	lastHyphen := false
	for _, r := range value {
		isAlphaNum := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if isAlphaNum {
			out.WriteRune(r)
			lastHyphen = false
			continue
		}
		if !lastHyphen {
			out.WriteByte('-')
			lastHyphen = true
		}
	}
	result := strings.Trim(out.String(), "-")
	if result == "" {
		result = fallback
	}
	if len(result) > 48 {
		result = strings.Trim(result[:48], "-")
	}
	if result == "" {
		return fallback
	}
	return result
}
