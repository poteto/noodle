package adapter

import "strings"

type BacklogStatus string

const (
	BacklogStatusOpen       BacklogStatus = "open"
	BacklogStatusInProgress BacklogStatus = "in_progress"
	BacklogStatusDone       BacklogStatus = "done"
)

type Estimate string

const (
	EstimateSmall  Estimate = "small"
	EstimateMedium Estimate = "medium"
	EstimateLarge  Estimate = "large"
)

type BacklogItem struct {
	ID          string        `json:"id"`
	Title       string        `json:"title"`
	Description string        `json:"description,omitempty"`
	Section     string        `json:"section,omitempty"`
	Status      BacklogStatus `json:"status"`
	Tags        []string      `json:"tags,omitempty"`
	Estimate    Estimate      `json:"estimate,omitempty"`
	Plan        string        `json:"plan,omitempty"`
}

func isValidBacklogStatus(status BacklogStatus) bool {
	switch status {
	case BacklogStatusOpen, BacklogStatusInProgress, BacklogStatusDone:
		return true
	default:
		return false
	}
}

func isValidEstimate(estimate Estimate) bool {
	if estimate == "" {
		return true
	}
	switch estimate {
	case EstimateSmall, EstimateMedium, EstimateLarge:
		return true
	default:
		return false
	}
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
