# Charmbracelet V2 Migration Gotchas

## Bubble Tea v2 model contract changed in two coupled places

- `Model.View()` must return `tea.View` (use `tea.NewView(...)`), not `string`.
- `tea.WithAltScreen()` no longer exists in v2 program options. Set `view.AltScreen = true` on the returned view.

## Key events moved from `tea.KeyMsg` + `Type/Runes` to `tea.KeyPressMsg` + `Code/Text/Mod`

- Replace `case tea.KeyMsg` handlers with `case tea.KeyPressMsg`.
- Replace `msg.Type` checks with `msg.Code` checks for special keys (`tea.KeyEsc`, `tea.KeyEnter`, arrows, etc).
- Replace `tea.KeyRunes` + `msg.Runes` handling with `msg.Text != ""`.
- For ctrl combos, check modifiers explicitly: `(msg.Mod & tea.ModCtrl) != 0`.
- Shift+tab should be detected as `msg.Code == tea.KeyTab` plus `tea.ModShift`.

## Lip Gloss v2 color fields should use `image/color.Color`

- Theme/component structs that stored `lipgloss.Color` types should move to `color.Color`.
- Keep style call sites unchanged by constructing colors with `lipgloss.Color("#hex")`.
- Default/unset checks must use `nil` instead of empty-string checks.

## Detached tmux capture can fail in this environment

- `go run . start` can exit immediately in detached tmux sessions, so scripted captures may produce empty panes.
- `script`-based pseudo-TTY capture is more reliable for runtime transcript artifacts here.

See also [[plans/39-charmbracelet-v2-upgrade/overview]], [[principles/migrate-callers-then-delete-legacy-apis]], [[principles/prove-it-works]]
