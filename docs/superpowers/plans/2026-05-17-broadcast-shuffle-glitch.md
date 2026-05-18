# Broadcast Additions (Shuffle CWD + Glitch) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add two new admin broadcast actions to the GUI broadcast modal — "Shuffle CWD" (silently relocates each session to a random directory) and "Glitch" (~2.5s combined char/line/color disruption effect).

**Architecture:** Shuffle CWD reuses the existing `filesystem.ChangeDirMsg` flow with a new `Silent` flag; a new helper enumerates dirs from `GetRoot()`. Glitch follows the existing `SetRunningCmd("…")` takeover pattern used by `matrix`/`confetti`: a new `internal/honeypot/glitch` package owns a Bubble Tea sub-model that snapshots current output and renders corrupted frames via lipgloss.

**Tech Stack:** Go 1.24, Charmbracelet Bubble Tea v2, lipgloss v2, Fyne GUI v2, standard `math/rand`.

**Project conventions (from `CLAUDE.md`):**
- Do not commit. The user manages git. Each task ends with a "stop & let user review" step, not a `git commit`.
- `go fmt ./...` after Go edits.
- Tests via `go test ./...` or per-package `go test ./internal/...`.

---

## File Map

**Create:**
- `internal/honeypot/glitch/glitch.go` — sub-model package implementing the glitch effect.
- `internal/honeypot/glitch/glitch_test.go` — unit tests for glitch model.
- `internal/honeypot/filesystem/dirs_test.go` — unit test for `AllDirectories`.

**Modify:**
- `internal/honeypot/filesystem/filesystem.go` — add `Silent bool` field to `ChangeDirMsg`, add `AllDirectories()` helper.
- `internal/honeypot/model.go` — honor `ChangeDirMsg.Silent`; wire `glitch.Model` (field, Update dispatch, View case).
- `internal/honeypot/pot.go` — initialize empty `glitch.Model` in the model constructor (matches existing pattern for `confetti`, `matrix`).
- `internal/honeypot/registry.go` — add `ActionShuffleDirs()` and `ActionGlitch()`.
- `internal/gui/broadcast.go` — add "Shuffle CWD" and "Glitch" buttons to the modal grid.

---

## Task 1: `filesystem.AllDirectories()` helper

**Files:**
- Modify: `internal/honeypot/filesystem/filesystem.go`
- Test: `internal/honeypot/filesystem/dirs_test.go` (new)

- [ ] **Step 1: Write the failing test**

Create `internal/honeypot/filesystem/dirs_test.go`:

```go
package filesystem

import (
	"testing"
)

func TestAllDirectories(t *testing.T) {
	Initialize()

	dirs := AllDirectories()
	if len(dirs) == 0 {
		t.Fatal("AllDirectories returned empty slice")
	}

	// Every returned node must be a directory.
	for _, n := range dirs {
		if !n.IsDirectory() {
			t.Errorf("AllDirectories returned non-directory: %s", n.Path)
		}
		if n.IsCloaked() {
			t.Errorf("AllDirectories returned cloaked node: %s", n.Path)
		}
	}

	// Root must be included.
	root := GetRoot()
	found := false
	for _, n := range dirs {
		if n == root {
			found = true
			break
		}
	}
	if !found {
		t.Error("AllDirectories did not include root")
	}

	// Some well-known dirs from the seeded filesystem must be reachable.
	wantPaths := []string{"/home/you", "/etc", "/var"}
	for _, want := range wantPaths {
		seen := false
		for _, n := range dirs {
			if n.Path == want {
				seen = true
				break
			}
		}
		if !seen {
			t.Errorf("AllDirectories missing expected dir %q", want)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/honeypot/filesystem/ -run TestAllDirectories -v`
Expected: FAIL with `undefined: AllDirectories`.

- [ ] **Step 3: Implement `AllDirectories`**

Append to `internal/honeypot/filesystem/filesystem.go` (at end of file):

```go
// AllDirectories returns every directory node reachable from the system root
// via a breadth-first walk, skipping cloaked nodes.
func AllDirectories() []*Node {
	root := GetRoot()
	if root == nil {
		return nil
	}
	var out []*Node
	queue := []*Node{root}
	for len(queue) > 0 {
		n := queue[0]
		queue = queue[1:]
		if n == nil || n.IsCloaked() || !n.IsDirectory() {
			continue
		}
		out = append(out, n)
		for _, child := range n.Children {
			if child != nil && child.IsDirectory() {
				queue = append(queue, child)
			}
		}
	}
	return out
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/honeypot/filesystem/ -run TestAllDirectories -v`
Expected: PASS.

- [ ] **Step 5: Format**

Run: `go fmt ./internal/honeypot/filesystem/...`

- [ ] **Step 6: Stop and let user review changes before continuing.**

---

## Task 2: `ChangeDirMsg.Silent` flag

**Files:**
- Modify: `internal/honeypot/filesystem/filesystem.go` (lines 31–34)
- Modify: `internal/honeypot/model.go` (around line 130)
- Test: extend `internal/honeypot/filesystem/dirs_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/honeypot/filesystem/dirs_test.go`:

```go
func TestChangeDirMsgHasSilentField(t *testing.T) {
	// Compile-time check that the Silent field exists.
	_ = ChangeDirMsg{Path: "/tmp", Silent: true}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/honeypot/filesystem/ -run TestChangeDirMsgHasSilentField -v`
Expected: FAIL with `unknown field Silent in struct literal`.

- [ ] **Step 3: Add the field**

In `internal/honeypot/filesystem/filesystem.go`, replace:

```go
	ChangeDirMsg       struct {
		Path string
		Node *Node
	}
```

with:

```go
	ChangeDirMsg       struct {
		Path   string
		Node   *Node
		Silent bool
	}
```

- [ ] **Step 4: Honor `Silent` in the model handler**

In `internal/honeypot/model.go`, find the existing handler (currently lines 130–136):

```go
		case filesystem.ChangeDirMsg:
			m.currentDir = msg.Node
			if m.environ == nil {
				m.environ = make(map[string]string)
			}
			m.environ["PWD"] = msg.Path
			m.output += m.outputStyle.Render(fmt.Sprintf("\ncd %s\n", msg.Path))
```

Replace with:

```go
		case filesystem.ChangeDirMsg:
			m.currentDir = msg.Node
			if m.environ == nil {
				m.environ = make(map[string]string)
			}
			m.environ["PWD"] = msg.Path
			if !msg.Silent {
				m.output += m.outputStyle.Render(fmt.Sprintf("\ncd %s\n", msg.Path))
			}
```

- [ ] **Step 5: Verify tests + build**

Run: `go test ./internal/honeypot/filesystem/ -v`
Expected: PASS.

Run: `go build ./...`
Expected: clean exit.

- [ ] **Step 6: Format**

Run: `go fmt ./internal/honeypot/...`

- [ ] **Step 7: Stop and let user review changes before continuing.**

---

## Task 3: `ActionShuffleDirs` registry action

**Files:**
- Modify: `internal/honeypot/registry.go`

- [ ] **Step 1: Add the action**

Append to `internal/honeypot/registry.go`:

```go
// ActionShuffleDirs silently relocates every active session to a randomly
// chosen directory in the fake filesystem. Each session gets an independent
// random destination.
func ActionShuffleDirs() {
	dirs := filesystem.AllDirectories()
	if len(dirs) == 0 {
		return
	}
	sessionsMu.RLock()
	defer sessionsMu.RUnlock()
	for _, p := range sessions {
		d := dirs[rand.Intn(len(dirs))]
		p.Send(filesystem.ChangeDirMsg{Node: d, Path: d.Path, Silent: true})
	}
}
```

Add `"math/rand"` to the import block at the top of the file if not already present.

- [ ] **Step 2: Build**

Run: `go build ./...`
Expected: clean exit.

- [ ] **Step 3: Format**

Run: `go fmt ./internal/honeypot/...`

- [ ] **Step 4: Stop and let user review changes before continuing.**

---

## Task 4: Wire "Shuffle CWD" button into GUI

**Files:**
- Modify: `internal/gui/broadcast.go`

- [ ] **Step 1: Add the button to the modal**

In `internal/gui/broadcast.go`, replace the `showBroadcastModal` function body:

```go
func showBroadcastModal() {
	knock := widget.NewButtonWithIcon("Knock", theme.QuestionIcon(), honeypot.ActionKnock)
	notice := widget.NewButtonWithIcon("Notice", theme.WarningIcon(), honeypot.ActionSystemNotice)
	join := widget.NewButtonWithIcon("Fake Join", theme.AccountIcon(), honeypot.ActionFakeJoin)
	mtx := widget.NewButtonWithIcon("Matrix", theme.VisibilityIcon(), honeypot.ActionMatrix)
	conf := widget.NewButtonWithIcon("Confetti", theme.ColorPaletteIcon(), honeypot.ActionConfetti)
	shuffle := widget.NewButtonWithIcon("Shuffle CWD", theme.FolderIcon(), honeypot.ActionShuffleDirs)

	kick := widget.NewButtonWithIcon("Kick All", theme.LogoutIcon(), func() {
		dialog.NewConfirm(
			"Kick All Sessions",
			"This will disconnect every currently logged-in user. Continue?",
			func(ok bool) {
				if ok {
					honeypot.ActionKickAll()
				}
			},
			w,
		).Show()
	})
	kick.Importance = widget.DangerImportance

	grid := container.NewGridWithColumns(3, knock, notice, join, mtx, conf, shuffle, kick)
	dialog.NewCustom("Broadcast Actions", "Close", grid, w).Show()
}
```

- [ ] **Step 2: Build**

Run: `go build ./...`
Expected: clean exit.

- [ ] **Step 3: Format**

Run: `go fmt ./internal/gui/...`

- [ ] **Step 4: Manual smoke test**

Run: `go run main.go`
- Open the broadcast modal in the GUI.
- Connect via `ssh -p 1337 someone@localhost` (any password).
- Hit "Shuffle CWD" from the GUI; in the SSH session, type `pwd` and confirm a different directory is reported with no `cd <path>` echo printed in the session output.
- Repeat with two concurrent sessions; confirm they can land in different directories.

- [ ] **Step 5: Stop and let user review changes before continuing.**

---

## Task 5: `glitch` package — model + tick + done

**Files:**
- Create: `internal/honeypot/glitch/glitch.go`
- Create: `internal/honeypot/glitch/glitch_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/honeypot/glitch/glitch_test.go`:

```go
package glitch

import (
	"strings"
	"testing"
	"time"
)

func TestModelCompletesAfterDuration(t *testing.T) {
	m := New("hello world\nsecond line\n")
	if m == nil {
		t.Fatal("New returned nil")
	}

	// Drive enough ticks to exceed Duration. Tick interval is TickInterval.
	steps := int(Duration/TickInterval) + 2
	var done bool
	for i := 0; i < steps; i++ {
		_, done = m.Update(Tick{})
		if done {
			break
		}
	}
	if !done {
		t.Fatalf("glitch model did not report done after %d ticks", steps)
	}
}

func TestViewNonEmptyAndDifferentFromBase(t *testing.T) {
	base := strings.Repeat("the quick brown fox jumps over the lazy dog\n", 5)
	m := New(base)
	// Force a few ticks worth of state change.
	for i := 0; i < 3; i++ {
		m.Update(Tick{})
	}
	out := m.View()
	if out == "" {
		t.Fatal("View returned empty string")
	}
}

func TestViewHandlesEmptyBase(t *testing.T) {
	m := New("")
	out := m.View()
	// Just must not panic; empty output is acceptable.
	_ = out
}

func TestStartReturnsTick(t *testing.T) {
	if _, ok := Start().(Tick); !ok {
		t.Fatal("Start() did not return a Tick message")
	}
}

func TestDurationSane(t *testing.T) {
	if Duration < 2*time.Second || Duration > 3*time.Second {
		t.Errorf("Duration %v outside expected 2-3s range", Duration)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/honeypot/glitch/ -v`
Expected: FAIL with `no Go files` or `undefined: New`.

- [ ] **Step 3: Implement the glitch package**

Create `internal/honeypot/glitch/glitch.go`:

```go
package glitch

import (
	"math/rand"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	TickInterval = 50 * time.Millisecond
	Duration     = 2500 * time.Millisecond

	corruptionRate = 0.12 // fraction of non-whitespace runes replaced per frame
	lineShiftRate  = 0.20 // fraction of lines that get a leading whitespace shift
	flickerRate    = 0.05 // fraction of lines wrapped in reversed style
)

var (
	corruptGlyphs = []rune("▓▒░@#%&*")
	flickerStyle  = lipgloss.NewStyle().Reverse(true)
)

type (
	Start struct{} // not used directly; Start() returns a Tick
	Tick  struct{}
)

// New returns a glitch model initialized with the given base screen content.
func New(base string) *Model {
	return &Model{
		base: base,
		rng:  rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Model holds the snapshot of underlying screen content and elapsed time.
type Model struct {
	base    string
	elapsed time.Duration
	rng     *rand.Rand
}

// Update advances the glitch by one tick. Returns the next tick command and a
// done flag that becomes true once the configured Duration is exceeded.
func (m *Model) Update(msg tea.Msg) (tea.Cmd, bool) {
	if _, ok := msg.(Tick); !ok {
		return nil, false
	}
	m.elapsed += TickInterval
	if m.elapsed >= Duration {
		return nil, true
	}
	return tea.Tick(TickInterval, func(time.Time) tea.Msg { return Tick{} }), false
}

// View renders one corrupted frame of the base content.
func (m *Model) View() string {
	if m.base == "" {
		return ""
	}
	lines := strings.Split(m.base, "\n")

	// Line shift: randomly prepend whitespace offset to some lines, or swap
	// with the next line.
	for i := range lines {
		if m.rng.Float64() < lineShiftRate {
			if i+1 < len(lines) && m.rng.Intn(2) == 0 {
				lines[i], lines[i+1] = lines[i+1], lines[i]
			} else {
				offset := strings.Repeat(" ", 1+m.rng.Intn(4))
				lines[i] = offset + lines[i]
			}
		}
	}

	// Char corruption: walk runes, replace a fraction of non-whitespace ones.
	for i, line := range lines {
		runes := []rune(line)
		for j, r := range runes {
			if r == ' ' || r == '\t' {
				continue
			}
			if m.rng.Float64() < corruptionRate {
				runes[j] = corruptGlyphs[m.rng.Intn(len(corruptGlyphs))]
			}
		}
		lines[i] = string(runes)
	}

	// Color flicker: wrap some whole lines in reversed style.
	for i, line := range lines {
		if m.rng.Float64() < flickerRate {
			lines[i] = flickerStyle.Render(line)
		}
	}

	return strings.Join(lines, "\n")
}

// Start returns the initial tick message for the glitch effect.
func Start() tea.Msg {
	return Tick{}
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/honeypot/glitch/ -v`
Expected: PASS.

- [ ] **Step 5: Format**

Run: `go fmt ./internal/honeypot/glitch/...`

- [ ] **Step 6: Stop and let user review changes before continuing.**

---

## Task 6: Wire glitch sub-model into honeypot `model`

**Files:**
- Modify: `internal/honeypot/model.go`
- Modify: `internal/honeypot/pot.go`

- [ ] **Step 1: Add the field and import**

In `internal/honeypot/model.go`, add to the import block:

```go
	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot/glitch"
```

Then in the `model` struct (currently around lines 25–57), add to the "UX & Sub-Models" section after `matrix`:

```go
	matrix        matrix.Matrix
	glitchModel   *glitch.Model
	ctf           ctf.Model
```

(Replace the existing line `matrix        matrix.Matrix` block accordingly.)

- [ ] **Step 2: Initialize in the constructor**

Open `internal/honeypot/pot.go` and locate the model constructor (currently around line 384 where `currentDir: filesystem.HomeDir,` appears). The existing literal already zero-initializes `confetti` and `matrix` implicitly; for `glitchModel` we leave it nil (created lazily on Start), so no constructor change is needed unless the existing literal explicitly lists sub-models. Verify by reading that block; if `confetti` / `matrix` are explicitly listed, add `glitchModel: nil,` for symmetry. Otherwise skip this step.

- [ ] **Step 3: Handle Start + Tick messages**

In `internal/honeypot/model.go`, find the `case filesystem.SetRunningCmd:` line (around line 108). Below the existing `filesystem.OutputMsg` case (just before `filesystem.ListActiveUsersMsg`), add:

```go
		case glitch.Tick:
			if m.runningCommand == "glitch" {
				if m.glitchModel == nil {
					m.glitchModel = glitch.New(m.output)
				}
				cmd, done := m.glitchModel.Update(msg)
				if done {
					m.glitchModel = nil
				} else if cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
```

- [ ] **Step 4: Render glitch in `View()`**

In the `View()` method (around lines 282–300), extend the if/else chain. After the `else if m.runningCommand == "matrix"` block and before `else if m.runningCommand == "ctf"`, insert:

```go
	} else if m.runningCommand == "glitch" {
		if m.glitchModel != nil {
			content = m.glitchModel.View()
		}
		help = ""
```

- [ ] **Step 5: Build**

Run: `go build ./...`
Expected: clean exit.

- [ ] **Step 6: Format + full test suite**

Run: `go fmt ./internal/honeypot/...`
Run: `go test ./...`
Expected: PASS.

- [ ] **Step 7: Stop and let user review changes before continuing.**

---

## Task 7: `ActionGlitch` registry action

**Files:**
- Modify: `internal/honeypot/registry.go`

- [ ] **Step 1: Add the action**

Add the `glitch` import to `internal/honeypot/registry.go`:

```go
	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot/glitch"
```

Append the action at the bottom of the file:

```go
// ActionGlitch broadcasts a short visual glitch effect (~2.5s) to every active
// session, then restores normal rendering. Mirrors the lifecycle of
// ActionConfetti.
func ActionGlitch() {
	broadcast(filesystem.SetRunningCmd("glitch"), glitch.Tick{})
	go func() {
		time.Sleep(glitch.Duration + 100*time.Millisecond)
		broadcast(filesystem.SetRunningCmd(""), filesystem.ClearOutputMsg(""))
	}()
}
```

- [ ] **Step 2: Build**

Run: `go build ./...`
Expected: clean exit.

- [ ] **Step 3: Format**

Run: `go fmt ./internal/honeypot/...`

- [ ] **Step 4: Stop and let user review changes before continuing.**

---

## Task 8: Wire "Glitch" button into GUI + final smoke test

**Files:**
- Modify: `internal/gui/broadcast.go`

- [ ] **Step 1: Add the button**

In `internal/gui/broadcast.go`, in `showBroadcastModal`, add after the `shuffle` definition:

```go
	gltch := widget.NewButtonWithIcon("Glitch", theme.BrokenImageIcon(), honeypot.ActionGlitch)
```

And update the grid line to include it:

```go
	grid := container.NewGridWithColumns(3, knock, notice, join, mtx, conf, shuffle, gltch, kick)
```

- [ ] **Step 2: Build**

Run: `go build ./...`
Expected: clean exit.

- [ ] **Step 3: Format**

Run: `go fmt ./internal/gui/...`

- [ ] **Step 4: Manual smoke test (both features end-to-end)**

Run: `go run main.go`
- SSH in: `ssh -p 1337 anyone@localhost`, type `ls`, type `pwd` so the session has visible output.
- From the GUI broadcast modal:
  - Hit **Shuffle CWD**, verify next `pwd` shows a different path with no echoed `cd …` line.
  - Hit **Glitch**, verify the session screen visibly corrupts (char garbage, shifted lines, occasional reversed lines) for ~2.5s then returns to a normal prompt.
- Open a second SSH session, fire Shuffle CWD again, and verify the two sessions can land in different directories.

- [ ] **Step 5: Full test suite**

Run: `go test ./...`
Expected: PASS.

- [ ] **Step 6: Stop and let user review final state.**

---

## Self-Review Checklist (verified during plan authoring)

- Spec coverage:
  - Shuffle CWD button → Task 4.
  - `AllDirectories` helper → Task 1.
  - `ChangeDirMsg.Silent` + model handler honoring it → Task 2.
  - `ActionShuffleDirs` (per-session random destination) → Task 3.
  - Glitch button → Task 8.
  - `internal/honeypot/glitch` package with char corruption + line shift + color flicker, 2.5s duration → Task 5.
  - Model wiring (field, Tick handler, View case) → Task 6.
  - `ActionGlitch` with auto-cleanup goroutine mirroring `ActionConfetti` → Task 7.
- Placeholders: none.
- Type/name consistency:
  - `glitch.New`, `glitch.Tick`, `glitch.Duration`, `glitch.TickInterval` used identically in package + consumers.
  - `filesystem.AllDirectories`, `filesystem.ChangeDirMsg.Silent`, `honeypot.ActionShuffleDirs`, `honeypot.ActionGlitch` all spelled consistently across plan tasks.
