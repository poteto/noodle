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

func ParsePlanItems(ndjson string) ([]PlanItem, error) {
	scanner := bufio.NewScanner(strings.NewReader(ndjson))
	items := make([]PlanItem, 0)
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var item PlanItem
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			return nil, fmt.Errorf("parse plans sync line %d: %w", lineNumber, err)
		}
		if err := validatePlanItem(item, lineNumber); err != nil {
			return nil, err
		}
		item.Tags = normalizeTags(item.Tags)
		items = append(items, item)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read plans sync output: %w", err)
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
	if !isValidBacklogStatus(item.Status) {
		return fmt.Errorf("backlog sync line %d: invalid status %q", lineNumber, item.Status)
	}
	if !isValidEstimate(item.Estimate) {
		return fmt.Errorf("backlog sync line %d: invalid estimate %q", lineNumber, item.Estimate)
	}
	return nil
}

func validatePlanItem(item PlanItem, lineNumber int) error {
	if strings.TrimSpace(item.ID) == "" {
		return fmt.Errorf("plans sync line %d: missing required field id", lineNumber)
	}
	if strings.TrimSpace(item.Title) == "" {
		return fmt.Errorf("plans sync line %d: missing required field title", lineNumber)
	}
	if !isValidPlanStatus(item.Status) {
		return fmt.Errorf("plans sync line %d: invalid status %q", lineNumber, item.Status)
	}
	if len(item.Phases) == 0 {
		return fmt.Errorf("plans sync line %d: missing required field phases", lineNumber)
	}
	for index, phase := range item.Phases {
		if strings.TrimSpace(phase.Name) == "" {
			return fmt.Errorf("plans sync line %d: phase %d missing required field name", lineNumber, index)
		}
		if !isValidPlanPhaseStatus(phase.Status) {
			return fmt.Errorf("plans sync line %d: phase %d invalid status %q", lineNumber, index, phase.Status)
		}
	}
	return nil
}
