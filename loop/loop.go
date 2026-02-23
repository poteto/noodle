package loop

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/poteto/noodle/adapter"
	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/debate"
	"github.com/poteto/noodle/mise"
	"github.com/poteto/noodle/parse"
	"github.com/poteto/noodle/recover"
	"github.com/poteto/noodle/spawner"
)

func New(projectDir, noodleBin string, cfg config.Config, deps Dependencies) *Loop {
	projectDir = strings.TrimSpace(projectDir)
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if deps.Spawner == nil || deps.Worktree == nil || deps.Adapter == nil || deps.Mise == nil || deps.Monitor == nil {
		defaults := defaultDependencies(projectDir, runtimeDir, noodleBin, cfg)
		if deps.Spawner == nil {
			deps.Spawner = defaults.Spawner
		}
		if deps.Worktree == nil {
			deps.Worktree = defaults.Worktree
		}
		if deps.Adapter == nil {
			deps.Adapter = defaults.Adapter
		}
		if deps.Mise == nil {
			deps.Mise = defaults.Mise
		}
		if deps.Monitor == nil {
			deps.Monitor = defaults.Monitor
		}
		if deps.Now == nil {
			deps.Now = defaults.Now
		}
		if deps.QueueFile == "" {
			deps.QueueFile = defaults.QueueFile
		}
	}
	if deps.Now == nil {
		deps.Now = time.Now
	}
	if deps.QueueFile == "" {
		deps.QueueFile = filepath.Join(runtimeDir, "queue.json")
	}

	return &Loop{
		projectDir:            projectDir,
		runtimeDir:            runtimeDir,
		config:                cfg,
		deps:                  deps,
		state:                 StateRunning,
		activeByTarget:        map[string]*activeCook{},
		activeByID:            map[string]*activeCook{},
		adoptedTargets:        map[string]string{},
		adoptedSessions:       []string{},
		failedTargets:         map[string]string{},
		processedIDs:          map[string]struct{}{},
		runtimeRepairAttempts: map[string]int{},
	}
}

func (l *Loop) Run(ctx context.Context) error {
	if strings.TrimSpace(l.projectDir) == "" {
		return fmt.Errorf("project directory is required")
	}
	if err := os.MkdirAll(l.runtimeDir, 0o755); err != nil {
		return fmt.Errorf("create runtime directory: %w", err)
	}
	if err := l.loadFailedTargets(); err != nil {
		return err
	}
	if err := l.reconcile(ctx); err != nil {
		return err
	}
	if err := l.hydrateProcessedCommands(); err != nil {
		return err
	}
	if err := l.Cycle(ctx); err != nil {
		return err
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create fsnotify watcher: %w", err)
	}
	defer watcher.Close()
	if err := watcher.Add(l.runtimeDir); err != nil {
		return fmt.Errorf("watch runtime directory: %w", err)
	}

	ticker := time.NewTicker(l.pollInterval())
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := l.Cycle(ctx); err != nil {
				return err
			}
			if l.state == StateDraining && len(l.activeByID) == 0 {
				return nil
			}
		case ev := <-watcher.Events:
			if strings.HasSuffix(ev.Name, "queue.json") || strings.HasSuffix(ev.Name, "control.ndjson") {
				if err := l.Cycle(ctx); err != nil {
					return err
				}
			}
		case err := <-watcher.Errors:
			if err != nil {
				return fmt.Errorf("watch runtime directory: %w", err)
			}
		}
	}
}

func (l *Loop) Cycle(ctx context.Context) error {
	if err := l.processControlCommands(); err != nil {
		return l.handleRuntimeIssue(ctx, "loop.control", err, nil)
	}
	if err := l.collectCompleted(ctx); err != nil {
		return l.handleRuntimeIssue(ctx, "loop.collect", err, nil)
	}
	if _, err := l.deps.Monitor.RunOnce(ctx); err != nil {
		return l.handleRuntimeIssue(ctx, "loop.monitor", err, nil)
	}
	if err := l.advanceRuntimeRepair(ctx); err != nil {
		return err
	}
	if l.runtimeRepairInFlight != nil {
		return nil
	}
	l.refreshAdoptedTargets()
	brief, warnings, err := l.deps.Mise.Build(ctx)
	if err != nil {
		return l.handleRuntimeIssue(ctx, "mise.build", err, warnings)
	}
	if l.state != StateRunning {
		return nil
	}

	queue, err := readQueue(l.deps.QueueFile)
	if err != nil {
		return err
	}
	if normalizedQueue, changed, err := normalizeAndValidateQueue(queue, brief.Backlog, l.config); err != nil {
		return l.handleRuntimeIssue(ctx, "loop.queue", err, nil)
	} else if changed {
		queue = normalizedQueue
		if err := writeQueueAtomic(l.deps.QueueFile, queue); err != nil {
			return err
		}
	}
	if shouldRecoverMissingSyncScripts(warnings, queue) &&
		len(l.activeByID) == 0 &&
		len(l.adoptedTargets) == 0 {
		return l.handleRuntimeIssue(ctx, "mise.sync", nil, warnings)
	}
	if len(queue.Items) == 0 &&
		len(l.activeByID) == 0 &&
		len(l.adoptedTargets) == 0 {
		queue = bootstrapPrioritizeQueue(l.config, "", l.deps.Now().UTC())
		if err := writeQueueAtomic(l.deps.QueueFile, queue); err != nil {
			return err
		}
	}
	if updatedQueue, changed := applyQueueRoutingDefaults(queue, l.config); changed {
		queue = updatedQueue
		if err := writeQueueAtomic(l.deps.QueueFile, queue); err != nil {
			return err
		}
	}

	limit := l.config.Concurrency.MaxCooks
	if limit <= 0 {
		limit = 1
	}
	for _, item := range queue.Items {
		if l.hasBlockingActive(queue.Items) {
			break
		}
		if len(l.activeByID)+len(l.adoptedTargets) >= limit {
			break
		}
		if isBlockingQueueItem(l.config, item) && len(l.activeByID)+len(l.adoptedTargets) > 0 {
			continue
		}
		if _, busy := l.activeByTarget[item.ID]; busy {
			continue
		}
		if _, failed := l.failedTargets[item.ID]; failed {
			continue
		}
		if _, adopted := l.adoptedTargets[item.ID]; adopted {
			continue
		}
		if hasActiveTicket(brief, item.ID) {
			continue
		}
		if err := l.spawnCook(ctx, item, 0, ""); err != nil {
			return l.handleRuntimeIssue(ctx, "loop.spawn", err, nil)
		}
	}
	return nil
}

func (l *Loop) hasBlockingActive(queueItems []QueueItem) bool {
	for _, cook := range l.activeByID {
		if isBlockingQueueItem(l.config, cook.queueItem) {
			return true
		}
	}
	for targetID := range l.adoptedTargets {
		if item, ok := findQueueItemByTarget(queueItems, targetID); ok {
			if isBlockingQueueItem(l.config, item) {
				return true
			}
		}
	}
	return false
}

func findQueueItemByTarget(items []QueueItem, targetID string) (QueueItem, bool) {
	targetID = strings.TrimSpace(targetID)
	if targetID == "" {
		return QueueItem{}, false
	}
	for _, item := range items {
		if strings.TrimSpace(item.ID) == targetID {
			return item, true
		}
	}
	return QueueItem{}, false
}

func shouldRecoverMissingSyncScripts(warnings []string, queue Queue) bool {
	if len(queue.Items) > 0 {
		return false
	}
	for _, warning := range warnings {
		warning = strings.ToLower(strings.TrimSpace(warning))
		if warning == "" {
			continue
		}
		if strings.Contains(warning, "sync script missing") {
			return true
		}
	}
	return false
}

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
	req := spawner.SpawnRequest{
		Name:         name,
		Prompt:       prompt,
		Provider:     nonEmpty(item.Provider, l.config.Routing.Defaults.Provider),
		Model:        nonEmpty(item.Model, l.config.Routing.Defaults.Model),
		Skill:        item.Skill,
		WorktreePath: worktreePath,
	}
	session, err := l.deps.Spawner.Spawn(ctx, req)
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
		accepted, feedback := l.runTaster(ctx, cook)
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
	if nextAttempt > l.config.Recovery.MaxRetries {
		if isPrioritizeItem(cook.queueItem) {
			return fmt.Errorf("prioritize failed after retries: %s", strings.TrimSpace(reason))
		}
		if err := l.markFailed(cook.queueItem.ID, reason); err != nil {
			return err
		}
		_ = l.skipQueueItem(cook.queueItem.ID)
		if strings.TrimSpace(cook.worktreeName) != "" {
			_ = l.deps.Worktree.Cleanup(cook.worktreeName, true)
		}
		return nil
	}
	info, err := recover.CollectRecoveryInfo(ctx, l.runtimeDir, cook.session.ID())
	if err != nil {
		info = recover.RecoveryInfo{SessionID: cook.session.ID(), ExitReason: reason}
	}
	if strings.TrimSpace(info.ExitReason) == "" {
		info.ExitReason = reason
	}
	resume := recover.BuildResumeContext(info, nextAttempt, l.config.Recovery.MaxRetries)
	if strings.TrimSpace(cook.worktreeName) != "" {
		_ = l.deps.Worktree.Cleanup(cook.worktreeName, true)
	}
	return l.spawnCook(ctx, cook.queueItem, nextAttempt, resume.Summary)
}

func (l *Loop) runTaster(ctx context.Context, cook *activeCook) (bool, string) {
	reviewReq := spawner.SpawnRequest{
		Name:         cook.worktreeName + "-taster",
		Prompt:       "Review completed cook work for item " + cook.queueItem.ID,
		Provider:     l.config.Routing.Defaults.Provider,
		Model:        l.config.Routing.Defaults.Model,
		Skill:        tasterTaskSkill(l.config),
		WorktreePath: cook.worktreePath,
	}
	session, err := l.deps.Spawner.Spawn(ctx, reviewReq)
	if err != nil {
		return false, "unable to spawn taster: " + err.Error()
	}
	select {
	case <-ctx.Done():
		return false, ctx.Err().Error()
	case <-session.Done():
	}

	verdict, found, err := readTasterVerdict(filepath.Join(l.runtimeDir, "sessions", session.ID(), "canonical.ndjson"))
	if err == nil && found {
		_ = l.writeDebateVerdict(cook, verdict.Accept, verdict.Feedback)
		if verdict.Accept {
			return true, verdict.Feedback
		}
		return false, verdict.Feedback
	}

	status := strings.ToLower(strings.TrimSpace(session.Status()))
	if status == "completed" {
		_ = l.writeDebateVerdict(cook, true, "")
		return true, ""
	}
	feedback := "taster rejected with status " + status
	_ = l.writeDebateVerdict(cook, false, feedback)
	return false, feedback
}

func (l *Loop) pollInterval() time.Duration {
	interval := strings.TrimSpace(l.config.Monitor.PollInterval)
	if interval == "" {
		return 5 * time.Second
	}
	duration, err := time.ParseDuration(interval)
	if err != nil || duration <= 0 {
		return 5 * time.Second
	}
	return duration
}

func hasActiveTicket(brief mise.Brief, target string) bool {
	target = strings.TrimSpace(target)
	if target == "" {
		return false
	}
	for _, ticket := range brief.Tickets {
		if ticket.Target != target {
			continue
		}
		switch ticket.Status {
		case "active", "blocked":
			return true
		}
	}
	return false
}

func isMissingAdapter(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "not configured") || strings.Contains(text, "no such file")
}

func isWorktreeAlreadyExistsError(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "worktree") && strings.Contains(text, "already exists at")
}

func buildCookPrompt(item QueueItem, resumePrompt string) string {
	parts := []string{fmt.Sprintf("Work backlog item %s", item.ID)}
	if strings.TrimSpace(item.Rationale) != "" {
		parts = append(parts, "Context: "+strings.TrimSpace(item.Rationale))
	}
	if strings.TrimSpace(resumePrompt) != "" {
		parts = append(parts, resumePrompt)
	}
	return strings.Join(parts, "\n\n")
}

func nonEmpty(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return strings.TrimSpace(fallback)
	}
	return value
}

func sanitizeName(value string) string {
	name := sanitizeToken(value)
	if name == "" {
		return "cook"
	}
	return name
}

func sanitizeToken(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return ""
	}
	builder := strings.Builder{}
	lastHyphen := false
	for _, r := range value {
		isAlphaNum := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if isAlphaNum {
			builder.WriteRune(r)
			lastHyphen = false
			continue
		}
		if !lastHyphen {
			builder.WriteByte('-')
			lastHyphen = true
		}
	}
	name := strings.Trim(builder.String(), "-")
	return name
}

func cookBaseName(item QueueItem) string {
	idToken := sanitizeToken(item.ID)
	if idToken == "" {
		idToken = "cook"
	}

	titleToken := sanitizeToken(item.Title)
	if titleToken == "" {
		return idToken
	}

	titleToken = truncateToken(titleToken, 32)
	if titleToken == "" {
		return idToken
	}

	const maxNameLen = 64
	base := idToken + "-" + titleToken
	if len(base) <= maxNameLen {
		return base
	}

	maxTitleLen := maxNameLen - len(idToken) - 1
	if maxTitleLen <= 0 {
		return idToken
	}
	titleToken = truncateToken(titleToken, maxTitleLen)
	if titleToken == "" {
		return idToken
	}
	return idToken + "-" + titleToken
}

func truncateToken(token string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	token = strings.Trim(token, "-")
	if token == "" {
		return ""
	}
	if len(token) <= maxLen {
		return token
	}
	token = token[:maxLen]
	return strings.Trim(token, "-")
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

type tasterVerdict struct {
	Accept   bool   `json:"accept"`
	Feedback string `json:"feedback,omitempty"`
}

var verdictJSONRegexp = regexp.MustCompile(`\{[^{}]*"accept"\s*:\s*(true|false)[^{}]*\}`)

func readTasterVerdict(path string) (tasterVerdict, bool, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return tasterVerdict{}, false, nil
		}
		return tasterVerdict{}, false, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var event parse.CanonicalEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}
		verdict, found := verdictFromText(event.Message)
		if found {
			return verdict, true, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return tasterVerdict{}, false, err
	}
	return tasterVerdict{}, false, nil
}

func verdictFromText(text string) (tasterVerdict, bool) {
	match := verdictJSONRegexp.FindString(text)
	if match == "" {
		return tasterVerdict{}, false
	}
	var verdict tasterVerdict
	if err := json.Unmarshal([]byte(match), &verdict); err != nil {
		return tasterVerdict{}, false
	}
	return verdict, true
}

func (l *Loop) writeDebateVerdict(cook *activeCook, accept bool, feedback string) error {
	store, err := debate.NewStore(filepath.Join(l.projectDir, "brain", "debates"))
	if err != nil {
		return err
	}
	d, err := store.Create("cook-"+cook.queueItem.ID, 6)
	if err != nil {
		return err
	}
	if _, err := store.AddRound(d, "reviewer", "Taster review for item "+cook.queueItem.ID); err != nil {
		return err
	}
	return store.WriteVerdict(d, debate.Verdict{Consensus: accept, Summary: strings.TrimSpace(feedback)})
}
