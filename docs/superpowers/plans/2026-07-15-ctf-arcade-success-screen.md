# CTF Arcade Success Screen Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the CTF correct-flag feedback (one-line message + full-screen confetti) with a dedicated animated arcade "LEVEL COMPLETE" success screen inside the CTF model.

**Architecture:** Add a `stateSuccess` screen to the CTF Bubble Tea model. On a correct flag we complete the task, snapshot the display data, transition to `stateSuccess`, and drive animation with an ~80ms self-terminating `tea.Tick`. Any keypress returns to the menu. Pure helpers (`successBanner`, `tallyValue`, `dottedLeader`) are unit-tested; the screen composition and wiring are verified by building + a manual smoke test.

**Tech Stack:** Go 1.24+, Charmbracelet Bubble Tea v2 / lipgloss v2, standard `testing`.

## Global Constraints

- Standard Go formatting (`go fmt ./...`).
- CGO required to build/run the full app; `go test ./internal/honeypot/ctf/` runs without a display.
- Never commit to git — the user handles all version control. "Leave changes for the user" replaces every commit step; do NOT run `git commit`.
- All changes live in `internal/honeypot/ctf/ctf.go` and `internal/honeypot/ctf/ctf_test.go`. No `model.go`, `pot.go`, or `config.go` changes.
- The confetti model and `celebrate` command must remain functional — only the CTF success path stops using `ConfettiMsg`. The `ConfettiMsg` type stays defined/exported.
- Palette (8-bit), used for the color-cycling headline: `9` red, `11` yellow, `10` green, `14` cyan, `13` magenta, `12` blue.

---

### Task 1: Pure helpers — replace `successMessage` with `successBanner`, add `tallyValue` and `dottedLeader`

**Files:**
- Modify: `internal/honeypot/ctf/ctf.go` (the `successMessage` helper, currently just above `updateAnswer`)
- Test: `internal/honeypot/ctf/ctf_test.go` (replace `TestSuccessMessage`)

**Interfaces:**
- Consumes: nothing new.
- Produces:
  - `func successBanner(custom string) string`
  - `func tallyValue(start, end, frame, duration int) int`
  - `func dottedLeader(label, value string, width int) string`

- [ ] **Step 1: Replace the test**

In `internal/honeypot/ctf/ctf_test.go`, delete the entire `TestSuccessMessage` function and add:

```go
func TestSuccessBanner(t *testing.T) {
	if got := successBanner(""); got != "Challenge Complete!" {
		t.Fatalf("empty: got %q", got)
	}
	if got := successBanner("   "); got != "Challenge Complete!" {
		t.Fatalf("whitespace: got %q", got)
	}
	if got := successBanner("  The vault is open.  "); got != "The vault is open." {
		t.Fatalf("custom: got %q", got)
	}
}

func TestTallyValue(t *testing.T) {
	if got := tallyValue(0, 50, 0, 10); got != 0 {
		t.Fatalf("frame 0: got %d", got)
	}
	if got := tallyValue(0, 50, -3, 10); got != 0 {
		t.Fatalf("negative frame: got %d", got)
	}
	if got := tallyValue(0, 50, 5, 10); got != 25 {
		t.Fatalf("midpoint: got %d", got)
	}
	if got := tallyValue(0, 50, 10, 10); got != 50 {
		t.Fatalf("end frame: got %d", got)
	}
	if got := tallyValue(0, 50, 99, 10); got != 50 {
		t.Fatalf("past end: got %d", got)
	}
	if got := tallyValue(0, 50, 5, 0); got != 50 {
		t.Fatalf("zero duration: got %d", got)
	}
	if got := tallyValue(100, 150, 5, 10); got != 125 {
		t.Fatalf("nonzero start midpoint: got %d", got)
	}
}

func TestDottedLeader(t *testing.T) {
	got := dottedLeader("BONUS", "+50", 32)
	if len([]rune(got)) != 32 {
		t.Fatalf("width: got len %d (%q)", len([]rune(got)), got)
	}
	if !strings.HasPrefix(got, "BONUS ") || !strings.HasSuffix(got, " +50") {
		t.Fatalf("format: got %q", got)
	}
	if !strings.Contains(got, "..") {
		t.Fatalf("expected dots: got %q", got)
	}
	// Cramped width still yields at least one dot and does not panic.
	cramped := dottedLeader("BONUS", "+50", 3)
	if !strings.Contains(cramped, ".") {
		t.Fatalf("cramped: expected a dot, got %q", cramped)
	}
}
```

Add `"strings"` to the test file's imports if not present. Current test file imports only `"testing"`, so change the import to:

```go
import (
	"strings"
	"testing"
)
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/honeypot/ctf/ -run 'TestSuccessBanner|TestTallyValue|TestDottedLeader'`
Expected: FAIL — `successBanner`/`tallyValue`/`dottedLeader` undefined (and `successMessage` no longer referenced).

- [ ] **Step 3: Replace the helper**

In `internal/honeypot/ctf/ctf.go`, delete the `successMessage` function and add in its place (just above `updateAnswer`):

```go
// successBanner returns the headline shown on the success screen: the custom
// message when set, otherwise a default.
func successBanner(custom string) string {
	if strings.TrimSpace(custom) == "" {
		return "Challenge Complete!"
	}
	return strings.TrimSpace(custom)
}

// tallyValue returns the animated count-up value at the given frame, ramping
// linearly from start to end over duration frames, then holding at end.
func tallyValue(start, end, frame, duration int) int {
	if duration <= 0 || frame >= duration {
		return end
	}
	if frame <= 0 {
		return start
	}
	return start + (end-start)*frame/duration
}

// dottedLeader renders "LABEL ....... value" padded to width with a dotted run.
func dottedLeader(label, value string, width int) string {
	dots := width - (len(label) + len(value) + 2)
	if dots < 1 {
		dots = 1
	}
	return fmt.Sprintf("%s %s %s", label, strings.Repeat(".", dots), value)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/honeypot/ctf/ -run 'TestSuccessBanner|TestTallyValue|TestDottedLeader'`
Expected: PASS.

Note: the package will NOT fully build yet if anything still references `successMessage` — it does not, because Task 2 removes the call site. But `updateAnswer` currently calls `successMessage`. To keep the tree compiling between tasks, do Step 5 now.

- [ ] **Step 5: Temporarily keep the call site compiling**

The current `updateAnswer` still calls `successMessage(...)`. Since Task 2 rewrites that block, update the single line now so the package compiles. In `updateAnswer`, replace:

```go
					successMessage(m.selectedTask.SuccessMessage, m.selectedTask.Points),
```

with:

```go
					fmt.Sprintf("%s +%d points!", successBanner(m.selectedTask.SuccessMessage), m.selectedTask.Points),
```

- [ ] **Step 6: Build + format**

Run: `go build ./internal/honeypot/ctf/ && go fmt ./internal/honeypot/ctf/`
Expected: builds clean.
Leave changes for the user (do not `git commit`).

---

### Task 2: `stateSuccess` — state, model fields, tick, transition, view

**Files:**
- Modify: `internal/honeypot/ctf/ctf.go` (enum, `Model`, imports, `Update`, `updateAnswer`, `View`; add `updateSuccess`, `renderSuccess`, `successTick`)

**Interfaces:**
- Consumes: `successBanner`, `tallyValue`, `dottedLeader` (Task 1); `m.user.Points` (updated by `CompleteTask`).
- Produces: `stateSuccess`, `successTickMsg`, `successTick()`, `updateSuccess`, `renderSuccess`, and `Model` fields `successFrame`, `successBanner`, `successBonus`, `successOldTotal`, `successNewTotal`.

- [ ] **Step 1: Add the state to the enum**

In `internal/honeypot/ctf/ctf.go`, extend the `gameState` const block:

```go
const (
	stateLogin gameState = iota
	stateMenu
	stateAnswer
	stateLeaderboard
	stateSuccess
	stateDone
)
```

- [ ] **Step 2: Add Model fields**

In the `Model` struct, add after `errMsg string`:

```go
	// success-screen animation snapshot
	successFrame    int
	successMsg      string
	successBonus    int
	successOldTotal int
	successNewTotal int
```

(Field is named `successMsg` — not `successBanner` — to avoid colliding with the `successBanner` function.)

- [ ] **Step 3: Add the tick message and command**

Add near the other message types (e.g. after `leaderboardLoadedMsg`/`loadLeaderboardCmd`):

```go
// successTickMsg drives the arcade success-screen animation.
type successTickMsg struct{}

func successTick() tea.Cmd {
	return tea.Tick(80*time.Millisecond, func(time.Time) tea.Msg {
		return successTickMsg{}
	})
}
```

- [ ] **Step 4: Handle the tick in top-level `Update`**

In `Update`, add a case to the first `switch msg := msg.(type)` block (alongside `leaderboardLoadedMsg`):

```go
	case successTickMsg:
		if m.state == stateSuccess {
			m.successFrame++
			return m, successTick()
		}
		return m, nil
```

- [ ] **Step 5: Route `stateSuccess` in the state switch**

In `Update`, add to the `switch m.state` block:

```go
	case stateSuccess:
		return m.updateSuccess(msg)
```

- [ ] **Step 6: Add `updateSuccess`**

Add the handler (e.g. after `updateAnswer`):

```go
func (m Model) updateSuccess(msg tea.Msg) (Model, tea.Cmd) {
	if _, ok := msg.(tea.KeyPressMsg); ok {
		m.state = stateMenu
		m.errMsg = ""
		return m, nil
	}
	return m, nil
}
```

- [ ] **Step 7: Rewrite the correct-flag branch in `updateAnswer`**

Replace the entire `if ans == m.selectedTask.Flag { ... }` block (the success path — from `var celebrate tea.Cmd` through `m.state = stateMenu` / `return m, celebrate`) with:

```go
			if ans == m.selectedTask.Flag {
				var next tea.Cmd
				if err := m.user.CompleteTask(m.selectedTask.Name, m.selectedTask.Points); err != nil {
					m.errMsg = err.Error()
					m.state = stateMenu
				} else {
					m.selectedTask.Completed = true
					publishLeaderboard("solve")

					// Create a GUI notification for task completion
					go func(taskName string, points int) {
						event := &entity.Event{
							User:      m.sshuser,
							Host:      m.sshhost,
							App:       "ctf",
							Source:    entity.EventSourceUser,
							Type:      "taskCompleted",
							Action:    fmt.Sprintf("Completed CTF task: %s (+%d pts)", taskName, points),
							Timestamp: time.Now(),
						}
						event.Publish()
						event.Save()
					}(m.selectedTask.Name, m.selectedTask.Points)

					// Recompute partition so activeTasks reflects the newly-completed task.
					m.activeTasks, m.archivedDone = partitionTasks(m.tasks)
					if m.cursor >= len(m.activeTasks) {
						m.cursor = 0
					}

					// Snapshot data for the arcade success screen.
					m.successMsg = successBanner(m.selectedTask.SuccessMessage)
					m.successBonus = m.selectedTask.Points
					m.successNewTotal = m.user.Points
					m.successOldTotal = m.user.Points - m.selectedTask.Points
					m.successFrame = 0
					m.errMsg = ""
					m.state = stateSuccess
					next = successTick()
				}
				return m, next
			}
```

- [ ] **Step 8: Remove the now-unused confetti import**

The `celebrate` batch used `confetti.Burst()`; it is gone. Remove the import line
`"github.com/mikeflynn/honeybearhoneypot/internal/honeypot/confetti"` from the
`import (...)` block in `ctf.go`. (The `ConfettiMsg` type defined in this file stays — it does not depend on the package.)

- [ ] **Step 9: Add the view routing and `renderSuccess`**

In `View`, add to the `switch m.state`:

```go
	case stateSuccess:
		return m.renderSuccess()
```

Add the renderer:

```go
func (m Model) renderSuccess() string {
	palette := []lipgloss.Color{
		lipgloss.Color("9"), lipgloss.Color("11"), lipgloss.Color("10"),
		lipgloss.Color("14"), lipgloss.Color("13"), lipgloss.Color("12"),
	}
	const tallyFrames = 12
	const blinkFrames = 6

	titleStyle := lipgloss.NewStyle().Foreground(palette[(m.successFrame/2)%len(palette)]).Bold(true)
	msgStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Bold(true)
	scoreStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("7"))

	bonus := tallyValue(0, m.successBonus, m.successFrame, tallyFrames)
	score := tallyValue(m.successOldTotal, m.successNewTotal, m.successFrame, tallyFrames)

	const leaderWidth = 30
	bonusLine := dottedLeader("BONUS", fmt.Sprintf("+%d", bonus), leaderWidth)
	scoreLine := dottedLeader("SCORE", fmt.Sprintf("%d", score), leaderWidth)

	// Wrap the message; only decorate with >>> <<< when it stays on one line.
	maxMsg := 44
	if m.width > 0 && m.width-12 < maxMsg {
		maxMsg = m.width - 12
	}
	if maxMsg < 10 {
		maxMsg = 10
	}
	wrapped := wordWrap(m.successMsg, maxMsg)
	var msgLine string
	if strings.Contains(wrapped, "\n") {
		msgLine = msgStyle.Render(wrapped)
	} else {
		msgLine = msgStyle.Render(">>>  " + wrapped + "  <<<")
	}

	prompt := " "
	if (m.successFrame/blinkFrames)%2 == 0 {
		prompt = "press any key to continue"
	}

	body := lipgloss.JoinVertical(lipgloss.Center,
		titleStyle.Render("·  ·  ✦  LEVEL COMPLETE  ✦  ·  ·"),
		"",
		"🏆   🍯   🏆",
		"",
		msgLine,
		"",
		scoreStyle.Render(bonusLine),
		scoreStyle.Render(scoreLine),
		"",
		dimStyle.Render(prompt),
	)

	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 4)
	inner := box.Render(body)
	if m.width > 0 {
		return lipgloss.PlaceHorizontal(m.width, lipgloss.Center, inner)
	}
	return inner
}
```

- [ ] **Step 10: Build the whole app to confirm it compiles (CGO)**

Run: `go build ./...`
Expected: no output (success).

- [ ] **Step 11: Run the CTF package tests**

Run: `go test ./internal/honeypot/ctf/`
Expected: PASS (all tests, including Task 1's helpers).

- [ ] **Step 12: Format**

Run: `go fmt ./internal/honeypot/ctf/`
Leave changes for the user (do not `git commit`).

---

## Final Verification

- [ ] Full suite: `go test ./...` → PASS (CGO dev environment). The `-lobjc` linker warning from the Fyne/gui package is benign.
- [ ] `gofmt -l internal/ main.go` prints nothing.
- [ ] **Manual smoke test (UI path not covered by unit tests):**
  Run `go run main.go -no-gui -ssh-port 2222` (uses `config.sample.json`? no — pass `-config config.sample.json` to get the demo task with a `success_message`). Then:
  ```bash
  go run main.go -no-gui -ssh-port 2222 -config config.sample.json
  ```
  In another terminal: `ssh -p 2222 test@localhost`, run `ctf`, register a user, select the `demo` task, submit `flag{demo}`. Confirm:
  - The arcade "LEVEL COMPLETE" screen appears (no confetti takeover).
  - Headline color cycles; BONUS counts up to `+100`; SCORE rolls to the new total.
  - `press any key to continue` blinks.
  - Any key returns to the menu with `demo` marked complete.
  - Separately, run `celebrate` at the shell to confirm confetti still works.

## Self-Review Notes

- **Spec coverage:** state + fields (T2 S1–S2), tick (T2 S3–S4), routing (T2 S5), transition (T2 S7), confetti detachment (T2 S8), view (T2 S9), helpers + tests (T1). All spec sections covered.
- **Type consistency:** the `Model` field is `successMsg` (function `successBanner` returns into it); `successBonus`/`successOldTotal`/`successNewTotal`/`successFrame` used identically in `updateAnswer` and `renderSuccess`. `successTick()`/`successTickMsg` names match between definition, `Update` handler, and transition.
- **Naming collision avoided:** function `successBanner` vs field `successMsg` — deliberately different.
- **Placeholders:** none — every code step is concrete.
- **Git:** all commits replaced with "leave changes for the user."
```