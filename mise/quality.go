package mise

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// QualityVerdict is one quality review result, read from .noodle/quality/.
type QualityVerdict struct {
	SessionID string    `json:"session_id"`
	TargetID  string    `json:"target_id,omitempty"`
	Accept    bool      `json:"accept"`
	Feedback  string    `json:"feedback,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// readQualityVerdicts reads verdict files from .noodle/quality/.
// Returns most recent 20, sorted newest first (by Timestamp).
func readQualityVerdicts(runtimeDir string) ([]QualityVerdict, error) {
	qualityDir := filepath.Join(runtimeDir, "quality")
	entries, err := os.ReadDir(qualityDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []QualityVerdict{}, nil
		}
		return nil, err
	}

	verdicts := make([]QualityVerdict, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(qualityDir, entry.Name()))
		if err != nil {
			continue
		}
		var v QualityVerdict
		if err := json.Unmarshal(data, &v); err != nil {
			continue
		}
		verdicts = append(verdicts, v)
	}

	sort.SliceStable(verdicts, func(i, j int) bool {
		return verdicts[i].Timestamp.After(verdicts[j].Timestamp)
	})

	if len(verdicts) > 20 {
		verdicts = verdicts[:20]
	}

	return verdicts, nil
}
