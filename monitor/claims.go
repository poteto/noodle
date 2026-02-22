package monitor

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/poteto/noodle/parse"
)

const defaultMaxEventLineBytes = 64 << 20

type CanonicalClaimsReader struct {
	runtimeDir   string
	maxLineBytes int
}

func NewCanonicalClaimsReader(runtimeDir string) *CanonicalClaimsReader {
	return &CanonicalClaimsReader{
		runtimeDir:   strings.TrimSpace(runtimeDir),
		maxLineBytes: defaultMaxEventLineBytes,
	}
}

func (r *CanonicalClaimsReader) ReadSession(sessionID string) (SessionClaims, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return SessionClaims{}, fmt.Errorf("session ID is required")
	}
	if r.runtimeDir == "" {
		return SessionClaims{}, fmt.Errorf("runtime directory is required")
	}

	path := filepath.Join(r.runtimeDir, "sessions", sessionID, "canonical.ndjson")
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return SessionClaims{SessionID: sessionID}, nil
		}
		return SessionClaims{}, fmt.Errorf("open canonical events: %w", err)
	}
	defer file.Close()

	claims := SessionClaims{SessionID: sessionID}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), r.maxLineBytes)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var event parse.CanonicalEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return SessionClaims{}, fmt.Errorf("parse canonical event: %w", err)
		}

		claims.HasEvents = true
		if claims.FirstEventAt.IsZero() || event.Timestamp.Before(claims.FirstEventAt) {
			claims.FirstEventAt = event.Timestamp
		}
		if event.Timestamp.After(claims.LastEventAt) {
			claims.LastEventAt = event.Timestamp
		}
		if strings.TrimSpace(event.Provider) != "" {
			claims.Provider = strings.TrimSpace(event.Provider)
		}

		switch event.Type {
		case parse.EventAction:
			if message := strings.TrimSpace(event.Message); message != "" {
				claims.LastAction = message
			}
		case parse.EventResult:
			claims.TotalCostUSD += event.CostUSD
			claims.TokensIn += event.TokensIn
			claims.TokensOut += event.TokensOut
		case parse.EventError:
			claims.Failed = true
		}
	}
	if err := scanner.Err(); err != nil {
		return SessionClaims{}, fmt.Errorf("read canonical events: %w", err)
	}

	return claims, nil
}
