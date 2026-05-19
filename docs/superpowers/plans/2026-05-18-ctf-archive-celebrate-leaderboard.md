# CTF Archive, Celebrate, Leaderboard Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add three features to the CTF subsystem — config-driven task archiving (visible to prior solvers only), confetti burst on correct flag submission, and a leaderboard view bound to `L` in the menu.

**Architecture:** Three changes localized to `internal/config/config.go`, `internal/honeypot/pot.go`, and `internal/honeypot/ctf/ctf.go`. The CTF Bubble Tea sub-model gains a derived `activeTasks`/`archivedDone` partition and a `stateLeaderboard` state. Confetti reuses the existing top-level overlay in the parent honeypot model by dispatching the same `SetRunningCmd("confetti")` + `confetti.Burst()` batch that the `celebrate` filesystem command uses.

**Tech Stack:** Go 1.24, Bubble Tea v2 (`charm.land/bubbletea/v2`), Lipgloss v2, SQLite (existing `entity.Leaderboard` reused). No new dependencies.

**Project rule (from CLAUDE.md):** Never commit. The user handles all git operations. So `git commit` steps below are replaced by "Pause for user to commit." This plan contains those pauses instead of commit commands.

**Testing reality:** This codebase has no tests for the `ctf` package (none exist today; verify with `ls internal/honeypot/ctf/`). The CTF UI is a Bubble Tea TUI tightly coupled to terminal I/O, and the existing pattern in this repo is to verify TUI behavior manually. So:

- For pure-data changes (config field plumbing) we add a small Go test.
- For TUI rendering / state changes we use **focused manual verification** with explicit reproduction steps and expected output. Each task lists the exact commands and what to look for.
- `go build ./...` and `go test ./...` must pass after every task.

---

## File Structure

| File | Change | Responsibility |
|---|---|---|
| `internal/config/config.go` | Modify (line ~12) | Add `Archived bool` to `Task` |
| `internal/honeypot/pot.go` | Modify (line ~421) | Propagate `Archived` through `convertTasks` |
| `internal/honeypot/ctf/ctf.go` | Modify | Add `Archived` field to `ctf.Task`; partition into active/archived-done; add `stateLeaderboard`; dispatch confetti on correct flag |
| `internal/config/config_test.go` | Create (or extend if exists) | Verify `Archived` JSON parses |
| `internal/honeypot/ctf/ctf_test.go` | Create | Unit-test the task partition helper |

---

## Task 1: Add `Archived` field to config Task

**Files:**
- Modify: `internal/config/config.go:12-17`
- Test: `internal/config/config_test.go` (create if missing)

- [ ] **Step 1: Check whether `config_test.go` already exists**

Run: `ls internal/config/`
Expected: see `config.go`. If `config_test.go` exists, you'll extend it; if not, create it in step 3.

- [ ] **Step 2: Write the failing test**

If `internal/config/config_test.go` does not exist, create it with the following content. If it exists, append the `TestTaskArchivedJSON` function and add `"encoding/json"` and `"testing"` to its imports (skip duplicates).

```go
package config

import (
	"encoding/json"
	"testing"
)

func TestTaskArchivedJSON(t *testing.T) {
	t.Run("archived true round-trips", func(t *testing.T) {
		in := Task{Name: "x", Flag: "f", Points: 1, Archived: true}
		b, err := json.Marshal(in)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		var out Task
		if err := json.Unmarshal(b, &out); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if !out.Archived {
			t.Fatalf("expected Archived=true, got false; json=%s", string(b))
		}
	})

	t.Run("archived omitted defaults false", func(t *testing.T) {
		var out Task
		if err := json.Unmarshal([]byte(`{"name":"x","flag":"f","points":1}`), &out); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if out.Archived {
			t.Fatalf("expected Archived=false when omitted, got true")
		}
	})

	t.Run("archived false omitted from output", func(t *testing.T) {
		b, err := json.Marshal(Task{Name: "x", Flag: "f", Points: 1})
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if contains(string(b), "archived") {
			t.Fatalf("expected omitempty to drop archived field, got %s", string(b))
		}
	})
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
```

- [ ] **Step 3: Run the test to verify it fails**

Run: `go test ./internal/config/ -run TestTaskArchivedJSON -v`
Expected: build failure — `out.Archived undefined (type Task has no field or method Archived)`.

- [ ] **Step 4: Add the `Archived` field to `Task`**

Edit `internal/config/config.go` lines 12-17. Change:

```go
type Task struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Flag        string `json:"flag"`
	Points      int    `json:"points"`
}
```

to:

```go
type Task struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Flag        string `json:"flag"`
	Points      int    `json:"points"`
	Archived    bool   `json:"archived,omitempty"`
}
```

- [ ] **Step 5: Run the test to verify it passes**

Run: `go test ./internal/config/ -run TestTaskArchivedJSON -v`
Expected: PASS, 3 subtests.

- [ ] **Step 6: Run the full test suite and build**

Run: `go test ./... && go build ./...`
Expected: PASS, clean build.

- [ ] **Step 7: Pause for user to commit**

Stop and tell the user: "Task 1 complete — `Archived` field added and tested. Please commit when ready, then I'll continue."

---

## Task 2: Propagate `Archived` through `convertTasks`

**Files:**
- Modify: `internal/honeypot/pot.go:421-432`

This task has no isolated test — `convertTasks` is exercised by the CTF model tests in later tasks. Verification is via build.

- [ ] **Step 1: Edit `convertTasks`**

Edit `internal/honeypot/pot.go` lines 421-432. Change:

```go
func convertTasks(t []config.Task) []ctf.Task {
	out := make([]ctf.Task, len(t))
	for i, task := range t {
		out[i] = ctf.Task{
			Name:        task.Name,
			Description: task.Description,
			Flag:        task.Flag,
			Points:      task.Points,
		}
	}
	return out
}
```

to:

```go
func convertTasks(t []config.Task) []ctf.Task {
	out := make([]ctf.Task, len(t))
	for i, task := range t {
		out[i] = ctf.Task{
			Name:        task.Name,
			Description: task.Description,
			Flag:        task.Flag,
			Points:      task.Points,
			Archived:    task.Archived,
		}
	}
	return out
}
```

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: build failure — `unknown field Archived in struct literal of type ctf.Task`. This is expected; Task 3 adds the field to `ctf.Task`. Leave this change in place and proceed.

- [ ] **Step 3: Note status**

Tell the user: "Task 2 partial — `pot.go` updated; build will fail until Task 3 adds the field to `ctf.Task`. Continuing without commit."

---

## Task 3: Add `Archived` field and partition helper to ctf.Task

**Files:**
- Modify: `internal/honeypot/ctf/ctf.go:39-45` (Task struct), `47-69` (Model struct), `71-89` (loadCompleted)
- Test: `internal/honeypot/ctf/ctf_test.go` (create)

- [ ] **Step 1: Write the failing test**

Create `internal/honeypot/ctf/ctf_test.go` with:

```go
package ctf

import "testing"

func TestPartitionTasks(t *testing.T) {
	tasks := []Task{
		{Name: "a", Points: 1, Archived: false, Completed: false},
		{Name: "b", Points: 2, Archived: false, Completed: true},
		{Name: "c", Points: 3, Archived: true, Completed: true},
		{Name: "d", Points: 4, Archived: true, Completed: false},
	}

	active, archivedDone := partitionTasks(tasks)

	if len(active) != 2 || active[0].Name != "a" || active[1].Name != "b" {
		t.Fatalf("active mismatch: %+v", active)
	}
	if len(archivedDone) != 1 || archivedDone[0].Name != "c" {
		t.Fatalf("archivedDone mismatch: %+v", archivedDone)
	}
	// Task d (archived + not completed) must be excluded entirely.
}

func TestPartitionTasksEmpty(t *testing.T) {
	active, archived := partitionTasks(nil)
	if len(active) != 0 || len(archived) != 0 {
		t.Fatalf("expected empty slices, got %d active, %d archived", len(active), len(archived))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/honeypot/ctf/ -run TestPartitionTasks -v`
Expected: build failure — `Archived undefined` and `partitionTasks undefined`.

- [ ] **Step 3: Add `Archived` field to `Task`**

Edit `internal/honeypot/ctf/ctf.go` lines 39-45. Change:

```go
type Task struct {
	Name        string
	Description string
	Flag        string
	Points      int
	Completed   bool
}
```

to:

```go
type Task struct {
	Name        string
	Description string
	Flag        string
	Points      int
	Completed   bool
	Archived    bool
}
```

- [ ] **Step 4: Add the `partitionTasks` helper**

Add this function to `internal/honeypot/ctf/ctf.go`, immediately above `func (m *Model) loadCompleted()` (around line 71):

```go
// partitionTasks splits tasks into (active, archivedDone).
// Active = not archived (regardless of completion state).
// archivedDone = archived AND completed by the current user.
// Archived-and-not-completed tasks are excluded entirely.
func partitionTasks(tasks []Task) (active, archivedDone []Task) {
	for _, t := range tasks {
		switch {
		case !t.Archived:
			active = append(active, t)
		case t.Archived && t.Completed:
			archivedDone = append(archivedDone, t)
		}
	}
	return active, archivedDone
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/honeypot/ctf/ -run TestPartitionTasks -v`
Expected: PASS, 2 subtests.

- [ ] **Step 6: Verify full build**

Run: `go build ./...`
Expected: clean build (Task 2's change now compiles).

- [ ] **Step 7: Run full test suite**

Run: `go test ./...`
Expected: PASS.

- [ ] **Step 8: Pause for user to commit**

Tell the user: "Tasks 2+3 complete — `Archived` plumbed through and `partitionTasks` tested. Please commit when ready."

---

## Task 4: Cache partition results on Model and update navigation/render

**Files:**
- Modify: `internal/honeypot/ctf/ctf.go` — `Model` struct (47-69), `loadCompleted` (71-89), `updateMenu` (261-289), `renderTasks` (376-417)

This is the largest single task because cursor bounds, render, and state must change atomically to keep behavior consistent.

- [ ] **Step 1: Add cached partition fields to Model**

Edit `internal/honeypot/ctf/ctf.go` lines 47-69. Inside the `Model` struct, add these two fields just after the `tasks []Task` line:

```go
	tasks         []Task
	activeTasks   []Task
	archivedDone  []Task
```

So the relevant block becomes:

```go
type Model struct {
	state    gameState
	username string
	password string
	user     *entity.CTFUser

	sshuser string
	sshhost string

	width  int
	height int

	usernameInput textinput.Model
	passwordInput textinput.Model
	answerInput   textinput.Model

	tasks        []Task
	activeTasks  []Task
	archivedDone []Task
	cursor       int

	selectedTask *Task

	errMsg string
}
```

(Note: remove the existing `cursor int` line in its original position to avoid duplication.)

- [ ] **Step 2: Recompute partition inside `loadCompleted`**

Edit `internal/honeypot/ctf/ctf.go` lines 71-89. Change:

```go
func (m *Model) loadCompleted() {
	if m.user == nil {
		return
	}
	tasks, err := m.user.CompletedTasks()
	if err != nil {
		m.errMsg = err.Error()
		return
	}
	done := map[string]struct{}{}
	for _, t := range tasks {
		done[t] = struct{}{}
	}
	for i := range m.tasks {
		if _, ok := done[m.tasks[i].Name]; ok {
			m.tasks[i].Completed = true
		}
	}
}
```

to:

```go
func (m *Model) loadCompleted() {
	if m.user == nil {
		return
	}
	tasks, err := m.user.CompletedTasks()
	if err != nil {
		m.errMsg = err.Error()
		return
	}
	done := map[string]struct{}{}
	for _, t := range tasks {
		done[t] = struct{}{}
	}
	for i := range m.tasks {
		if _, ok := done[m.tasks[i].Name]; ok {
			m.tasks[i].Completed = true
		}
	}
	m.activeTasks, m.archivedDone = partitionTasks(m.tasks)
	if m.cursor >= len(m.activeTasks) {
		m.cursor = 0
	}
}
```

- [ ] **Step 3: Update cursor bounds and selection in `updateMenu`**

Edit `internal/honeypot/ctf/ctf.go` lines 261-289. Change:

```go
func (m Model) updateMenu(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.tasks)-1 {
				m.cursor++
			}
		case "enter":
			if len(m.tasks) > 0 {
				if m.tasks[m.cursor].Completed {
					return m, nil
				}
				m.selectedTask = &m.tasks[m.cursor]
				m.state = stateAnswer
				m.answerInput.SetValue("")
				m.answerInput.Focus()
			}
		case "q", "ctrl+c":
			m.state = stateDone
			return m, func() tea.Msg { return QuitMsg{} }
		}
	}
	return m, nil
}
```

to:

```go
func (m Model) updateMenu(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.activeTasks)-1 {
				m.cursor++
			}
		case "enter":
			if len(m.activeTasks) > 0 {
				if m.activeTasks[m.cursor].Completed {
					return m, nil
				}
				// Find the master-list pointer so CompleteTask updates the right element.
				for i := range m.tasks {
					if m.tasks[i].Name == m.activeTasks[m.cursor].Name {
						m.selectedTask = &m.tasks[i]
						break
					}
				}
				m.state = stateAnswer
				m.answerInput.SetValue("")
				m.answerInput.Focus()
			}
		case "q", "ctrl+c":
			m.state = stateDone
			return m, func() tea.Msg { return QuitMsg{} }
		}
	}
	return m, nil
}
```

- [ ] **Step 4: Update `renderTasks` to render active list + archived section**

Edit `internal/honeypot/ctf/ctf.go` lines 376-417 (`renderTasks`). Replace the entire function with:

```go
func (m Model) renderTasks(showAllDesc bool) string {
	var b strings.Builder
	bullet := lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Render("🍯")
	doneBullet := lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render("✓")
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("0")).Background(lipgloss.Color("6")).Bold(true)
	normalStyle := lipgloss.NewStyle()
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	archivedHeader := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("── Archived ──")

	renderOne := func(t Task, i int, selectable bool) {
		blet := bullet
		style := normalStyle
		if t.Completed {
			blet = doneBullet
			style = normalStyle.Foreground(lipgloss.Color("8"))
		}

		marker := " "
		if selectable && m.state == stateMenu && i == m.cursor {
			marker = ">"
		}

		line := fmt.Sprintf("%s %s %s (%d pts)", marker, blet, t.Name, t.Points)
		if selectable && m.state == stateMenu && i == m.cursor {
			line = selectedStyle.Render(line)
		} else {
			line = style.Render(line)
		}
		b.WriteString(line + "\n")
		desc := t.Description
		if !showAllDesc {
			r := []rune(desc)
			limit := 60
			if len(r) > limit {
				desc = string(r[:limit-3]) + "..."
			}
		}
		b.WriteString("  " + descStyle.Render(desc) + "\n")
	}

	if len(m.activeTasks) == 0 && len(m.archivedDone) == 0 {
		return descStyle.Render("No tasks available.")
	}

	for i, t := range m.activeTasks {
		renderOne(t, i, true)
	}

	if len(m.activeTasks) == 0 {
		b.WriteString(descStyle.Render("All active tasks complete!") + "\n")
	}

	if len(m.archivedDone) > 0 {
		b.WriteString("\n" + archivedHeader + "\n")
		for i, t := range m.archivedDone {
			renderOne(t, i, false)
		}
	}

	return strings.TrimRight(b.String(), "\n")
}
```

- [ ] **Step 5: Build and run existing tests**

Run: `go build ./... && go test ./...`
Expected: PASS.

- [ ] **Step 6: Manual smoke test**

Run: `go run main.go -no-gui -ssh-port 2222 -log-level info -config config.sample.json`

In another terminal: `ssh -p 2222 test@127.0.0.1` (password: anything).

Then in the SSH session:
1. Type `ctf`.
2. Register a user (any name/password).
3. Verify the task menu renders. Up/down moves cursor only across active tasks.

Now stop the server, edit `config.sample.json` (or whichever config you used) and add `"archived": true` to one task that you completed. Restart, ssh back in, login to ctf with the same username.

Expected:
- The archived task does **not** appear in the active list.
- A `── Archived ──` header appears below the active tasks.
- Your completed-archived task appears under it with a `✓` bullet, dimmed, not selectable.

If no tasks exist in your config, copy a couple from `config.sample.json`'s `tasks` block first. If `config.sample.json` has no tasks, create a temporary `test-ctf.json` with:

```json
{
  "ssh_ports": ["2222"],
  "no_gui": true,
  "tasks": [
    {"name": "Task A", "description": "First task", "flag": "flagA", "points": 10},
    {"name": "Task B", "description": "Second task", "flag": "flagB", "points": 20},
    {"name": "Task C (archived)", "description": "Old task", "flag": "flagC", "points": 30, "archived": true}
  ]
}
```

Run with `go run main.go -config test-ctf.json`. Solve Task C *before* marking it archived (i.e., remove `"archived": true`, solve, then add it back and restart).

- [ ] **Step 7: Pause for user to commit**

Tell the user: "Task 4 complete — archive partition + render landed. Manual test instructions in the plan. Please commit when satisfied."

---

## Task 5: Fire confetti on correct flag submission

**Files:**
- Modify: `internal/honeypot/ctf/ctf.go` — imports (1-14), `updateAnswer` correct-flag branch (291-336)

- [ ] **Step 1: Add the new imports**

Edit `internal/honeypot/ctf/ctf.go` import block (lines 1-14). Add the two new imports:

```go
package ctf

import (
	"fmt"
	"strings"
	"time"

	"unicode"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/mikeflynn/honeybearhoneypot/internal/entity"
	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot/confetti"
	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot/filesystem"
)
```

- [ ] **Step 2: Update the correct-flag branch to return a tea.Batch**

Edit `internal/honeypot/ctf/ctf.go` lines 291-336 (`updateAnswer`). Change the correct-flag branch:

```go
			if ans == m.selectedTask.Flag {
				if err := m.user.CompleteTask(m.selectedTask.Name, m.selectedTask.Points); err != nil {
					m.errMsg = err.Error()
				} else {
					m.errMsg = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true).Render(
						fmt.Sprintf("🎉 Correct! +%d points! 🎉", m.selectedTask.Points),
					)
					m.selectedTask.Completed = true

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
				}
				m.state = stateMenu
				return m, nil
			}
```

to:

```go
			if ans == m.selectedTask.Flag {
				var celebrate tea.Cmd
				if err := m.user.CompleteTask(m.selectedTask.Name, m.selectedTask.Points); err != nil {
					m.errMsg = err.Error()
				} else {
					m.errMsg = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true).Render(
						fmt.Sprintf("🎉 Correct! +%d points! 🎉", m.selectedTask.Points),
					)
					m.selectedTask.Completed = true
					// Recompute partition so the just-completed task moves to the right slice.
					m.activeTasks, m.archivedDone = partitionTasks(m.tasks)
					if m.cursor >= len(m.activeTasks) && m.cursor > 0 {
						m.cursor = len(m.activeTasks) - 1
						if m.cursor < 0 {
							m.cursor = 0
						}
					}

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

					celebrate = tea.Batch(
						func() tea.Msg { return filesystem.SetRunningCmd("confetti") },
						func() tea.Msg { return confetti.Burst() },
						tea.Tick(4*time.Second, func(time.Time) tea.Msg {
							return filesystem.SetRunningCmd("")
						}),
					)
				}
				m.state = stateMenu
				return m, celebrate
			}
```

Note: `celebrate` is `nil` if `CompleteTask` errored, so `return m, celebrate` is equivalent to `return m, nil` in the error path.

- [ ] **Step 3: Build and run tests**

Run: `go build ./... && go test ./...`
Expected: PASS.

- [ ] **Step 4: Manual smoke test**

Start the honeypot with a test config that has at least one solvable task (see Task 4 step 6 instructions). SSH in, `ctf`, register, select a task, submit the correct flag.

Expected:
- Confetti animation overlays the terminal for ~4 seconds.
- After confetti clears, the CTF menu is visible again with the green "🎉 Correct! +N points! 🎉" message above/below the task list.
- The solved task now shows the `✓` bullet and is dimmed.
- Submitting a wrong flag still shows "Incorrect flag" with no confetti.

If confetti does not appear, verify by running the standalone `celebrate` command from the shell (`celebrate` at the fake `$` prompt) — if that works but the CTF path doesn't, recheck the `tea.Batch` and that `SetRunningCmd` is the `filesystem` package type.

- [ ] **Step 5: Pause for user to commit**

Tell the user: "Task 5 complete — confetti fires on correct flag. Please test interactively and commit when happy."

---

## Task 6: Leaderboard view

**Files:**
- Modify: `internal/honeypot/ctf/ctf.go` — `gameState` consts (32-37), `Model` struct, `Update` switch (180-191), `View` switch (338-374), add `updateLeaderboard` + `renderLeaderboard`, add `l`/`L` binding in `updateMenu`

- [ ] **Step 1: Add the new state constant**

Edit `internal/honeypot/ctf/ctf.go` lines 32-37. Change:

```go
const (
	stateLogin gameState = iota
	stateMenu
	stateAnswer
	stateDone
)
```

to:

```go
const (
	stateLogin gameState = iota
	stateMenu
	stateAnswer
	stateLeaderboard
	stateDone
)
```

- [ ] **Step 2: Add leaderboard cache to Model**

Edit the `Model` struct (around lines 47-69 as modified in Task 4). Add one field just before `errMsg string`:

```go
	leaderboard []entity.CTFUser
```

- [ ] **Step 3: Wire `stateLeaderboard` into `Update`**

Edit `internal/honeypot/ctf/ctf.go` lines 180-191. Change:

```go
	switch m.state {
	case stateLogin:
		return m.updateLogin(msg)
	case stateMenu:
		return m.updateMenu(msg)
	case stateAnswer:
		return m.updateAnswer(msg)
	case stateDone:
		return m, func() tea.Msg { return QuitMsg{} }
	}
	return m, nil
}
```

to:

```go
	switch m.state {
	case stateLogin:
		return m.updateLogin(msg)
	case stateMenu:
		return m.updateMenu(msg)
	case stateAnswer:
		return m.updateAnswer(msg)
	case stateLeaderboard:
		return m.updateLeaderboard(msg)
	case stateDone:
		return m, func() tea.Msg { return QuitMsg{} }
	}
	return m, nil
}
```

- [ ] **Step 4: Add `l`/`L` binding to `updateMenu`**

In the `updateMenu` `switch msg.String()` (lines ~261-289 as modified in Task 4), add a new case **before** `case "q", "ctrl+c":`:

```go
		case "l", "L":
			board, err := entity.Leaderboard(10)
			if err != nil {
				m.errMsg = err.Error()
				return m, nil
			}
			m.leaderboard = board
			m.state = stateLeaderboard
			return m, nil
```

- [ ] **Step 5: Add `updateLeaderboard`**

Add this function in `internal/honeypot/ctf/ctf.go`, immediately after `updateAnswer`:

```go
func (m Model) updateLeaderboard(msg tea.Msg) (Model, tea.Cmd) {
	if _, ok := msg.(tea.KeyPressMsg); ok {
		m.state = stateMenu
		return m, nil
	}
	return m, nil
}
```

- [ ] **Step 6: Add the `stateLeaderboard` case to `View`**

Edit `internal/honeypot/ctf/ctf.go` lines 338-374 (`View`). Add a new case in the `switch m.state` block, after `case stateAnswer:` and before `case stateDone:`:

```go
	case stateLeaderboard:
		box := lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Padding(1, 2)
		return box.Render(m.renderLeaderboard())
```

- [ ] **Step 7: Add `renderLeaderboard`**

Add this method at the end of `internal/honeypot/ctf/ctf.go`:

```go
func (m Model) renderLeaderboard() string {
	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	rowStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	meStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true)
	footerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	var b strings.Builder
	b.WriteString(titleStyle.Render("Leaderboard") + "\n\n")

	if len(m.leaderboard) == 0 {
		b.WriteString(rowStyle.Render("No scores yet.") + "\n")
	} else {
		for i, u := range m.leaderboard {
			line := fmt.Sprintf("%2d. %s — %d pts", i+1, u.Username, u.Points)
			if m.user != nil && u.Username == m.user.Username {
				b.WriteString(meStyle.Render(line) + "\n")
			} else {
				b.WriteString(rowStyle.Render(line) + "\n")
			}
		}
	}

	b.WriteString("\n" + footerStyle.Render("press any key to return"))
	return b.String()
}
```

- [ ] **Step 8: Hint the binding in the menu footer (optional polish)**

In `View`'s `stateMenu` case (around line 352-358 as currently structured), update the rendered block so the footer mentions `L`. Change:

```go
	case stateMenu:
		header := fmt.Sprintf("Honey Bear Honey Pot CTF - %s (%d pts)", m.user.Username, m.user.Points)
		return lipgloss.JoinVertical(lipgloss.Left,
			titleStyle.Render(header),
			m.renderTasks(false),
			m.errMsg,
		)
```

to:

```go
	case stateMenu:
		header := fmt.Sprintf("Honey Bear Honey Pot CTF - %s (%d pts)", m.user.Username, m.user.Points)
		footer := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("press L for leaderboard, q to quit")
		return lipgloss.JoinVertical(lipgloss.Left,
			titleStyle.Render(header),
			m.renderTasks(false),
			m.errMsg,
			footer,
		)
```

- [ ] **Step 9: Build and run tests**

Run: `go build ./... && go test ./...`
Expected: PASS.

- [ ] **Step 10: Manual smoke test**

Start the honeypot. SSH in, `ctf`, login. From the task menu press `L`.

Expected:
- A bordered box appears with "Leaderboard" header.
- If you've earned points, your row is highlighted in yellow/bold.
- If no users have non-zero points yet, "No scores yet." is shown.
- "press any key to return" footer is visible.
- Any keypress (e.g., space, enter, q) returns to the task menu.

Test with multiple registered users to confirm ordering (highest points first, max 10).

- [ ] **Step 11: Pause for user to commit**

Tell the user: "Task 6 complete — leaderboard view bound to L. Please test and commit when ready."

---

## Final verification

- [ ] **Step 1: Format**

Run: `go fmt ./...`
Expected: no output (everything already formatted) or trivial whitespace fixes.

- [ ] **Step 2: Vet**

Run: `go vet ./...`
Expected: no warnings.

- [ ] **Step 3: Full test suite**

Run: `go test ./...`
Expected: PASS.

- [ ] **Step 4: Full build**

Run: `go build ./...`
Expected: clean.

- [ ] **Step 5: End-to-end manual run**

Start the honeypot with a config containing a mix of active and archived tasks (some completed by you, some not). Verify all three features end-to-end in a single session:

1. Login → see only active + your archived-completed tasks.
2. Press `L` → see leaderboard with yourself highlighted, return on any key.
3. Solve a task → confetti, returns to updated menu with task marked complete.

Report any deviations from expected behavior. The plan is complete when the user confirms behavior matches the spec.
