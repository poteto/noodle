package loop

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/poteto/noodle/internal/filex"
)

type PendingReviewItem struct {
	ID           string   `json:"id"`
	TaskKey      string   `json:"task_key,omitempty"`
	Title        string   `json:"title,omitempty"`
	Prompt       string   `json:"prompt,omitempty"`
	Provider     string   `json:"provider,omitempty"`
	Model        string   `json:"model,omitempty"`
	Runtime      string   `json:"runtime,omitempty"`
	Skill        string   `json:"skill,omitempty"`
	Plan         []string `json:"plan,omitempty"`
	Rationale    string   `json:"rationale,omitempty"`
	WorktreeName string   `json:"worktree_name"`
	WorktreePath string   `json:"worktree_path"`
	SessionID    string   `json:"session_id,omitempty"`
}

type pendingReviewFile struct {
	Items []PendingReviewItem `json:"items"`
}

func pendingReviewFilePath(runtimeDir string) string {
	return filepath.Join(runtimeDir, "pending-review.json")
}

func ReadPendingReview(runtimeDir string) ([]PendingReviewItem, error) {
	path := pendingReviewFilePath(runtimeDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []PendingReviewItem{}, nil
		}
		return nil, err
	}

	var payload pendingReviewFile
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	if payload.Items == nil {
		return []PendingReviewItem{}, nil
	}
	return payload.Items, nil
}

func (l *Loop) parkPendingReview(cook *activeCook) error {
	l.pendingReview[cook.queueItem.ID] = &pendingReviewCook{
		queueItem:    cook.queueItem,
		worktreeName: cook.worktreeName,
		worktreePath: cook.worktreePath,
		sessionID:    cook.session.ID(),
	}
	return l.writePendingReview()
}

func (l *Loop) writePendingReview() error {
	items := make([]PendingReviewItem, 0, len(l.pendingReview))
	for _, pending := range l.pendingReview {
		if pending == nil {
			continue
		}
		items = append(items, PendingReviewItem{
			ID:           pending.queueItem.ID,
			TaskKey:      pending.queueItem.TaskKey,
			Title:        pending.queueItem.Title,
			Prompt:       pending.queueItem.Prompt,
			Provider:     pending.queueItem.Provider,
			Model:        pending.queueItem.Model,
			Runtime:      pending.queueItem.Runtime,
			Skill:        pending.queueItem.Skill,
			Plan:         pending.queueItem.Plan,
			Rationale:    pending.queueItem.Rationale,
			WorktreeName: pending.worktreeName,
			WorktreePath: pending.worktreePath,
			SessionID:    pending.sessionID,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})
	payload := pendingReviewFile{Items: items}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return filex.WriteFileAtomic(pendingReviewFilePath(l.runtimeDir), append(data, '\n'))
}

func (l *Loop) loadPendingReview() error {
	items, err := ReadPendingReview(l.runtimeDir)
	if err != nil {
		return err
	}

	next := make(map[string]*pendingReviewCook, len(items))
	for _, item := range items {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			continue
		}
		next[id] = &pendingReviewCook{
			queueItem: QueueItem{
				ID:        id,
				TaskKey:   strings.TrimSpace(item.TaskKey),
				Title:     strings.TrimSpace(item.Title),
				Prompt:    strings.TrimSpace(item.Prompt),
				Provider:  strings.TrimSpace(item.Provider),
				Model:     strings.TrimSpace(item.Model),
				Runtime:   strings.TrimSpace(item.Runtime),
				Skill:     strings.TrimSpace(item.Skill),
				Plan:      item.Plan,
				Rationale: strings.TrimSpace(item.Rationale),
			},
			worktreeName: strings.TrimSpace(item.WorktreeName),
			worktreePath: strings.TrimSpace(item.WorktreePath),
			sessionID:    strings.TrimSpace(item.SessionID),
		}
	}
	l.pendingReview = next
	return nil
}
