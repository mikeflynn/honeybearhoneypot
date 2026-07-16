# CTF Custom Success Message Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let each CTF task define an optional custom success message shown when a player submits the correct flag, falling back to the current default when unset.

**Architecture:** Add an optional `SuccessMessage` field to `config.Task`, carry it through `convertTasks` into `ctf.Task`, and render it via a small pure helper (`successMessage`) in the CTF answer flow. Display-time only — no DB or migration changes.

**Tech Stack:** Go 1.24+, Charmbracelet Bubble Tea / lipgloss, standard `testing` package.

## Global Constraints

- Standard Go formatting (`go fmt ./...`).
- CGO required to build/run the full app (SQLite + Fyne); package-level `go test` for `internal/config` and `internal/honeypot/ctf` runs without a display.
- Never commit to git — the user handles all version control and deployment manually. The "Commit" steps below are replaced by leaving changes staged/unstaged for the user; do NOT run `git commit`.
- Custom message rendering: custom text replaces the "Correct!" portion; the `+N points!` suffix is always appended. Empty or whitespace-only custom → default `🎉 Correct! +N points! 🎉`.

---

### Task 1: Add `SuccessMessage` to config schema

**Files:**
- Modify: `internal/config/config.go:23-29` (the `Task` struct)
- Test: `internal/config/config_test.go`

**Interfaces:**
- Produces: `config.Task.SuccessMessage string` with JSON tag `success_message,omitempty`.

- [ ] **Step 1: Write the failing test**

Add to `internal/config/config_test.go`:

```go
func TestTaskSuccessMessageJSON(t *testing.T) {
	t.Run("success_message round-trips", func(t *testing.T) {
		in := Task{Name: "x", Flag: "f", Points: 1, SuccessMessage: "well done"}
		b, err := json.Marshal(in)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		var out Task
		if err := json.Unmarshal(b, &out); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if out.SuccessMessage != "well done" {
			t.Fatalf("expected SuccessMessage=well done, got %q; json=%s", out.SuccessMessage, string(b))
		}
	})

	t.Run("success_message omitted when empty", func(t *testing.T) {
		b, err := json.Marshal(Task{Name: "x", Flag: "f", Points: 1})
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if contains(string(b), "success_message") {
			t.Fatalf("expected omitempty to drop success_message field, got %s", string(b))
		}
	})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/config/ -run TestTaskSuccessMessageJSON`
Expected: FAIL — `Task` has no field `SuccessMessage`.

- [ ] **Step 3: Add the field**

In `internal/config/config.go`, update the `Task` struct:

```go
type Task struct {
	Name           string `json:"name"`
	Description    string `json:"description"`
	Flag           string `json:"flag"`
	Points         int    `json:"points"`
	Archived       bool   `json:"archived,omitempty"`
	SuccessMessage string `json:"success_message,omitempty"`
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/config/ -run TestTaskSuccessMessageJSON`
Expected: PASS.

- [ ] **Step 5: Format**

Run: `go fmt ./internal/config/`
Leave changes for the user (do not `git commit`).

---

### Task 2: Add `successMessage` helper + `SuccessMessage` field to `ctf.Task`

**Files:**
- Modify: `internal/honeypot/ctf/ctf.go:78-85` (the `Task` struct) and `ctf.go:374-441` (`updateAnswer`)
- Test: `internal/honeypot/ctf/ctf_test.go`

**Interfaces:**
- Consumes: nothing new.
- Produces:
  - `ctf.Task.SuccessMessage string`
  - `func successMessage(custom string, points int) string`

- [ ] **Step 1: Write the failing test**

Add to `internal/honeypot/ctf/ctf_test.go`:

```go
func TestSuccessMessage(t *testing.T) {
	t.Run("empty uses default", func(t *testing.T) {
		got := successMessage("", 50)
		want := "🎉 Correct! +50 points! 🎉"
		if got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})

	t.Run("whitespace-only uses default", func(t *testing.T) {
		got := successMessage("   ", 50)
		want := "🎉 Correct! +50 points! 🎉"
		if got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})

	t.Run("custom replaces with points suffix", func(t *testing.T) {
		got := successMessage("The vault is open.", 50)
		want := "The vault is open. +50 points!"
		if got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/honeypot/ctf/ -run TestSuccessMessage`
Expected: FAIL — `successMessage` undefined.

- [ ] **Step 3: Add the helper and struct field**

In `internal/honeypot/ctf/ctf.go`, add the field to `Task`:

```go
type Task struct {
	Name           string
	Description    string
	Flag           string
	Points         int
	Completed      bool
	Archived       bool
	SuccessMessage string
}
```

Add the helper (place it near `updateAnswer`, e.g. just above it):

```go
// successMessage returns the celebratory text shown after a correct flag.
// A non-blank custom message replaces the default "Correct!" text; the points
// suffix is always appended.
func successMessage(custom string, points int) string {
	if strings.TrimSpace(custom) == "" {
		return fmt.Sprintf("🎉 Correct! +%d points! 🎉", points)
	}
	return fmt.Sprintf("%s +%d points!", custom, points)
}
```

(`strings` and `fmt` are already imported in `ctf.go`.)

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/honeypot/ctf/ -run TestSuccessMessage`
Expected: PASS.

- [ ] **Step 5: Wire the helper into `updateAnswer`**

In `internal/honeypot/ctf/ctf.go`, inside `updateAnswer`, replace the current success-text block (currently `ctf.go:386-388`):

```go
					m.errMsg = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true).Render(
						fmt.Sprintf("🎉 Correct! +%d points! 🎉", m.selectedTask.Points),
					)
```

with:

```go
					m.errMsg = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true).Render(
						successMessage(m.selectedTask.SuccessMessage, m.selectedTask.Points),
					)
```

- [ ] **Step 6: Run the full package test to confirm nothing broke**

Run: `go test ./internal/honeypot/ctf/`
Expected: PASS (all tests).

- [ ] **Step 7: Format**

Run: `go fmt ./internal/honeypot/ctf/`
Leave changes for the user (do not `git commit`).

---

### Task 3: Plumb `SuccessMessage` through `convertTasks`

**Files:**
- Modify: `internal/honeypot/pot.go:421-433` (`convertTasks`)

**Interfaces:**
- Consumes: `config.Task.SuccessMessage` (Task 1), `ctf.Task.SuccessMessage` (Task 2).
- Produces: `convertTasks` copies `SuccessMessage` into each `ctf.Task`.

- [ ] **Step 1: Update `convertTasks`**

In `internal/honeypot/pot.go`, update the struct literal in `convertTasks`:

```go
func convertTasks(t []config.Task) []ctf.Task {
	out := make([]ctf.Task, len(t))
	for i, task := range t {
		out[i] = ctf.Task{
			Name:           task.Name,
			Description:    task.Description,
			Flag:           task.Flag,
			Points:         task.Points,
			Archived:       task.Archived,
			SuccessMessage: task.SuccessMessage,
		}
	}
	return out
}
```

- [ ] **Step 2: Build the package to confirm it compiles**

Run: `go build ./internal/honeypot/`
Expected: no output (success). Requires CGO — run in the normal dev environment.

- [ ] **Step 3: Format**

Run: `go fmt ./internal/honeypot/`
Leave changes for the user (do not `git commit`).

---

### Task 4: Document the option in the sample config

**Files:**
- Modify: `config.sample.json:17-24` (the `tasks` array)

**Interfaces:**
- Consumes: the `success_message` JSON key (Task 1).

- [ ] **Step 1: Add `success_message` to the demo task**

In `config.sample.json`, update the `tasks` entry:

```json
  "tasks": [
    {
      "name": "demo",
      "description": "Example flag",
      "flag": "flag{demo}",
      "points": 100,
      "success_message": "Nicely done — the vault is open."
    }
  ],
```

- [ ] **Step 2: Verify the JSON is valid**

Run: `python3 -m json.tool config.sample.json > /dev/null && echo OK`
Expected: `OK`.

Leave changes for the user (do not `git commit`).

---

## Final Verification

- [ ] Run the full test suite: `go test ./...`
  Expected: PASS (CGO-enabled dev environment).
- [ ] Confirm formatting is clean: `go fmt ./...` prints no changed files.

## Self-Review Notes

- **Spec coverage:** config schema (Task 1), ctf.Task field + render behavior via helper (Task 2), plumbing (Task 3), sample config docs (Task 4), tests for both helper branches + whitespace + JSON round-trip (Tasks 1–2). All spec sections covered.
- **Type consistency:** `SuccessMessage string` and `success_message` JSON tag used identically across `config.Task`, `ctf.Task`, and `convertTasks`; helper signature `successMessage(custom string, points int) string` matches its call site.
- **Placeholders:** none — all steps contain concrete code/commands.
- **Git:** per project rule, all "commit" actions are replaced with "leave changes for the user."
