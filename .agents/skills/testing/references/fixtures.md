# Fixture Test Reference

## Directory Structure

```
package/testdata/
└── fixture-name/
    ├── expected.md          # Expected output with YAML frontmatter
    ├── .noodle.toml         # Optional config override
    ├── state-01/
    │   ├── input.json       # or input.ndjson
    │   └── .noodle/         # Runtime state (queue.json, etc.)
    └── state-02/            # Additional states numbered sequentially
```

## expected.md Format

The file starts with YAML frontmatter delimited by `---`, then markdown sections containing fenced JSON blocks.

**Frontmatter fields:**

    ---
    schema_version: 1
    expected_failure: false
    bug: false
    source_hash: pending
    ---

**Sections** (as H2 headings with fenced json blocks):
- `## Expected` — Expected output (format varies by package)
- `## Expected Error` — Error expectations (optional)
- `## Runtime Dump` — Loop package uses this for multi-state transitions

### Frontmatter Fields

| Field | Type | Purpose |
|-------|------|---------|
| `schema_version` | int | Always `1` |
| `expected_failure` | bool | Test expects an error |
| `bug` | bool | Known failing test — requires `expected_failure: true` |
| `source_hash` | string | SHA256 of inputs, auto-synced by `make fixtures-hash MODE=sync` |

## Marking a Fixture as a Known Bug

Set both fields in frontmatter:

```yaml
expected_failure: true
bug: true
```

Add a placeholder `## Expected Error` section with JSON body `{"any": true}`.

Verify it appears in `make bugs`.

## Scaffold Script

```sh
./scripts/scaffold-fixture.sh <package-dir> <fixture-name> [state-count]

# Examples:
./scripts/scaffold-fixture.sh loop my-new-test 2
./scripts/scaffold-fixture.sh parse edge-case-input 1
```

Creates the directory, empty input files, and expected.md with `source_hash: pending`. Runs `fixturehash sync` automatically.

## Recording Loop Fixtures

```sh
# Record new expected output
make fixtures-loop MODE=record

# Or manually:
NOODLE_LOOP_FIXTURE_MODE=record go test ./loop -run TestLoopDirectoryFixtures -count=1
```

After recording, sync hashes:

```sh
make fixtures-hash MODE=sync
```

## Test Helper API (internal/testutil/fixturedir)

```go
// Discover and load
inventory := fixturedir.LoadInventory(t, "testdata")

// Validate structure
fixturedir.AssertValidFixtureRoot(t, "testdata")

// Parse state data
input := fixturedir.ParseStateJSON[MyInput](t, state, "input.json")

// Get expected section
expected := fixturedir.MustSection(t, fixtureCase, "Expected")

// Assert errors
fixturedir.AssertError(t, "context", err, fixtureCase.ExpectedError)
```

## Typical Fixture Test Pattern

```go
func TestDirectoryFixtures(t *testing.T) {
    fixturedir.AssertValidFixtureRoot(t, "testdata")
    inventory := fixturedir.LoadInventory(t, "testdata")

    for _, fc := range inventory.Cases {
        fc := fc
        t.Run(fc.Name, func(t *testing.T) {
            state := fc.States[0]
            input := fixturedir.ParseStateJSON[MyType](t, state, "input.json")

            actual, err := MyFunction(input)

            fixturedir.AssertError(t, "test", err, fc.ExpectedError)
            if fc.ExpectedError != nil {
                return
            }

            expected := fixturedir.MustSection(t, fc, "Expected")
            // ... compare actual vs expected
        })
    }
}
```
