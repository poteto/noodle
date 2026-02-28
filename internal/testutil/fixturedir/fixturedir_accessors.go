package fixturedir

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func (inventory FixtureInventory) Names() []string {
	names := make([]string, 0, len(inventory.Cases))
	for _, fixtureCase := range inventory.Cases {
		names = append(names, fixtureCase.Name)
	}
	sort.Strings(names)
	return names
}

func (fixtureCase FixtureCase) Section(name string) (string, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", false
	}
	for key, value := range fixtureCase.Sections {
		if strings.EqualFold(strings.TrimSpace(key), name) {
			return value, true
		}
	}
	return "", false
}

func (fixtureCase FixtureCase) State(stateID string) (FixtureState, bool) {
	stateID = strings.TrimSpace(stateID)
	for _, state := range fixtureCase.States {
		if strings.EqualFold(state.ID, stateID) {
			return state, true
		}
	}
	return FixtureState{}, false
}

func (state FixtureState) FilePath(relPath string) (string, bool) {
	relPath = filepath.ToSlash(filepath.Clean(strings.TrimSpace(relPath)))
	relPath = strings.TrimPrefix(relPath, "./")
	path, ok := state.Files[relPath]
	return path, ok
}

func (state FixtureState) MustReadFile(tb testing.TB, relPath string) []byte {
	tb.Helper()
	path, ok := state.FilePath(relPath)
	if !ok {
		tb.Fatalf("state %s missing file %s (available: %v)", state.ID, relPath, state.FileOrder)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		tb.Fatalf("read state file %s: %v", path, err)
	}
	return data
}

func (state FixtureState) MustReadText(tb testing.TB, relPath string) string {
	tb.Helper()
	return string(state.MustReadFile(tb, relPath))
}
