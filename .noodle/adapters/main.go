package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	reSectionHeading = regexp.MustCompile(`^##\s+(.+?)\s*$`)
	reTodoLine       = regexp.MustCompile(`^([0-9]+)\. \[([ xX])\] (.*)$`)
	reTag            = regexp.MustCompile(`#[A-Za-z0-9_-]+`)
	reEstimate       = regexp.MustCompile(`~(small|medium|large)`)
	rePlanRef        = regexp.MustCompile(`\[\[plans/([0-9]+-[^/]+)/overview\]\]`)
	reSpaces         = regexp.MustCompile(`\s+`)
	rePlansIndexRef  = regexp.MustCompile(`\[\[(plans/[0-9][0-9]-[^/]+/overview)\]\]`)
	rePlanDir        = regexp.MustCompile(`^[0-9][0-9]-`)
	rePhaseChecklist = regexp.MustCompile(`^- \[([ xX])\] (.*)$`)
	rePhaseName      = regexp.MustCompile(`—\s*(.+)$`)
)

type backlogPayload struct {
	Title   string `json:"title"`
	Section string `json:"section"`
}

type planPayload struct {
	Title string `json:"title"`
	Slug  string `json:"slug"`
}

type phasePayload struct {
	Name string `json:"name"`
}

func main() {
	if len(os.Args) < 2 {
		die("command is required")
	}

	var err error
	switch os.Args[1] {
	case "backlog-sync":
		err = backlogSync()
	case "backlog-add":
		err = backlogAdd()
	case "backlog-done":
		err = backlogDone(arg(2))
	case "backlog-edit":
		err = backlogEdit(arg(2))
	case "plans-sync":
		err = plansSync()
	case "plan-create":
		err = planCreate()
	case "plan-done":
		err = planDone(arg(2))
	case "plan-phase-add":
		err = planPhaseAdd(arg(2))
	default:
		err = fmt.Errorf("unknown command %q", os.Args[1])
	}
	if err != nil {
		die(err.Error())
	}
}

func arg(index int) string {
	if len(os.Args) <= index {
		return ""
	}
	return strings.TrimSpace(os.Args[index])
}

func die(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}

func backlogSync() error {
	todosFile := envOrDefault("NOODLE_TODOS_FILE", "brain/todos.md")
	file, err := os.Open(todosFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	section := ""
	encoder := json.NewEncoder(os.Stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if m := reSectionHeading.FindStringSubmatch(line); len(m) == 2 {
			section = strings.TrimSpace(m[1])
			continue
		}

		m := reTodoLine.FindStringSubmatch(line)
		if len(m) != 4 {
			continue
		}

		id := m[1]
		mark := m[2]
		raw := m[3]
		status := "open"
		if mark == "x" || mark == "X" {
			status = "done"
		}

		tags := reTag.FindAllString(raw, -1)
		normalizedTags := make([]string, 0, len(tags))
		for _, tag := range tags {
			tag = strings.TrimPrefix(tag, "#")
			if tag == "" {
				continue
			}
			normalizedTags = append(normalizedTags, tag)
		}

		estimate := ""
		if em := reEstimate.FindStringSubmatch(raw); len(em) == 2 {
			estimate = em[1]
		}

		plan := ""
		if pm := rePlanRef.FindStringSubmatch(raw); len(pm) == 2 {
			plan = "brain/plans/" + pm[1] + "/overview.md"
		}

		title := raw
		title = reTag.ReplaceAllString(title, "")
		title = reEstimate.ReplaceAllString(title, "")
		title = rePlanRef.ReplaceAllString(title, "")
		title = reSpaces.ReplaceAllString(strings.TrimSpace(title), " ")

		item := map[string]any{
			"id":     id,
			"title":  title,
			"status": status,
			"tags":   normalizedTags,
		}
		if section != "" {
			item["section"] = section
		}
		if plan != "" {
			item["plan"] = plan
		}
		if estimate != "" {
			item["estimate"] = estimate
		}
		if err := encoder.Encode(item); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func backlogAdd() error {
	todosFile := envOrDefault("NOODLE_TODOS_FILE", "brain/todos.md")
	if err := ensureTodosFile(todosFile); err != nil {
		return err
	}

	payload, err := readBacklogPayload(os.Stdin)
	if err != nil {
		return err
	}
	if strings.TrimSpace(payload.Title) == "" {
		return fmt.Errorf("title required")
	}
	section := strings.TrimSpace(payload.Section)
	if section == "" {
		section = "Inbox"
	}

	lines, err := readLines(todosFile)
	if err != nil {
		return err
	}
	id := nextTodoID(lines)

	if !hasSection(lines, section) {
		if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) != "" {
			lines = append(lines, "")
		}
		lines = append(lines, "## "+section)
	}
	lines = append(lines, fmt.Sprintf("%d. [ ] %s", id, payload.Title))
	lines = updateNextID(lines, id+1)

	if err := writeLinesAtomic(todosFile, lines); err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, strconv.Itoa(id))
	return nil
}

func backlogDone(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("id required")
	}
	todosFile := envOrDefault("NOODLE_TODOS_FILE", "brain/todos.md")
	lines, err := readLines(todosFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("todos file not found")
		}
		return err
	}

	prefix := id + ". [ ]"
	replacement := id + ". [x]"
	for i, line := range lines {
		if strings.HasPrefix(line, prefix) {
			lines[i] = replacement + strings.TrimPrefix(line, prefix)
		}
	}
	return writeLinesAtomic(todosFile, lines)
}

func backlogEdit(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("id required")
	}
	payload, err := readBacklogPayload(os.Stdin)
	if err != nil {
		return err
	}
	if strings.TrimSpace(payload.Title) == "" {
		return fmt.Errorf("title required")
	}

	todosFile := envOrDefault("NOODLE_TODOS_FILE", "brain/todos.md")
	lines, err := readLines(todosFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("todos file not found")
		}
		return err
	}

	re := regexp.MustCompile(`^` + regexp.QuoteMeta(id) + `\. \[[ xX]\] `)
	for i, line := range lines {
		if re.MatchString(line) {
			prefix := re.FindString(line)
			lines[i] = prefix + payload.Title
		}
	}
	return writeLinesAtomic(todosFile, lines)
}

func plansSync() error {
	plansDir := envOrDefault("NOODLE_PLANS_DIR", "brain/plans")
	indexFile := filepath.Join(plansDir, "index.md")
	file, err := os.Open(indexFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	encoder := json.NewEncoder(os.Stdout)
	for scanner.Scan() {
		line := scanner.Text()
		m := rePlansIndexRef.FindStringSubmatch(line)
		if len(m) != 2 {
			continue
		}
		rel := m[1]
		id := extractPlanID(rel)
		if id == "" {
			continue
		}
		dir := filepath.Join(plansDir, strings.TrimSuffix(strings.TrimPrefix(rel, "plans/"), "/overview"))
		overview := filepath.Join(dir, "overview.md")

		status := "active"
		title := filepath.Base(dir)
		if overviewLines, err := readLines(overview); err == nil {
			if s := parseFrontmatterStatus(overviewLines); s != "" {
				status = s
			}
			if h := parseHeading(overviewLines); h != "" {
				title = h
			}
		}

		phases := []map[string]string{}
		if overviewLines, err := readLines(overview); err == nil {
			firstOpen := true
			for _, pline := range overviewLines {
				mm := rePhaseChecklist.FindStringSubmatch(pline)
				if len(mm) != 3 {
					continue
				}
				phaseName := phaseNameFromChecklistLine(pline)
				phaseStatus := "pending"
				if mm[1] == "x" || mm[1] == "X" {
					phaseStatus = "done"
				} else if firstOpen {
					phaseStatus = "active"
					firstOpen = false
				}
				phases = append(phases, map[string]string{
					"name":   phaseName,
					"status": phaseStatus,
				})
			}
		}
		if len(phases) == 0 {
			phases = append(phases, map[string]string{"name": "phase-1", "status": "active"})
		}

		item := map[string]any{
			"id":     id,
			"title":  title,
			"status": status,
			"phases": phases,
			"tags":   []string{},
		}
		if err := encoder.Encode(item); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func planCreate() error {
	payload, err := readPlanPayload(os.Stdin)
	if err != nil {
		return err
	}
	if strings.TrimSpace(payload.Title) == "" {
		return fmt.Errorf("title required")
	}
	plansDir := envOrDefault("NOODLE_PLANS_DIR", "brain/plans")
	indexFile := filepath.Join(plansDir, "index.md")

	if err := os.MkdirAll(plansDir, 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(indexFile); errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(indexFile, []byte("# Plans\n\n"), 0o644); err != nil {
			return err
		}
	}

	slug := strings.TrimSpace(payload.Slug)
	if slug == "" {
		slug = slugify(payload.Title)
	}

	id, err := nextPlanID(plansDir)
	if err != nil {
		return err
	}
	id2 := fmt.Sprintf("%02d", id)
	dir := filepath.Join(plansDir, id2+"-"+slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	now := time.Now().Format("2006-01-02")
	overview := strings.Join([]string{
		"---",
		fmt.Sprintf("id: %d", id),
		fmt.Sprintf("created: %s", now),
		fmt.Sprintf("updated: %s", now),
		"status: active",
		"---",
		"",
		"# " + payload.Title,
		"",
		"## Phases",
		"",
		fmt.Sprintf("- [ ] [[plans/%s-%s/phase-01-scaffold]] - Scaffold", id2, slug),
		"",
	}, "\n")
	if err := os.WriteFile(filepath.Join(dir, "overview.md"), []byte(overview), 0o644); err != nil {
		return err
	}

	phase := strings.Join([]string{
		fmt.Sprintf("Back to [[plans/%s-%s/overview]]", id2, slug),
		"",
		"# Phase 1 - Scaffold",
		"",
		"## Goal",
		"## Changes",
		"## Data Structures",
		"## Verification",
		"### Static",
		"### Runtime",
		"",
	}, "\n")
	if err := os.WriteFile(filepath.Join(dir, "phase-01-scaffold.md"), []byte(phase), 0o644); err != nil {
		return err
	}

	idxLines, err := readLines(indexFile)
	if err != nil {
		return err
	}
	idxLines = append(idxLines, fmt.Sprintf("- [ ] [[plans/%s-%s/overview]]", id2, slug))
	if err := writeLinesAtomic(indexFile, idxLines); err != nil {
		return err
	}

	fmt.Fprintln(os.Stdout, id2)
	return nil
}

func planDone(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("id required")
	}
	idNum, err := strconv.Atoi(id)
	if err != nil {
		return fmt.Errorf("id required")
	}
	id2 := fmt.Sprintf("%02d", idNum)

	plansDir := envOrDefault("NOODLE_PLANS_DIR", "brain/plans")
	planDir, err := findPlanDir(plansDir, id2)
	if err != nil {
		return err
	}

	overviewFile := filepath.Join(planDir, "overview.md")
	overviewLines, err := readLines(overviewFile)
	if err != nil {
		return err
	}
	for i, line := range overviewLines {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "status:") {
			overviewLines[i] = "status: done"
		}
	}
	if err := writeLinesAtomic(overviewFile, overviewLines); err != nil {
		return err
	}

	indexFile := filepath.Join(plansDir, "index.md")
	if indexLines, err := readLines(indexFile); err == nil {
		needle := "- [ ] [[plans/" + id2 + "-"
		repl := "- [x] [[plans/" + id2 + "-"
		for i, line := range indexLines {
			if strings.Contains(line, needle) {
				indexLines[i] = strings.Replace(line, needle, repl, 1)
			}
		}
		if err := writeLinesAtomic(indexFile, indexLines); err != nil {
			return err
		}
	}

	todosFile := envOrDefault("NOODLE_TODOS_FILE", "brain/todos.md")
	if todoLines, err := readLines(todosFile); err == nil {
		re := regexp.MustCompile(`^([0-9]+\. )\[ \](.*\[\[plans/` + regexp.QuoteMeta(id2) + `-[^\]]*/overview\]\].*)$`)
		for i, line := range todoLines {
			if m := re.FindStringSubmatch(line); len(m) == 3 {
				todoLines[i] = m[1] + "[x]" + m[2]
			}
		}
		if err := writeLinesAtomic(todosFile, todoLines); err != nil {
			return err
		}
	}

	return nil
}

func planPhaseAdd(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("id required")
	}
	idNum, err := strconv.Atoi(id)
	if err != nil {
		return fmt.Errorf("id required")
	}
	id2 := fmt.Sprintf("%02d", idNum)

	payload, err := readPhasePayload(os.Stdin)
	if err != nil {
		return err
	}
	if strings.TrimSpace(payload.Name) == "" {
		return fmt.Errorf("name required")
	}

	plansDir := envOrDefault("NOODLE_PLANS_DIR", "brain/plans")
	planDir, err := findPlanDir(plansDir, id2)
	if err != nil {
		return err
	}

	slug := slugify(payload.Name)
	phaseFiles, err := filepath.Glob(filepath.Join(planDir, "phase-*.md"))
	if err != nil {
		return err
	}
	num := len(phaseFiles) + 1
	num2 := fmt.Sprintf("%02d", num)
	base := filepath.Base(planDir)
	phaseFile := filepath.Join(planDir, "phase-"+num2+"-"+slug+".md")

	phase := strings.Join([]string{
		"Back to [[plans/" + base + "/overview]]",
		"",
		fmt.Sprintf("# Phase %d - %s", num, payload.Name),
		"",
		"## Goal",
		"## Changes",
		"## Data Structures",
		"## Verification",
		"### Static",
		"### Runtime",
		"",
	}, "\n")
	if err := os.WriteFile(phaseFile, []byte(phase), 0o644); err != nil {
		return err
	}

	overviewFile := filepath.Join(planDir, "overview.md")
	lines, err := readLines(overviewFile)
	if err != nil {
		return err
	}
	lines = append(lines, fmt.Sprintf("- [ ] [[plans/%s/phase-%s-%s]] - %s", base, num2, slug, payload.Name))
	return writeLinesAtomic(overviewFile, lines)
}

func readBacklogPayload(r io.Reader) (backlogPayload, error) {
	var payload backlogPayload
	data, err := io.ReadAll(r)
	if err != nil {
		return payload, err
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return payload, nil
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return payload, err
	}
	return payload, nil
}

func readPlanPayload(r io.Reader) (planPayload, error) {
	var payload planPayload
	data, err := io.ReadAll(r)
	if err != nil {
		return payload, err
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return payload, err
	}
	return payload, nil
}

func readPhasePayload(r io.Reader) (phasePayload, error) {
	var payload phasePayload
	data, err := io.ReadAll(r)
	if err != nil {
		return payload, err
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return payload, err
	}
	return payload, nil
}

func ensureTodosFile(path string) error {
	_, statErr := os.Stat(path)
	if statErr == nil {
		return nil
	}
	if !errors.Is(statErr, os.ErrNotExist) {
		return statErr
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	seed := strings.Join([]string{
		"# Todos",
		"",
		"<!-- next-id: 1 -->",
		"",
		"## Inbox",
		"",
	}, "\n")
	return os.WriteFile(path, []byte(seed), 0o644)
}

func readLines(path string) ([]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	content = bytes.ReplaceAll(content, []byte("\r\n"), []byte("\n"))
	trimmed := strings.TrimSuffix(string(content), "\n")
	if trimmed == "" {
		return []string{}, nil
	}
	return strings.Split(trimmed, "\n"), nil
}

func writeLinesAtomic(path string, lines []string) error {
	data := strings.Join(lines, "\n")
	if len(lines) > 0 {
		data += "\n"
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(data), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func nextTodoID(lines []string) int {
	for _, line := range lines {
		if strings.Contains(line, "next-id:") {
			if m := regexp.MustCompile(`next-id:\s*([0-9]+)`).FindStringSubmatch(line); len(m) == 2 {
				if id, err := strconv.Atoi(m[1]); err == nil {
					return id
				}
			}
		}
	}
	maxID := 0
	for _, line := range lines {
		if m := reTodoLine.FindStringSubmatch(line); len(m) == 4 {
			if id, err := strconv.Atoi(m[1]); err == nil && id > maxID {
				maxID = id
			}
		}
	}
	return maxID + 1
}

func updateNextID(lines []string, nextID int) []string {
	updated := false
	for i, line := range lines {
		if strings.Contains(line, "next-id:") {
			lines[i] = fmt.Sprintf("<!-- next-id: %d -->", nextID)
			updated = true
			break
		}
	}
	if updated {
		return lines
	}
	out := make([]string, 0, len(lines)+2)
	inserted := false
	for i, line := range lines {
		out = append(out, line)
		if i == 1 {
			out = append(out, "", fmt.Sprintf("<!-- next-id: %d -->", nextID))
			inserted = true
		}
	}
	if !inserted {
		out = append([]string{"# Todos", "", fmt.Sprintf("<!-- next-id: %d -->", nextID), ""}, lines...)
	}
	return out
}

func hasSection(lines []string, section string) bool {
	target := "## " + section
	for _, line := range lines {
		if strings.TrimSpace(line) == target {
			return true
		}
	}
	return false
}

func extractPlanID(rel string) string {
	rel = strings.TrimSpace(rel)
	rel = strings.TrimPrefix(rel, "plans/")
	parts := strings.SplitN(rel, "-", 2)
	if len(parts) == 0 {
		return ""
	}
	id := strings.TrimSpace(parts[0])
	if len(id) != 2 {
		return ""
	}
	return id
}

func parseFrontmatterStatus(lines []string) string {
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "status:") {
			status := strings.TrimSpace(strings.TrimPrefix(trim, "status:"))
			switch status {
			case "draft", "active", "done":
				return status
			}
		}
	}
	return ""
}

func parseHeading(lines []string) string {
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(trim, "# "))
		}
	}
	return ""
}

func nextPlanID(plansDir string) (int, error) {
	entries, err := os.ReadDir(plansDir)
	if err != nil {
		return 0, err
	}
	maxID := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !rePlanDir.MatchString(name) {
			continue
		}
		id, err := strconv.Atoi(name[:2])
		if err != nil {
			continue
		}
		if id > maxID {
			maxID = id
		}
	}
	return maxID + 1, nil
}

func findPlanDir(plansDir, id2 string) (string, error) {
	entries, err := os.ReadDir(plansDir)
	if err != nil {
		return "", err
	}
	prefix := id2 + "-"
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if strings.HasPrefix(entry.Name(), prefix) {
			return filepath.Join(plansDir, entry.Name()), nil
		}
	}
	return "", fmt.Errorf("plan not found")
}

func slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		return "plan"
	}
	return slug
}

func envOrDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func phaseNameFromChecklistLine(line string) string {
	if match := rePhaseName.FindStringSubmatch(line); len(match) == 2 {
		name := strings.TrimSpace(match[1])
		if name != "" {
			return name
		}
	}
	if idx := strings.LastIndex(line, " - "); idx >= 0 {
		name := strings.TrimSpace(line[idx+3:])
		if name != "" {
			return name
		}
	}
	return "phase"
}
