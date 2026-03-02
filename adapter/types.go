package adapter

import "strings"

// Well-known status values. Not exhaustive — adapters may return any string.
const (
	BacklogStatusOpen       = "open"
	BacklogStatusInProgress = "in_progress"
	BacklogStatusDone       = "done"
)

type BacklogItem struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Section     string   `json:"section,omitempty"`
	Status      string   `json:"status,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Estimate    string   `json:"estimate,omitempty"`
	Plan        string   `json:"plan,omitempty"`
}

func normalizeTags(tags []string) []string {
	if len(tags) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		out = append(out, tag)
	}
	if len(out) == 0 {
		return []string{}
	}
	return out
}
