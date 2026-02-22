package loop

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/poteto/noodle/adapter"
	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/debate"
	"github.com/poteto/noodle/mise"
	"github.com/poteto/noodle/monitor"
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
		projectDir:      projectDir,
		runtimeDir:      runtimeDir,
		config:          cfg,
		deps:            deps,
		state:           StateRunning,
		activeByTarget:  map[string]*activeCook{},
		activeByID:      map[string]*activeCook{},
		adoptedTargets:  map[string]string{},
		adoptedSessions: []string{},
		failedTargets:   map[string]string{},
		processedIDs:    map[string]struct{}{},
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
		return err
	}
	if err := l.collectCompleted(ctx); err != nil {
		return err
	}
	if _, err := l.deps.Monitor.RunOnce(ctx); err != nil {
		return err
	}
	l.refreshAdoptedTargets()
	brief, _, err := l.deps.Mise.Build(ctx)
	if err != nil {
		return err
	}
	if l.state != StateRunning {
		return nil
	}

	queue, err := readQueue(l.deps.QueueFile)
	if err != nil {
		return err
	}
	if len(queue.Items) == 0 {
		queue = queueFromBacklog(brief.Backlog, l.config)
		if len(queue.Items) > 0 {
			if err := writeQueueAtomic(l.deps.QueueFile, queue); err != nil {
				return err
			}
		}
	}

	limit := l.config.Concurrency.MaxCooks
	if limit <= 0 {
		limit = 1
	}
	for _, item := range queue.Items {
		if len(l.activeByID)+len(l.adoptedTargets) >= limit {
			break
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
			return err
		}
	}
	return nil
}

func (l *Loop) spawnCook(ctx context.Context, item QueueItem, attempt int, resumePrompt string) error {
	baseName := sanitizeName(item.ID)
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

	if err := l.deps.Worktree.Create(name); err != nil {
		return fmt.Errorf("create worktree %s: %w", name, err)
	}

	prompt := buildCookPrompt(item, resumePrompt)
	req := spawner.SpawnRequest{
		Name:         name,
		Prompt:       prompt,
		Provider:     nonEmpty(item.Provider, l.config.Routing.Defaults.Provider),
		Model:        nonEmpty(item.Model, l.config.Routing.Defaults.Model),
		Skill:        item.Skill,
		WorktreePath: filepath.Join(l.projectDir, ".worktrees", name),
	}
	session, err := l.deps.Spawner.Spawn(ctx, req)
	if err != nil {
		_ = l.deps.Worktree.Cleanup(name, true)
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
	return nil
}

func (l *Loop) handleCompletion(ctx context.Context, cook *activeCook) error {
	status := strings.ToLower(strings.TrimSpace(cook.session.Status()))
	success := status == "completed" || status == "exited"
	if success && cook.reviewEnabled {
		accepted, feedback := l.runTaster(ctx, cook)
		if !accepted {
			return l.retryCook(ctx, cook, feedback)
		}
	}
	if success {
		if err := l.deps.Worktree.Merge(cook.worktreeName); err != nil {
			return l.retryCook(ctx, cook, "merge failed: "+err.Error())
		}
		if _, err := l.deps.Adapter.Run(ctx, "backlog", "done", adapter.RunOptions{Args: []string{cook.queueItem.ID}}); err != nil {
			if !isMissingAdapter(err) {
				return err
			}
		}
		return nil
	}
	return l.retryCook(ctx, cook, "cook exited with status "+status)
}

func (l *Loop) retryCook(ctx context.Context, cook *activeCook, reason string) error {
	nextAttempt := cook.attempt + 1
	if nextAttempt > l.config.Recovery.MaxRetries {
		if err := l.markFailed(cook.queueItem.ID, reason); err != nil {
			return err
		}
		_ = l.skipQueueItem(cook.queueItem.ID)
		_ = l.deps.Worktree.Cleanup(cook.worktreeName, true)
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
	_ = l.deps.Worktree.Cleanup(cook.worktreeName, true)
	return l.spawnCook(ctx, cook.queueItem, nextAttempt, resume.Summary)
}

func (l *Loop) runTaster(ctx context.Context, cook *activeCook) (bool, string) {
	reviewReq := spawner.SpawnRequest{
		Name:         cook.worktreeName + "-taster",
		Prompt:       "Review completed cook work for item " + cook.queueItem.ID,
		Provider:     l.config.Routing.Defaults.Provider,
		Model:        l.config.Routing.Defaults.Model,
		Skill:        "taster",
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
	if status == "completed" || status == "exited" {
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

func (l *Loop) reconcile(ctx context.Context) error {
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
		_ = exec.Command("tmux", "kill-session", "-t", name).Run()
	}

	if len(l.adoptedSessions) > 0 {
		tickets := monitor.NewEventTicketMaterializer(l.runtimeDir)
		_ = tickets.Materialize(ctx, l.adoptedSessions)
	}
	return nil
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
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return "cook"
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
	if name == "" {
		return "cook"
	}
	return name
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
	if strings.EqualFold(target, "sous-chef") {
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

func (l *Loop) reprioritizeForChefPrompt(prompt string) error {
	brief, _, err := l.deps.Mise.Build(context.Background())
	if err != nil {
		return err
	}
	queue := queueFromBacklog(brief.Backlog, l.config)
	prompt = strings.TrimSpace(prompt)
	for i := range queue.Items {
		if prompt != "" {
			queue.Items[i].Rationale = "Chef steer: " + prompt
		}
	}
	return writeQueueAtomic(l.deps.QueueFile, queue)
}

func (l *Loop) trimQueueForAdoptedTargets() error {
	if len(l.adoptedTargets) == 0 {
		return nil
	}
	queue, err := readQueue(l.deps.QueueFile)
	if err != nil {
		return err
	}
	filtered := make([]QueueItem, 0, len(queue.Items))
	for _, item := range queue.Items {
		if _, adopted := l.adoptedTargets[item.ID]; adopted {
			continue
		}
		filtered = append(filtered, item)
	}
	queue.Items = filtered
	return writeQueueAtomic(l.deps.QueueFile, queue)
}

var promptItemRegexp = regexp.MustCompile(`(?im)^work backlog item\s+([^\r\n]+)$`)

func readSessionTarget(promptPath string) string {
	data, err := os.ReadFile(promptPath)
	if err != nil {
		return ""
	}
	matches := promptItemRegexp.FindStringSubmatch(string(data))
	if len(matches) != 2 {
		return ""
	}
	return strings.TrimSpace(matches[1])
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
