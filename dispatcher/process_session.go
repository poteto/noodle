package dispatcher

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/poteto/noodle/event"
	"github.com/poteto/noodle/parse"
	"github.com/poteto/noodle/stamp"
)

// processSession implements Session backed by a direct child process.
// It runs the stamp processor in-process (like spritesSession) rather than
// as a shell pipeline stage.
type processSession struct {
	id            string
	process       *ProcessHandle
	eventWriter   *event.EventWriter
	canonicalPath string
	stampedPath   string

	mu       sync.Mutex
	status   string
	costUSD  float64
	done     chan struct{}
	doneOnce sync.Once
	wg       sync.WaitGroup
	eventsMu sync.Once
	events   chan SessionEvent
	dropped  atomic.Uint64

	prompt       string
	promptLogged bool
	warnings     []string
	controller   *claudeController // nil for non-steerable sessions
	sink         SessionEventSink
}

type processSessionConfig struct {
	id            string
	process       *ProcessHandle
	eventWriter   *event.EventWriter
	canonicalPath string
	stampedPath   string
	prompt        string
	warnings      []string
	controller    *claudeController // nil for non-steerable sessions
	sink          SessionEventSink
}

func newProcessSession(cfg processSessionConfig) *processSession {
	return &processSession{
		id:            cfg.id,
		process:       cfg.process,
		eventWriter:   cfg.eventWriter,
		canonicalPath: cfg.canonicalPath,
		stampedPath:   cfg.stampedPath,
		prompt:        cfg.prompt,
		warnings:      append([]string(nil), cfg.warnings...),
		controller:    cfg.controller,
		sink:          cfg.sink,
		status:        "running",
		done:          make(chan struct{}),
		events:        make(chan SessionEvent, 32),
	}
}

func (s *processSession) start(ctx context.Context) {
	s.writeHeartbeat(nowUTC())
	s.wg.Add(2)

	go func() {
		defer s.wg.Done()
		s.processStream(ctx)
	}()

	go func() {
		defer s.wg.Done()
		s.waitForExit(ctx)
	}()

	go s.closeEventsWhenDone()

	for _, warning := range s.warnings {
		s.publish(SessionEvent{
			Type:      "warning",
			Message:   warning,
			Timestamp: nowUTC(),
		})
	}
}

func (s *processSession) processStream(ctx context.Context) {
	stampedFile, err := os.Create(s.stampedPath)
	if err != nil {
		s.publish(SessionEvent{Type: "warning", Message: "create stamped file: " + err.Error(), Timestamp: nowUTC()})
		return
	}
	defer stampedFile.Close()

	canonicalFile, err := os.Create(s.canonicalPath)
	if err != nil {
		s.publish(SessionEvent{Type: "warning", Message: "create canonical file: " + err.Error(), Timestamp: nowUTC()})
		return
	}
	defer canonicalFile.Close()

	processor := stamp.NewProcessor()
	interceptor := &processEventInterceptor{session: s}

	if err := processor.Process(ctx, s.process.Stdout(), stampedFile, io.MultiWriter(canonicalFile, interceptor)); err != nil {
		if ctx.Err() == nil {
			s.publish(SessionEvent{Type: "warning", Message: "process stream: " + err.Error(), Timestamp: nowUTC()})
		}
	}
}

func (s *processSession) waitForExit(ctx context.Context) {
	select {
	case <-s.process.Done():
	case <-ctx.Done():
		_ = s.process.Kill()
		<-s.process.Done()
	}

	exitCode, _ := s.process.ExitCode()
	status := "completed"
	if exitCode != 0 {
		status = s.terminalStatus()
	}
	if ctx.Err() != nil {
		status = "cancelled"
	}
	s.markDone(status)
}

func (s *processSession) terminalStatus() string {
	events, err := readCanonicalEvents(s.canonicalPath)
	if err != nil {
		return "failed"
	}
	for _, ev := range events {
		switch ev.Type {
		case parse.EventComplete, parse.EventResult:
			return "completed"
		}
	}
	return "failed"
}

func (s *processSession) ID() string { return s.id }

func (s *processSession) Status() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}

func (s *processSession) Events() <-chan SessionEvent { return s.events }
func (s *processSession) Done() <-chan struct{}       { return s.done }

func (s *processSession) TotalCost() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.costUSD
}

func (s *processSession) Kill() error {
	_ = s.process.Kill()
	s.markDone("killed")
	return nil
}

func (s *processSession) Controller() AgentController {
	if s.controller != nil {
		return s.controller
	}
	return noopController{}
}

// processEventInterceptor captures canonical event lines for live publishing.
type processEventInterceptor struct {
	session *processSession
}

func (w *processEventInterceptor) Write(p []byte) (int, error) {
	lines := strings.Split(string(p), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		w.session.consumeCanonicalLine([]byte(line))
	}
	return len(p), nil
}

func (s *processSession) consumeCanonicalLine(line []byte) {
	var ce parse.CanonicalEvent
	if err := json.Unmarshal(line, &ce); err != nil {
		return
	}

	// Deltas are ephemeral — route directly to sink, skip event log and
	// in-memory channel.
	if ce.Type == parse.EventDelta {
		if s.sink != nil {
			s.sink.PublishSessionDelta(s.id, ce.Message, ce.Timestamp)
		}
		return
	}

	s.writeHeartbeat(ce.Timestamp)

	// Notify the controller of turn-boundary events so it can track state
	// without consuming stdout.
	if s.controller != nil {
		s.controller.NotifyEvent(string(ce.Type))
	}

	if ce.CostUSD > 0 {
		s.mu.Lock()
		s.costUSD += ce.CostUSD
		s.mu.Unlock()
	}

	message := strings.TrimSpace(ce.Message)
	if message == "" {
		message = string(ce.Type)
	}

	s.publish(SessionEvent{
		Type:      string(ce.Type),
		Message:   message,
		Timestamp: ce.Timestamp,
		CostUSD:   ce.CostUSD,
		TokensIn:  ce.TokensIn,
		TokensOut: ce.TokensOut,
	})

	if s.eventWriter != nil {
		if record, ok := eventFromCanonical(s.id, ce); ok {
			if err := s.eventWriter.Append(context.Background(), record); err != nil {
				s.publish(SessionEvent{
					Type:      "warning",
					Message:   "event log append failed: " + err.Error(),
					Timestamp: nowUTC(),
				})
			}
		}
	}

	if s.sink != nil {
		if ev, ok := FormatEventLine(s.id, ce); ok {
			s.sink.PublishSessionEvent(s.id, ev)
		}
	}

	if ce.Type == parse.EventInit {
		s.emitPromptEvent(ce.Timestamp)
	}
}

func (s *processSession) emitPromptEvent(timestamp time.Time) {
	if s.promptLogged {
		return
	}
	s.promptLogged = true
	prompt := strings.TrimSpace(s.prompt)
	if prompt == "" {
		return
	}
	if timestamp.IsZero() {
		timestamp = nowUTC()
	}
	s.publish(SessionEvent{
		Type:      string(parse.EventAction),
		Message:   prompt,
		Timestamp: timestamp,
	})

	if s.eventWriter == nil {
		return
	}
	payload, err := json.Marshal(map[string]any{
		"tool":    "prompt",
		"action":  "prompt_injected",
		"message": s.prompt,
	})
	if err != nil {
		return
	}
	promptEvent := event.Event{
		Type:      event.EventAction,
		Payload:   payload,
		Timestamp: timestamp,
		SessionID: s.id,
	}
	if err := s.eventWriter.Append(context.Background(), promptEvent); err != nil {
		s.publish(SessionEvent{
			Type:      "warning",
			Message:   "event log append failed: " + err.Error(),
			Timestamp: nowUTC(),
		})
	}

	if s.sink != nil {
		s.sink.PublishSessionEvent(s.id, promptEvent)
	}
}

func (s *processSession) writeHeartbeat(timestamp time.Time) {
	if strings.TrimSpace(s.canonicalPath) == "" {
		return
	}
	if timestamp.IsZero() {
		timestamp = nowUTC()
	}
	path := filepath.Join(filepath.Dir(s.canonicalPath), "heartbeat.json")
	payload, err := json.Marshal(struct {
		Timestamp  time.Time `json:"timestamp"`
		TTLSeconds int       `json:"ttl_seconds"`
	}{
		Timestamp:  timestamp.UTC(),
		TTLSeconds: sessionHeartbeatTTLSeconds,
	})
	if err != nil {
		return
	}
	_ = os.WriteFile(path, payload, 0o644)
}

func (s *processSession) publish(ev SessionEvent) {
	if ev.Timestamp.IsZero() {
		ev.Timestamp = nowUTC()
	}
	select {
	case <-s.done:
		return
	default:
	}
	select {
	case s.events <- ev:
	default:
		select {
		case <-s.events:
			s.dropped.Add(1)
		default:
		}
		select {
		case <-s.done:
			return
		default:
		}
		select {
		case s.events <- ev:
		default:
		}
	}
}

func (s *processSession) markDone(status string) {
	s.doneOnce.Do(func() {
		s.mu.Lock()
		s.status = status
		s.mu.Unlock()
		close(s.done)
	})
}

func (s *processSession) closeEventsWhenDone() {
	<-s.done
	s.wg.Wait()
	s.emitDroppedSummary()
	s.eventsMu.Do(func() {
		close(s.events)
	})
}

func (s *processSession) emitDroppedSummary() {
	dropped := s.dropped.Load()
	if dropped == 0 {
		return
	}
	message := fmt.Sprintf("dropped %d live session event(s) because the consumer was slow", dropped)
	select {
	case s.events <- SessionEvent{Type: "warning", Message: message, Timestamp: nowUTC()}:
	default:
		select {
		case <-s.events:
		default:
		}
		select {
		case s.events <- SessionEvent{Type: "warning", Message: message, Timestamp: nowUTC()}:
		default:
		}
	}
}

var _ Session = (*processSession)(nil)
