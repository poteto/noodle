# Go Best Practices — Code Patterns

## 1. Minimal main

```go
func main() {
    if os.Getenv("APP_PROFILE") != "" {
        go func() {
            slog.Info("Serving pprof", "addr", "localhost:6060")
            http.ListenAndServe("localhost:6060", nil) //nolint:errcheck
        }()
    }
    cmd.Execute()
}
```

Import `_ "net/http/pprof"` for side-effect registration. The env gate keeps
diagnostics out of the production path.

## 2. Single Bootstrap Path

```go
func setupApp(cmd *cobra.Command) (*App, error) {
    cfg, err := config.Load(cwd)
    if err != nil {
        return nil, err
    }
    db, err := openDB(ctx, cfg.DataDir)
    if err != nil {
        return nil, err
    }
    return NewApp(ctx, db, cfg)
}
```

Every mode calls `setupApp`. One path, one set of bugs.

## 3. Ordered Graceful Shutdown

```go
func (a *App) Shutdown() {
    // Phase 1: cancel dependents that must finish before resources close.
    a.workers.CancelAll()

    // Phase 2: independent cleanup in parallel with a shared timeout.
    var wg sync.WaitGroup
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    for _, fn := range a.cleanups {
        wg.Go(func() {
            if err := fn(ctx); err != nil {
                slog.Error("Cleanup failed", "error", err)
            }
        })
    }
    wg.Wait()
}
```

- Ordered phases — dependents before their dependencies.
- Shared timeout context bounds the entire parallel phase.
- Register cleanup funcs during setup, run them at teardown.

## 4. Non-Blocking Fanout

### Publisher — drop on full channel

```go
func (b *Broker[T]) Publish(event T) {
    b.mu.RLock()
    defer b.mu.RUnlock()

    for ch := range b.subs {
        select {
        case ch <- event:
        default:
            // Slow subscriber — drop, never block publisher.
        }
    }
}
```

### Consumer — timeout-bounded forward

```go
func forward[T any](ctx context.Context, in <-chan T, out chan<- T, timeout time.Duration) {
    timer := time.NewTimer(0)
    <-timer.C
    defer timer.Stop()

    for {
        select {
        case v, ok := <-in:
            if !ok { return }
            // Safe timer reset: Stop, drain, Reset.
            if !timer.Stop() {
                select {
                case <-timer.C:
                default:
                }
            }
            timer.Reset(timeout)

            select {
            case out <- v:
            case <-timer.C:
                slog.Debug("Dropped message", "reason", "slow consumer")
            case <-ctx.Done():
                return
            }
        case <-ctx.Done():
            return
        }
    }
}
```

The `Stop` → drain → `Reset` sequence prevents timer leaks and deadlocks.
Always write a dedicated test for this (see pattern 5).

## 5. Concurrency Testing

### synctest for deterministic timing

```go
func TestNormalFlow(t *testing.T) {
    synctest.Test(t, func(t *testing.T) {
        ch := make(chan int, 1)
        go func() { ch <- 42 }()

        time.Sleep(10 * time.Millisecond)
        synctest.Wait()

        select {
        case v := <-ch:
            require.Equal(t, 42, v)
        default:
            t.Fatal("expected value")
        }
    })
}
```

### goleak for goroutine leak detection

```go
func TestNoLeak(t *testing.T) {
    defer goleak.VerifyNone(t)
    // ... test that starts goroutines ...
}
```

Add `goleak.VerifyNone` to at least one test per concurrent subsystem.

### Deadlock regression tests

```go
func TestTimerDrainDeadlock(t *testing.T) {
    synctest.Test(t, func(t *testing.T) {
        // Reproduce the exact sequence that triggered the deadlock.
        // Publish → timeout fires (drop) → publish again → cancel.
        done := make(chan struct{})
        go func() {
            // ... exercise the code path ...
            close(done)
        }()

        select {
        case <-done:
        case <-time.After(5 * time.Second):
            t.Fatal("hung — likely timer drain deadlock")
        }
    })
}
```

### Test fixture pattern

```go
type fixture struct {
    cancel context.CancelFunc
    wg     sync.WaitGroup
    out    chan Msg
}

func newFixture(t *testing.T) *fixture {
    t.Helper()
    ctx, cancel := context.WithCancel(t.Context())
    t.Cleanup(cancel)
    f := &fixture{cancel: cancel, out: make(chan Msg, 10)}
    // wire up goroutines ...
    return f
}
```

Use `t.Context()` and `t.Cleanup` to tie lifecycle to the test.

## 6. Layered Config Loading

```go
func discoverConfigs(cwd string) []string {
    paths := []string{globalConfigPath()} // lowest priority

    found, _ := walkUpFor(cwd, "app.json", ".app.json")
    slices.Reverse(found) // closest to CWD = highest priority

    return append(paths, found...)
}

func loadConfigs(paths []string) (*Config, error) {
    var layers [][]byte
    for _, p := range paths {
        data, err := os.ReadFile(p)
        if errors.Is(err, fs.ErrNotExist) { continue }
        if err != nil { return nil, fmt.Errorf("reading %s: %w", p, err) }
        if len(data) == 0 { continue }
        layers = append(layers, data)
    }
    merged := deepMergeJSON(layers...)
    var cfg Config
    return &cfg, json.Unmarshal(merged, &cfg)
}
```

Precedence: global (lowest) → discovered project configs (closest to CWD
wins) → CLI flags (highest). Missing and empty files silently skipped.

## 7. Cross-Platform Paths

```go
func configDir() string {
    if v := os.Getenv("APP_CONFIG_DIR"); v != "" {
        return v
    }
    if v := os.Getenv("XDG_CONFIG_HOME"); v != "" {
        return filepath.Join(v, appName)
    }
    return filepath.Join(home(), ".config", appName)
}

func dataDir() string {
    if v := os.Getenv("APP_DATA_DIR"); v != "" {
        return v
    }
    if v := os.Getenv("XDG_DATA_HOME"); v != "" {
        return filepath.Join(v, appName)
    }
    if runtime.GOOS == "windows" {
        base := cmp.Or(os.Getenv("LOCALAPPDATA"),
            filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Local"))
        return filepath.Join(base, appName)
    }
    return filepath.Join(home(), ".local", "share", appName)
}
```

One function per path concern. Resolution: env override → XDG → platform
default → fallback. Centralize in a single file.

## 8. Secure Debug HTTP Logging

```go
type debugTransport struct{ base http.RoundTripper }

func (d *debugTransport) RoundTrip(req *http.Request) (*http.Response, error) {
    if slog.Default().Enabled(req.Context(), slog.LevelDebug) {
        slog.Debug("HTTP request", "method", req.Method, "url", req.URL)
    }

    start := time.Now()
    resp, err := d.base.RoundTrip(req)

    if err == nil && slog.Default().Enabled(req.Context(), slog.LevelDebug) {
        slog.Debug("HTTP response",
            "status", resp.StatusCode,
            "headers", redactHeaders(resp.Header),
            "ms", time.Since(start).Milliseconds())
    }
    return resp, err
}

func redactHeaders(h http.Header) map[string][]string {
    out := make(map[string][]string, len(h))
    for k, v := range h {
        lk := strings.ToLower(k)
        if strings.Contains(lk, "authorization") ||
            strings.Contains(lk, "api-key") ||
            strings.Contains(lk, "token") ||
            strings.Contains(lk, "secret") {
            out[k] = []string{"[REDACTED]"}
        } else {
            out[k] = v
        }
    }
    return out
}
```

Gate on `slog.LevelDebug` to skip body drain / allocation in production.
If logging request/response bodies, drain and restore via `io.NopCloser`.

## 9. Golden Test Matrices

```go
func TestRender(t *testing.T) {
    for name, layout := range layouts {
        t.Run(name, func(t *testing.T) {
            for name, theme := range themes {
                t.Run(name, func(t *testing.T) {
                    t.Parallel()
                    w := NewWidget(WithLayout(layout), WithTheme(theme))
                    golden.RequireEqual(t, []byte(w.Render()))
                })
            }
        })
    }
}
```

- Dimension maps produce N*M subtests, all parallel.
- `golden.RequireEqual` diffs against committed `.golden` files (update with
  `-update` flag).
- Sweep continuous ranges (width 1-120, height 1-30) to catch off-by-one
  rendering bugs.

## 10. Focused Linters + CI

### .golangci.yml

```yaml
version: "2"
linters:
  enable:
    - bodyclose        # unclosed HTTP response bodies
    - noctx            # HTTP requests without context
    - sqlclosecheck    # unclosed sql.Rows
    - tparallel        # missing t.Parallel()
    - staticcheck      # comprehensive static analysis
    - misspell         # comment/string typos
    - gofumpt          # strict formatting
  disable:
    - errcheck         # too noisy — use explicit checks where it matters
```

### Task runner

```yaml
tasks:
  test:
    cmds: [go test -race -failfast ./...]
  lint:
    cmds:
      - task: lint:custom
      - golangci-lint run --timeout=5m
  lint:custom:
    cmds: [./scripts/check_log_style.sh]
```

### Project-specific lint script

```bash
#!/bin/bash
if grep -rE 'slog\.(Error|Info|Warn|Debug)\("[a-z]' --include="*.go" .; then
  echo "Log messages must start with a capital letter." && exit 1
fi
```

Always `-race` in tests. `-failfast` for local iteration. Custom scripts for
style rules that golangci-lint can't express.
