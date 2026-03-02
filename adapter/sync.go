package adapter

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strings"
)

func ParseBacklogItems(ndjson string) ([]BacklogItem, error) {
	scanner := bufio.NewScanner(strings.NewReader(ndjson))
	items := make([]BacklogItem, 0)
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var item BacklogItem
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			return nil, fmt.Errorf("parse backlog sync line %d: %w", lineNumber, err)
		}
		if err := validateBacklogItem(item, lineNumber); err != nil {
			return nil, err
		}
		item.Tags = normalizeTags(item.Tags)
		items = append(items, item)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read backlog sync output: %w", err)
	}
	return items, nil
}

func validateBacklogItem(item BacklogItem, lineNumber int) error {
	if strings.TrimSpace(item.ID) == "" {
		return fmt.Errorf("backlog sync line %d: missing required field id", lineNumber)
	}
	if strings.TrimSpace(item.Title) == "" {
		return fmt.Errorf("backlog sync line %d: missing required field title", lineNumber)
	}
	return nil
}

