package adapter

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const maxBacklogSyncLineBytes = 1024 * 1024 // 1 MiB

func ParseBacklogItems(ndjson string) ([]BacklogItem, []string, error) {
	scanner := bufio.NewScanner(strings.NewReader(ndjson))
	scanner.Buffer(make([]byte, 0, 64*1024), maxBacklogSyncLineBytes)
	items := make([]BacklogItem, 0)
	warnings := make([]string, 0)
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var item BacklogItem
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			warnings = append(warnings, fmt.Sprintf("parse backlog sync line %d: %v", lineNumber, err))
			continue
		}
		if warning := validateBacklogItem(item, lineNumber); warning != "" {
			warnings = append(warnings, warning)
			continue
		}
		items = append(items, item)
	}
	if err := scanner.Err(); err != nil {
		if errors.Is(err, bufio.ErrTooLong) {
			warnings = append(warnings, fmt.Sprintf("backlog sync line %d: token too long (max %d bytes)", lineNumber+1, maxBacklogSyncLineBytes))
			return items, warnings, nil
		}
		return nil, nil, fmt.Errorf("read backlog sync output: %w", err)
	}
	return items, warnings, nil
}

func validateBacklogItem(item BacklogItem, lineNumber int) string {
	if strings.TrimSpace(item.ID) == "" {
		return fmt.Sprintf("backlog sync line %d: missing required field id", lineNumber)
	}
	if strings.TrimSpace(item.Title) == "" {
		return fmt.Sprintf("backlog sync line %d: missing required field title", lineNumber)
	}
	return ""
}
