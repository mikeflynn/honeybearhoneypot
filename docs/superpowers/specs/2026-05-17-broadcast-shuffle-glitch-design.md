# Broadcast Additions: Shuffle CWD & Glitch

Two new admin broadcast actions for the GUI broadcast modal, alongside the existing Knock / Notice / Fake Join / Matrix / Confetti / Kick All buttons.

## 1. Shuffle CWD

Randomly relocates every connected session to a different directory in the fake filesystem. Each session lands somewhere independent; the change is silent (no echoed message), so users only notice on the next prompt or `pwd` / `ls`.

### Components

- **GUI button** (`internal/gui/broadcast.go`): "Shuffle CWD" with `theme.FolderIcon()`, wired to `honeypot.ActionShuffleDirs`.
- **Filesystem helper** (`internal/honeypot/filesystem/filesystem.go` or new file): `AllDirectories() []*Node` — BFS walk from `GetRoot()`, returns every `Node` where `IsDirectory()` is true, skipping cloaked nodes.
- **`ChangeDirMsg` extension** (`internal/honeypot/filesystem/filesystem.go`): add `Silent bool` field. Default zero value preserves current echo behavior.
- **Model handler update** (`internal/honeypot/model.go:130`): if `msg.Silent`, skip the `m.output += "\ncd <path>\n"` line; still update `m.currentDir` and `PWD` env.
- **Action** (`internal/honeypot/registry.go`): `ActionShuffleDirs()` collects dirs once, then under `sessionsMu.RLock` iterates sessions, picking a fresh random dir per session and sending `filesystem.ChangeDirMsg{Node: d, Path: d.Path, Silent: true}` to that session's program.

### Edge cases

- Empty directory list (shouldn't happen in practice): no-op, return early.
- A session is mid-`matrix`/`confetti` mode: the message still queues; cwd updates underneath the running effect and takes visible effect when the user returns to the prompt.

## 2. Glitch Effect

Visual disruption broadcast — character corruption, line shift, and color flicker layered together for ~2.5 seconds, then auto-clears. Modeled on `ActionConfetti` / `ActionMatrix` (uses the `SetRunningCmd` takeover pattern).

### Components

- **GUI button** (`internal/gui/broadcast.go`): "Glitch" with `theme.BrokenImageIcon()` (or similar), wired to `honeypot.ActionGlitch`.
- **New package** `internal/honeypot/glitch/glitch.go`:
  - `Model` struct: base screen snapshot (string), elapsed ticks, `*rand.Rand`.
  - Messages: `Start{Base string}`, `Tick{}`.
  - `func New(base string) *Model`.
  - `func (m *Model) Update(msg tea.Msg) (tea.Cmd, bool)` — returns next tick cmd and a `done` flag once elapsed >= 2.5s worth of ticks.
  - `func (m *Model) View() string` — applies all three effects each render:
    - **Char corruption:** replace ~10–15% of non-whitespace runes with a random glyph from `▓▒░@#%&*`.
    - **Line shift:** for ~20% of lines, prepend a random small whitespace offset, or swap with an adjacent line.
    - **Color flicker:** wrap ~5% of lines in a lipgloss reversed style.
  - Tick interval ~50ms (20fps), total duration ~2.5s.
- **Model wiring** (`internal/honeypot/model.go`):
  - Add `glitchModel *glitch.Model` field.
  - `case glitch.Start`: capture current `m.output` (or composed view) as base, init `glitchModel`, return first tick cmd.
  - `case glitch.Tick`: forward to `glitchModel.Update`; if done, clear `glitchModel`.
  - `View()`: if `runningCmd == "glitch"` and `glitchModel != nil`, return `glitchModel.View()` instead of the default render.
- **Action** (`internal/honeypot/registry.go`): `ActionGlitch()` broadcasts `filesystem.SetRunningCmd("glitch")` and `glitch.Start{}` (base is captured per-session in the handler). A goroutine waits ~2.6s, then broadcasts `filesystem.SetRunningCmd("")` and `filesystem.ClearOutputMsg("")` to restore — mirroring `ActionConfetti`.

### Edge cases

- Session connects mid-glitch: it just won't be affected (not registered until ready).
- Session disconnects mid-glitch: registry cleanup already handles this; the cleanup goroutine's broadcast is a no-op for that session.
- Empty `output` base: effects degrade gracefully (nothing to corrupt), still safe.

## Out of Scope (YAGNI)

- No config knobs for either feature — fixed behavior.
- No per-user "send home" undo for Shuffle.
- No selective inclusion/exclusion lists for shuffle destinations.
- No persistent "glitch mode" — strictly one-shot.

## Testing

- `filesystem.AllDirectories()` — unit test against a small synthetic tree, verify cloaked dirs are excluded.
- `ChangeDirMsg{Silent: true}` handler path — model unit test confirming no `"cd "` appears in `m.output` but `currentDir` and `PWD` update.
- Glitch model — unit test that `Update` returns `done == true` after expected number of ticks, and `View()` returns a non-empty string for a sample base.
- Manual: launch GUI, connect 2+ SSH sessions, fire each button, verify behavior.
