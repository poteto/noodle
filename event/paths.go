package event

import "path/filepath"

func sessionDir(runtimeDir, sessionID string) string {
	return filepath.Join(runtimeDir, "sessions", sessionID)
}

func sessionEventsPath(runtimeDir, sessionID string) string {
	return filepath.Join(sessionDir(runtimeDir, sessionID), "events.ndjson")
}

func ticketsPath(runtimeDir string) string {
	return filepath.Join(runtimeDir, "tickets.json")
}
