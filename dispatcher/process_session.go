package dispatcher

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/poteto/noodle/event"
	"github.com/poteto/noodle/parse"
)

// processSession implements Session backed by a direct child process.
// It runs the stamp processor in-process (like spritesSession) rather than
// as a shell pipeline stage.
type processSession struct {
	sessionBase
	process    *ProcessHandle
	controller *claudeController // nil for non-steerable sessions
	warnings   []string

	streamDone     chan struct{}
	streamDoneOnce sync.Once

	trackerMu   sync.Mutex
	sawInit     bool
	sawAction   bool
	sawResult   bool
	sawComplete bool
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
		sessionBase: newSessionBase(sessionBaseConfig{
			id:            cfg.id,
			eventWriter:   cfg.eventWriter,
			stampedPath:   cfg.stampedPath,
			canonicalPath: cfg.canonicalPath,
			prompt:        cfg.prompt,
			sink:          cfg.sink,
		}),
		process:    cfg.process,
		controller: cfg.controller,
		warnings:   append([]string(nil), cfg.warnings...),
		streamDone: make(chan struct{}),
	}
}

func (s *processSession) start(ctx context.Context) {
	s.writeHeartbeat(nowUTC())
	s.wg.Add(2)

	go func() {
		defer s.wg.Done()
		defer s.closeStreamDone()
		interceptor := &canonicalLineInterceptor{onLine: func(line []byte) {
			s.consumeCanonicalLine(line, s.processHook)
		}}
		s.processStream(ctx, s.process.Stdout(), interceptor)
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

func (s *processSession) processHook(ce parse.CanonicalEvent) {
	s.trackerMu.Lock()
	switch ce.Type {
	case parse.EventInit:
		s.sawInit = true
	case parse.EventAction:
		s.sawAction = true
	case parse.EventResult:
		s.sawResult = true
	case parse.EventComplete:
		s.sawComplete = true
	}
	s.trackerMu.Unlock()

	s.writeHeartbeat(ce.Timestamp)
	if s.controller != nil {
		s.controller.NotifyEvent(string(ce.Type))
	}
}

func (s *processSession) waitForExit(ctx context.Context) {
	select {
	case <-s.process.Done():
	case <-ctx.Done():
		_ = s.process.ForceKill()
		<-s.process.Done()
	}

	exitCode, _ := s.process.ExitCode()
	s.resolveAndMarkDone(exitCode, ctx.Err() != nil)
}

func (s *processSession) closeStreamDone() {
	s.streamDoneOnce.Do(func() {
		close(s.streamDone)
	})
}

func (s *processSession) resolveAndMarkDone(exitCode int, ctxCancelled bool) {
	<-s.streamDone
	outcome := s.resolveOutcome(exitCode, ctxCancelled)
	s.doneOnce.Do(func() {
		s.mu.Lock()
		s.status = outcome.Status.String()
		s.outcome = outcome
		s.mu.Unlock()
		close(s.done)
	})
}

func (s *processSession) resolveOutcome(exitCode int, ctxCancelled bool) SessionOutcome {
	s.trackerMu.Lock()
	sawComplete := s.sawComplete
	sawResult := s.sawResult
	sawAction := s.sawAction
	sawInit := s.sawInit
	s.trackerMu.Unlock()

	outcome := SessionOutcome{
		ExitCode:       exitCode,
		HasDeliverable: sawComplete || sawResult,
	}

	switch {
	case sawComplete || sawResult:
		outcome.Status = StatusCompleted
		outcome.Reason = "completion event observed"
	case ctxCancelled:
		outcome.Status = StatusCancelled
		outcome.Reason = "context cancelled before completion"
	case exitCode < 0:
		outcome.Status = StatusKilled
		outcome.Reason = "process terminated by signal"
	case sawAction:
		outcome.Status = StatusFailed
		outcome.Reason = "no turn completed"
	case sawInit:
		outcome.Status = StatusFailed
		outcome.Reason = "no work produced"
	default:
		outcome.Status = StatusFailed
		outcome.Reason = "no events emitted"
	}

	return outcome
}

func (s *processSession) Terminate() error {
	return s.process.Terminate()
}

func (s *processSession) ForceKill() error {
	err := s.process.ForceKill()
	if err != nil {
		return err
	}
	s.markDone("killed")
	return nil
}

func (s *processSession) Controller() AgentController {
	if s.controller != nil {
		return s.controller
	}
	return noopController{}
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

var _ Session = (*processSession)(nil)
