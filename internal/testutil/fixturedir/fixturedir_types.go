package fixturedir

import "regexp"

const FixtureSchemaVersion = 1

var stateDirPattern = regexp.MustCompile(`^state-(\d{2})$`)

type FixtureLayout struct {
	RootPath       string
	ExpectedPath   string
	BaseConfigPath string
	States         []FixtureStateDir
}

type FixtureStateDir struct {
	ID         string
	Index      int
	Path       string
	ConfigPath string
}

type FixtureConfigScope struct {
	BaseConfigPath    string
	StateOverridePath string
}

type FixtureMetadata struct {
	ExpectedFailure bool
	Bug             bool
	SchemaVersion   int
	SourceHash      string
}

type FixtureCase struct {
	Name          string
	Path          string
	Layout        FixtureLayout
	Metadata      FixtureMetadata
	ExpectedError *ErrorExpectation
	Sections      map[string]string
	States        []FixtureState
}

type FixtureState struct {
	ID          string
	Path        string
	ConfigScope FixtureConfigScope
	Files       map[string]string
	FileOrder   []string
}

type FixtureInventory struct {
	Root  string
	Cases []FixtureCase
}

type FixtureValidationIssue struct {
	Path     string
	Severity string
	Message  string
}
