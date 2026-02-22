package debate

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
)

const (
	// DefaultMaxRounds is used when a debate does not specify a max.
	DefaultMaxRounds = 6
)

var roundFileRegexp = regexp.MustCompile(`^round-(\d+)\.md$`)

type Role string

const (
	RoleReviewer  Role = "reviewer"
	RoleResponder Role = "responder"
)

// Debate tracks one debate directory and runtime settings.
type Debate struct {
	ID        string
	Target    string
	Dir       string
	MaxRounds int
}

// Round is one debate turn persisted as round-N.md.
type Round struct {
	Number  int
	Role    Role
	Content string
}

// Verdict is persisted at verdict.json.
type Verdict struct {
	Consensus bool   `json:"consensus"`
	Summary   string `json:"summary"`
}

// Store manages debates rooted under one directory.
type Store struct {
	rootDir string
	mu      sync.Mutex
}

// NewStore initializes a store at rootDir.
func NewStore(rootDir string) (*Store, error) {
	rootDir = strings.TrimSpace(rootDir)
	if rootDir == "" {
		return nil, fmt.Errorf("root directory is required")
	}
	if err := os.MkdirAll(rootDir, 0o755); err != nil {
		return nil, fmt.Errorf("create root directory: %w", err)
	}
	return &Store{rootDir: rootDir}, nil
}

// Create creates a new debate directory under the store root.
func (s *Store) Create(target string, maxRounds int) (Debate, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return Debate{}, fmt.Errorf("target is required")
	}
	maxRounds = normalizedMaxRounds(maxRounds)

	debate := Debate{
		ID:        DebateID(target),
		Target:    target,
		MaxRounds: maxRounds,
	}
	debate.Dir = filepath.Join(s.rootDir, debate.ID)

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.Mkdir(debate.Dir, 0o755); err != nil {
		if os.IsExist(err) {
			return Debate{}, fmt.Errorf("debate directory already exists")
		}
		return Debate{}, fmt.Errorf("create debate directory: %w", err)
	}
	return debate, nil
}

// AddRound appends one round file if max rounds has not been reached.
func (s *Store) AddRound(debate Debate, role Role, content string) (Round, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return Round{}, fmt.Errorf("round content is required")
	}
	if !isValidRole(role) {
		return Round{}, fmt.Errorf("round role is invalid")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	debate, err := s.resolveDebate(debate)
	if err != nil {
		return Round{}, err
	}

	rounds, err := s.readRounds(debate)
	if err != nil {
		return Round{}, err
	}
	nextRound := nextRoundNumber(rounds)
	if nextRound > debate.MaxRounds {
		return Round{}, fmt.Errorf("max rounds reached")
	}

	expectedRole := roleForRound(nextRound)
	if role != expectedRole {
		return Round{}, fmt.Errorf("expected %s role for round %d", expectedRole, nextRound)
	}

	path := roundPath(debate.Dir, nextRound)
	if err := os.WriteFile(path, []byte(content+"\n"), 0o644); err != nil {
		return Round{}, fmt.Errorf("write round file: %w", err)
	}

	return Round{
		Number:  nextRound,
		Role:    role,
		Content: content,
	}, nil
}

// ReadRounds returns persisted rounds in ascending order.
func (s *Store) ReadRounds(debate Debate) ([]Round, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	debate, err := s.resolveDebate(debate)
	if err != nil {
		return nil, err
	}
	return s.readRounds(debate)
}

// WriteVerdict writes verdict.json for the debate.
func (s *Store) WriteVerdict(debate Debate, verdict Verdict) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	debate, err := s.resolveDebate(debate)
	if err != nil {
		return err
	}

	verdict.Summary = strings.TrimSpace(verdict.Summary)
	encoded, err := json.MarshalIndent(verdict, "", "  ")
	if err != nil {
		return fmt.Errorf("encode verdict: %w", err)
	}
	if err := os.WriteFile(verdictPath(debate.Dir), append(encoded, '\n'), 0o644); err != nil {
		return fmt.Errorf("write verdict file: %w", err)
	}
	return nil
}

// ReadVerdict reads verdict.json for the debate.
func (s *Store) ReadVerdict(debate Debate) (Verdict, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	debate, err := s.resolveDebate(debate)
	if err != nil {
		return Verdict{}, err
	}

	content, err := os.ReadFile(verdictPath(debate.Dir))
	if err != nil {
		return Verdict{}, fmt.Errorf("read verdict file: %w", err)
	}

	var verdict Verdict
	if err := json.Unmarshal(content, &verdict); err != nil {
		return Verdict{}, fmt.Errorf("parse verdict file: %w", err)
	}
	return verdict, nil
}

// HasConsensus returns verdict consensus from verdict.json.
func (s *Store) HasConsensus(debate Debate) (bool, error) {
	verdict, err := s.ReadVerdict(debate)
	if err != nil {
		return false, err
	}
	return verdict.Consensus, nil
}

// IsComplete returns true when consensus is true or max rounds are reached.
func (s *Store) IsComplete(debate Debate) (bool, error) {
	verdict, err := s.ReadVerdict(debate)
	if err == nil && verdict.Consensus {
		return true, nil
	}
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return false, err
	}

	rounds, err := s.ReadRounds(debate)
	if err != nil {
		return false, err
	}
	maxRounds := normalizedMaxRounds(debate.MaxRounds)
	return len(rounds) >= maxRounds, nil
}

// DebateID returns slugified target + short hash.
func DebateID(target string) string {
	normalized := strings.TrimSpace(target)
	slug := slugify(normalized)
	sum := sha1.Sum([]byte(normalized))
	hash := hex.EncodeToString(sum[:4])
	return slug + "-" + hash
}

func (s *Store) resolveDebate(debate Debate) (Debate, error) {
	debate.ID = strings.TrimSpace(debate.ID)
	if debate.ID == "" {
		return Debate{}, fmt.Errorf("debate ID is required")
	}
	debate.MaxRounds = normalizedMaxRounds(debate.MaxRounds)
	if strings.TrimSpace(debate.Dir) == "" {
		debate.Dir = filepath.Join(s.rootDir, debate.ID)
	}
	info, err := os.Stat(debate.Dir)
	if err != nil {
		return Debate{}, fmt.Errorf("read debate directory: %w", err)
	}
	if !info.IsDir() {
		return Debate{}, fmt.Errorf("debate directory is not a directory")
	}
	return debate, nil
}

func (s *Store) readRounds(debate Debate) ([]Round, error) {
	entries, err := os.ReadDir(debate.Dir)
	if err != nil {
		return nil, fmt.Errorf("read debate directory: %w", err)
	}

	numbers := make([]int, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		matches := roundFileRegexp.FindStringSubmatch(entry.Name())
		if len(matches) != 2 {
			continue
		}

		var number int
		if _, err := fmt.Sscanf(matches[1], "%d", &number); err != nil {
			continue
		}
		if number <= 0 {
			continue
		}
		numbers = append(numbers, number)
	}
	sort.Ints(numbers)

	rounds := make([]Round, 0, len(numbers))
	for _, number := range numbers {
		content, err := os.ReadFile(roundPath(debate.Dir, number))
		if err != nil {
			return nil, fmt.Errorf("read round file: %w", err)
		}
		rounds = append(rounds, Round{
			Number:  number,
			Role:    roleForRound(number),
			Content: strings.TrimSpace(string(content)),
		})
	}
	return rounds, nil
}

func normalizedMaxRounds(maxRounds int) int {
	if maxRounds <= 0 {
		return DefaultMaxRounds
	}
	return maxRounds
}

func roleForRound(number int) Role {
	if number%2 == 1 {
		return RoleReviewer
	}
	return RoleResponder
}

func isValidRole(role Role) bool {
	return role == RoleReviewer || role == RoleResponder
}

func nextRoundNumber(rounds []Round) int {
	next := 1
	for _, round := range rounds {
		if round.Number >= next {
			next = round.Number + 1
		}
	}
	return next
}

func roundPath(debateDir string, number int) string {
	return filepath.Join(debateDir, fmt.Sprintf("round-%d.md", number))
}

func verdictPath(debateDir string) string {
	return filepath.Join(debateDir, "verdict.json")
}

func slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "debate"
	}

	var builder strings.Builder
	hyphenPending := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			if hyphenPending && builder.Len() > 0 {
				builder.WriteByte('-')
			}
			builder.WriteRune(r)
			hyphenPending = false
			continue
		}
		hyphenPending = true
	}

	slug := strings.Trim(builder.String(), "-")
	if slug == "" {
		return "debate"
	}
	return slug
}
