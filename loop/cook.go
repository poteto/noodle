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
	"github.com/poteto/noodle/recover"
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
	reviewEnabled := l.config.Review.Enabled
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
		Runtime:      taskType.Runtime,
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
			return err
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
		if err := l.deps.Worktree.Merge(cook.worktreeName); err != nil {
			return l.retryCook(ctx, cook, "merge failed: "+err.Error())
		}
		if _, err := l.deps.Adapter.Run(ctx, "backlog", "done", adapter.RunOptions{Args: []string{cook.queueItem.ID}}); err != nil {
			if !isMissingAdapter(err) {
				return err
			}
		}
		if err := l.skipQueueItem(cook.queueItem.ID); err != nil {
			return err
		}
		return nil
	}
	return l.retryCook(ctx, cook, "cook exited with status "+status)
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
			return err
		}
		l.dropAdoptedTarget(targetID, sessionID)
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
	reviewEnabled := l.config.Review.Enabled
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
		if err := cook.session.Kill(); err != nil {
			return err
		}
		delete(l.activeByID, cook.session.ID())
		delete(l.activeByTarget, cook.queueItem.ID)
		return l.spawnCook(context.Background(), cook.queueItem, cook.attempt, strings.TrimSpace(prompt))
	}
	return errors.New("session not found")
}
