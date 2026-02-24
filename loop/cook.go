package loop

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/poteto/noodle/adapter"
	"github.com/poteto/noodle/dispatcher"
	"github.com/poteto/noodle/event"
	"github.com/poteto/noodle/recover"
	"github.com/poteto/noodle/worktree"
)

func (l *Loop) spawnCook(ctx context.Context, item QueueItem, attempt int, resumePrompt string) error {
	if isPrioritizeItem(item) {
		return l.spawnPrioritize(ctx, item, attempt, resumePrompt)
	}

	baseName := cookBaseName(item)
	name := baseName
	if attempt > 0 {
		var err error
		name, err = recover.NextRecoveryName(baseName, attempt, l.config.Recovery.RetrySuffixPattern)
		if err != nil {
			return err
		}
	}
	reviewEnabled := l.config.ReviewEnabled()
	if item.Review != nil {
		reviewEnabled = *item.Review
	}

	created, err := l.ensureWorktree(name)
	if err != nil {
		return fmt.Errorf("create worktree %s: %w", name, err)
	}

	worktreePath := l.worktreePath(name)
	prompt := buildCookPrompt(item, resumePrompt)

	taskType, _ := l.registry.ByKey(item.TaskKey)
	req := dispatcher.DispatchRequest{
		Name:         name,
		Prompt:       prompt,
		Provider:     nonEmpty(item.Provider, l.config.Routing.Defaults.Provider),
		Model:        nonEmpty(item.Model, l.config.Routing.Defaults.Model),
		Skill:        item.Skill,
		WorktreePath: worktreePath,
		TaskKey:      taskType.Key,
		Runtime:      nonEmpty(item.Runtime, "tmux"),
	}
	if taskType.Key == "execute" {
		if adapter, exists := l.config.Adapters["backlog"]; exists {
			req.DomainSkill = adapter.Skill
		}
	}
	session, err := l.deps.Dispatcher.Dispatch(ctx, req)
	if err != nil {
		if created {
			_ = l.deps.Worktree.Cleanup(name, true)
		}
		return err
	}
	cook := &activeCook{
		queueItem:     item,
		session:       session,
		worktreeName:  name,
		worktreePath:  req.WorktreePath,
		attempt:       attempt,
		reviewEnabled: reviewEnabled,
	}
	l.activeByTarget[item.ID] = cook
	l.activeByID[session.ID()] = cook
	return nil
}

func (l *Loop) collectCompleted(ctx context.Context) error {
	completed := make([]*activeCook, 0)
	for _, cook := range l.activeByID {
		select {
		case <-cook.session.Done():
			completed = append(completed, cook)
		default:
		}
	}
	for _, cook := range completed {
		delete(l.activeByID, cook.session.ID())
		delete(l.activeByTarget, cook.queueItem.ID)
		if err := l.handleCompletion(ctx, cook); err != nil {
			if conflictErr := l.handleMergeConflict(cook, err); conflictErr != nil {
				return conflictErr
			}
		}
	}
	return l.collectAdoptedCompletions(ctx)
}

func (l *Loop) handleCompletion(ctx context.Context, cook *activeCook) error {
	status := strings.ToLower(strings.TrimSpace(cook.session.Status()))
	success := status == "completed"
	if success && cook.reviewEnabled {
		accepted, feedback := l.runQuality(ctx, cook)
		if !accepted {
			return l.retryCook(ctx, cook, feedback)
		}
	}
	if success {
		if isPrioritizeItem(cook.queueItem) {
			return l.skipQueueItem(cook.queueItem.ID)
		}
		// In approve mode, park the cook for human review instead of auto-merging.
		if l.config.PendingApproval() {
			l.pendingReview[cook.queueItem.ID] = &pendingReviewCook{
				queueItem:    cook.queueItem,
				worktreeName: cook.worktreeName,
				worktreePath: cook.worktreePath,
				sessionID:    cook.session.ID(),
			}
			return nil
		}
		return l.mergeCook(ctx, cook.queueItem, cook.worktreeName, cook.session.ID())
	}
	return l.retryCook(ctx, cook, "cook exited with status "+status)
}

func (l *Loop) mergeCook(ctx context.Context, item QueueItem, worktreeName string, sessionID string) error {
	syncResult, hasSyncResult, err := l.readSessionSyncResult(sessionID)
	if err != nil {
		return err
	}

	if hasSyncResult && syncResult.Type == dispatcher.SyncResultTypeBranch && strings.TrimSpace(syncResult.Branch) != "" {
		if err := l.deps.Worktree.MergeRemoteBranch(syncResult.Branch); err != nil {
			return fmt.Errorf("merge remote branch %s: %w", syncResult.Branch, err)
		}
	} else {
		if err := l.deps.Worktree.Merge(worktreeName); err != nil {
			return fmt.Errorf("merge %s: %w", worktreeName, err)
		}
	}
	if _, err := l.deps.Adapter.Run(ctx, "backlog", "done", adapter.RunOptions{Args: []string{item.ID}}); err != nil {
		if !isMissingAdapter(err) {
			return err
		}
	}
	return l.skipQueueItem(item.ID)
}

func (l *Loop) readSessionSyncResult(sessionID string) (dispatcher.SyncResult, bool, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return dispatcher.SyncResult{}, false, nil
	}
	path := filepath.Join(l.runtimeDir, "sessions", sessionID, "spawn.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return dispatcher.SyncResult{}, false, nil
		}
		return dispatcher.SyncResult{}, false, fmt.Errorf("read spawn metadata: %w", err)
	}
	var payload struct {
		Sync dispatcher.SyncResult `json:"sync"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return dispatcher.SyncResult{}, false, fmt.Errorf("parse spawn metadata: %w", err)
	}
	if strings.TrimSpace(payload.Sync.Type) == "" && strings.TrimSpace(payload.Sync.Branch) == "" {
		return dispatcher.SyncResult{}, false, nil
	}
	payload.Sync.Type = strings.ToLower(strings.TrimSpace(payload.Sync.Type))
	payload.Sync.Branch = strings.TrimSpace(payload.Sync.Branch)
	return payload.Sync, true, nil
}

func (l *Loop) collectAdoptedCompletions(ctx context.Context) error {
	for targetID, sessionID := range l.adoptedTargets {
		status, ok, err := l.readSessionStatus(sessionID)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		switch status {
		case "running", "stuck", "spawning":
			continue
		}
		cook, processable, err := l.buildAdoptedCook(targetID, sessionID, status)
		if err != nil {
			return err
		}
		if !processable {
			l.dropAdoptedTarget(targetID, sessionID)
			continue
		}
		if err := l.handleCompletion(ctx, cook); err != nil {
			if conflictErr := l.handleMergeConflict(cook, err); conflictErr != nil {
				return conflictErr
			}
		}
		l.dropAdoptedTarget(targetID, sessionID)
	}
	return nil
}

func (l *Loop) handleMergeConflict(cook *activeCook, err error) error {
	var conflictErr *worktree.MergeConflictError
	if !errors.As(err, &conflictErr) {
		return err
	}
	if isPrioritizeItem(cook.queueItem) {
		return err
	}
	if markErr := l.markFailed(cook.queueItem.ID, conflictErr.Error()); markErr != nil {
		return markErr
	}
	if skipErr := l.skipQueueItem(cook.queueItem.ID); skipErr != nil {
		return skipErr
	}
	return nil
}

func (l *Loop) worktreePath(name string) string {
	return filepath.Join(l.projectDir, ".worktrees", name)
}

func (l *Loop) ensureWorktree(name string) (bool, error) {
	path := l.worktreePath(name)
	info, err := os.Stat(path)
	if err == nil {
		if !info.IsDir() {
			return false, fmt.Errorf("worktree path %s is not a directory", path)
		}
		return false, nil
	}
	if !os.IsNotExist(err) {
		return false, fmt.Errorf("stat worktree path %s: %w", path, err)
	}

	if err := l.deps.Worktree.Create(name); err != nil {
		if isWorktreeAlreadyExistsError(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (l *Loop) readSessionStatus(sessionID string) (string, bool, error) {
	metaPath := filepath.Join(l.runtimeDir, "sessions", sessionID, "meta.json")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, err
	}
	var payload struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return "", false, err
	}
	return strings.ToLower(strings.TrimSpace(payload.Status)), true, nil
}

func (l *Loop) buildAdoptedCook(targetID string, sessionID string, status string) (*activeCook, bool, error) {
	item, found, err := l.lookupQueueItem(targetID)
	if err != nil {
		return nil, false, err
	}
	if !found {
		return nil, false, nil
	}
	reviewEnabled := l.config.ReviewEnabled()
	if isPrioritizeItem(item) {
		reviewEnabled = false
	} else if item.Review != nil {
		reviewEnabled = *item.Review
	}
	worktreeName, worktreePath := l.readAdoptedWorktree(sessionID, item)
	return &activeCook{
		queueItem: item,
		session: &adoptedSession{
			id:     sessionID,
			status: status,
		},
		worktreeName:  worktreeName,
		worktreePath:  worktreePath,
		attempt:       recover.RecoveryChainLength(worktreeName),
		reviewEnabled: reviewEnabled,
	}, true, nil
}

func (l *Loop) lookupQueueItem(targetID string) (QueueItem, bool, error) {
	queue, err := readQueue(l.deps.QueueFile)
	if err != nil {
		return QueueItem{}, false, err
	}
	for _, item := range queue.Items {
		if item.ID == targetID {
			return item, true, nil
		}
	}
	return QueueItem{}, false, nil
}

func (l *Loop) readAdoptedWorktree(sessionID string, item QueueItem) (string, string) {
	path := filepath.Join(l.runtimeDir, "sessions", sessionID, "spawn.json")
	worktreePath := ""
	data, err := os.ReadFile(path)
	if err == nil {
		var payload struct {
			WorktreePath string `json:"worktree_path"`
		}
		if jsonErr := json.Unmarshal(data, &payload); jsonErr == nil {
			worktreePath = strings.TrimSpace(payload.WorktreePath)
		}
	}
	if worktreePath == "" {
		name := cookBaseName(item)
		return name, filepath.Join(l.projectDir, ".worktrees", name)
	}
	name := filepath.Base(worktreePath)
	if strings.TrimSpace(name) == "" || name == "." || name == string(filepath.Separator) {
		name = cookBaseName(item)
		worktreePath = filepath.Join(l.projectDir, ".worktrees", name)
	}
	return name, worktreePath
}

func (l *Loop) dropAdoptedTarget(targetID string, sessionID string) {
	delete(l.adoptedTargets, targetID)
	filtered := l.adoptedSessions[:0]
	for _, id := range l.adoptedSessions {
		if id == sessionID {
			continue
		}
		filtered = append(filtered, id)
	}
	l.adoptedSessions = filtered
}

func (l *Loop) retryCook(ctx context.Context, cook *activeCook, reason string) error {
	nextAttempt := cook.attempt + 1
	info, err := recover.CollectRecoveryInfo(ctx, l.runtimeDir, cook.session.ID())
	if err != nil {
		info = recover.RecoveryInfo{SessionID: cook.session.ID(), ExitReason: reason}
	}
	resolvedReason := retryFailureReason(reason, info)
	if nextAttempt > l.config.Recovery.MaxRetries {
		if isPrioritizeItem(cook.queueItem) {
			return fmt.Errorf("prioritize failed after retries: %s", resolvedReason)
		}
		if err := l.markFailed(cook.queueItem.ID, resolvedReason); err != nil {
			return err
		}
		_ = l.skipQueueItem(cook.queueItem.ID)
		if strings.TrimSpace(cook.worktreeName) != "" {
			_ = l.deps.Worktree.Cleanup(cook.worktreeName, true)
		}
		return nil
	}
	if strings.TrimSpace(info.ExitReason) == "" {
		info.ExitReason = resolvedReason
	}
	resume := recover.BuildResumeContext(info, nextAttempt, l.config.Recovery.MaxRetries)
	if strings.TrimSpace(cook.worktreeName) != "" {
		_ = l.deps.Worktree.Cleanup(cook.worktreeName, true)
	}
	return l.spawnCook(ctx, cook.queueItem, nextAttempt, resume.Summary)
}

func retryFailureReason(base string, info recover.RecoveryInfo) string {
	base = strings.TrimSpace(base)
	if base == "" {
		base = "cook failed"
	}

	exitReason := strings.TrimSpace(info.ExitReason)
	if exitReason == "" {
		return base
	}
	if strings.EqualFold(exitReason, "session exited without explicit reason") {
		return base
	}

	if strings.HasPrefix(strings.ToLower(base), "cook exited with status") {
		return exitReason
	}
	return base
}

func (l *Loop) skipQueueItem(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("skip requires item")
	}
	queue, err := readQueue(l.deps.QueueFile)
	if err != nil {
		return err
	}
	filtered := make([]QueueItem, 0, len(queue.Items))
	for _, item := range queue.Items {
		if item.ID == id {
			continue
		}
		filtered = append(filtered, item)
	}
	queue.Items = filtered
	return writeQueueAtomic(l.deps.QueueFile, queue)
}

func (l *Loop) killCook(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("kill requires name")
	}
	for _, cook := range l.activeByID {
		if cook.worktreeName == name || cook.session.ID() == name {
			return cook.session.Kill()
		}
	}
	return fmt.Errorf("session not found")
}

func (l *Loop) steer(target string, prompt string) error {
	target = strings.TrimSpace(target)
	if target == "" {
		return fmt.Errorf("steer requires target")
	}
	if strings.EqualFold(target, PrioritizeTaskKey()) {
		return l.reprioritizeForChefPrompt(prompt)
	}
	for _, cook := range l.activeByID {
		if cook.worktreeName != target && cook.session.ID() != target {
			continue
		}
		// Build resume context before killing — the new session needs
		// to know what the old one was doing.
		resumeCtx := buildSteerResumeContext(l.runtimeDir, cook.session.ID())
		steerPrompt := strings.TrimSpace(prompt)
		if resumeCtx != "" {
			steerPrompt = "Resume context: " + resumeCtx + "\n\nChef steering: " + steerPrompt
		}

		if err := cook.session.Kill(); err != nil {
			return err
		}
		delete(l.activeByID, cook.session.ID())
		delete(l.activeByTarget, cook.queueItem.ID)
		return l.spawnCook(context.Background(), cook.queueItem, cook.attempt, steerPrompt)
	}
	return errors.New("session not found")
}

// buildSteerResumeContext reads a session's event log and extracts a progress
// summary so the respawned session doesn't start from scratch.
func buildSteerResumeContext(runtimeDir string, sessionID string) string {
	reader := event.NewEventReader(runtimeDir)
	events, err := reader.ReadSession(sessionID, event.EventFilter{})
	if err != nil || len(events) == 0 {
		return ""
	}

	files := make(map[string]struct{})
	var lastActions []string
	var ticketProgress []string

	for _, ev := range events {
		switch ev.Type {
		case event.EventAction:
			var action struct {
				Tool    string `json:"tool"`
				Path    string `json:"path"`
				Summary string `json:"summary"`
			}
			_ = json.Unmarshal(ev.Payload, &action)
			tool := strings.ToLower(strings.TrimSpace(action.Tool))
			if path := strings.TrimSpace(action.Path); path != "" {
				switch tool {
				case "read", "edit", "write":
					files[path] = struct{}{}
				}
			}
			summary := strings.TrimSpace(action.Summary)
			if summary == "" {
				summary = strings.TrimSpace(action.Tool)
			}
			if summary != "" {
				lastActions = append(lastActions, summary)
			}
		case event.EventTicketProgress, event.EventTicketDone:
			var payload struct {
				Summary string `json:"summary"`
				Outcome string `json:"outcome"`
			}
			_ = json.Unmarshal(ev.Payload, &payload)
			if s := strings.TrimSpace(payload.Summary); s != "" {
				ticketProgress = append(ticketProgress, s)
			} else if s := strings.TrimSpace(payload.Outcome); s != "" {
				ticketProgress = append(ticketProgress, s)
			}
		}
	}

	var parts []string
	if len(files) > 0 {
		fileList := make([]string, 0, len(files))
		for f := range files {
			fileList = append(fileList, f)
		}
		if len(fileList) > 10 {
			fileList = fileList[:10]
		}
		parts = append(parts, fmt.Sprintf("Files touched: %s", strings.Join(fileList, ", ")))
	}
	if len(ticketProgress) > 0 {
		if len(ticketProgress) > 3 {
			ticketProgress = ticketProgress[len(ticketProgress)-3:]
		}
		parts = append(parts, fmt.Sprintf("Progress: %s", strings.Join(ticketProgress, "; ")))
	}
	if len(lastActions) > 0 {
		tail := lastActions
		if len(tail) > 5 {
			tail = tail[len(tail)-5:]
		}
		parts = append(parts, fmt.Sprintf("Recent actions: %s", strings.Join(tail, " → ")))
	}

	return strings.Join(parts, ". ")
}
