package dispatcher

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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
	}
}

func (s *processSession) start(ctx context.Context) {
	s.writeHeartbeat(nowUTC())
	s.wg.Add(2)

	go func() {
		defer s.wg.Done()
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
	status := "completed"
	if exitCode != 0 {
		status = s.terminalStatus(exitCode)
	}
	if ctx.Err() != nil {
		status = "cancelled"
	}
	s.markDone(status)
}

func (s *processSession) terminalStatus(exitCode int) string {
	// Negative exit codes represent signal-based termination.
	// Treat these as runtime crashes.
	if exitCode < 0 {
		return "failed"
	}
	events, err := readCanonicalEvents(s.canonicalPath)
	if err != nil {
		return "failed"
	}
	sawLifecycleEvent := false
	for _, ev := range events {
		switch ev.Type {
		case parse.EventComplete, parse.EventResult:
			return "completed"
		case parse.EventInit, parse.EventAction, parse.EventError, parse.EventDelta:
			sawLifecycleEvent = true
		}
	}
	if sawLifecycleEvent {
		return "completed"
	}
	return "failed"
}

func (s *processSession) Terminate() error {
	if err := s.process.Terminate(); err != nil {
		return err
	}
	return nil
}

func (s *processSession) ForceKill() error {
	if err := s.process.ForceKill(); err != nil {
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
