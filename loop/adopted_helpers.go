package loop

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/poteto/noodle/internal/stringx"
)

var recoverySuffixRegexp = regexp.MustCompile(`-recover-(\d+)$`)

func (l *Loop) readSessionStatus(sessionID string) (string, bool, error) {
	metaPath := filepath.Join(l.runtimeDir, "sessions", sessionID, "meta.json")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, err
	}
	var payload struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return "", false, err
	}
	return stringx.Normalize(payload.Status), true, nil
}

func (l *Loop) buildAdoptedCook(targetID string, sessionID string, status string) (*cookHandle, bool, error) {
	orders, err := l.currentOrders()
	if err != nil {
		return nil, false, err
	}
	for _, order := range orders.Orders {
		if order.ID != targetID {
			continue
		}
		idx, stg := activeStageForOrder(order)
		if idx < 0 || stg == nil {
			return nil, false, nil
		}
		name := cookBaseName(order.ID, idx, stg.TaskKey)
		worktreePath := l.worktreePath(name)
		return &cookHandle{
			cookIdentity: cookIdentity{
				orderID:    order.ID,
				stageIndex: idx,
				stage:      *stg,
				plan:       order.Plan,
			},
			orderStatus: order.Status,
			session: &adoptedSession{
				id:     sessionID,
				status: status,
			},
			worktreeName: name,
			worktreePath: worktreePath,
			attempt:      recoveryChainLength(name),
		}, true, nil
	}

	return nil, false, nil
}

func (l *Loop) dropAdoptedTarget(targetID string, sessionID string) {
	delete(l.cooks.adoptedTargets, targetID)
	filtered := l.cooks.adoptedSessions[:0]
	for _, id := range l.cooks.adoptedSessions {
		if id == sessionID {
			continue
		}
		filtered = append(filtered, id)
	}
	l.cooks.adoptedSessions = filtered
}

func recoveryChainLength(name string) int {
	name = strings.TrimSpace(name)
	if name == "" {
		return 0
	}
	matches := recoverySuffixRegexp.FindStringSubmatch(name)
	if len(matches) != 2 {
		return 0
	}
	var value int
	_, _ = fmt.Sscanf(matches[1], "%d", &value)
	if value < 0 {
		return 0
	}
	return value
}
