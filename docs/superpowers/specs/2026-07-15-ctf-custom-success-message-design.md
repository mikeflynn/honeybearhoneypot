# CTF Custom Success Message — Design

**Date:** 2026-07-15
**Status:** Approved

## Summary

Add an optional per-task custom success message to the CTF system. When a player
submits the correct flag for a task, the custom message is shown instead of the
default "Correct!" text. When no custom message is configured, the existing
default message is used unchanged.

## Motivation

Task authors want to give players a tailored, in-narrative reward when they solve
a specific task (e.g. hints toward the next task, story beats, congratulations
specific to the challenge) rather than the generic celebration line.

## Behavior

On a correct flag submission the success line is rendered in the existing
green/bold lipgloss style. Two cases:

- **No custom message (default):** `🎉 Correct! +N points! 🎉`
- **Custom message set:** `<SuccessMessage> +N points!`

The custom text replaces the "Correct!" portion, but the `+N points!` suffix is
**always appended** so score feedback is never lost. `N` is the task's point
value.

Everything else about a successful submission is unchanged: task marked
completed, leaderboard published, GUI notification event, confetti burst.

## Changes

### 1. Config schema (`internal/config/config.go`)

Add an optional field to `config.Task`:

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

`omitempty` keeps the field out of serialized output when unset, consistent with
`Archived`.

### 2. CTF model (`internal/honeypot/ctf/ctf.go`)

Add a matching field to `ctf.Task`:

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

Extract the success-message construction into a small pure helper so it is
testable without driving the Bubble Tea model:

```go
// successMessage returns the celebratory text shown after a correct flag.
// A custom SuccessMessage replaces the default "Correct!" text; the points
// suffix is always appended.
func successMessage(custom string, points int) string {
	if strings.TrimSpace(custom) == "" {
		return fmt.Sprintf("🎉 Correct! +%d points! 🎉", points)
	}
	return fmt.Sprintf("%s +%d points!", custom, points)
}
```

In `updateAnswer` (currently `ctf.go:386-388`), replace the hardcoded
`fmt.Sprintf(...)` inside the green/bold style with a call to
`successMessage(m.selectedTask.SuccessMessage, m.selectedTask.Points)`. Styling
is unchanged.

### 3. Plumbing (`internal/honeypot/pot.go`)

In `convertTasks`, copy the new field:

```go
out[i] = ctf.Task{
	Name:           task.Name,
	Description:    task.Description,
	Flag:           task.Flag,
	Points:         task.Points,
	Archived:       task.Archived,
	SuccessMessage: task.SuccessMessage,
}
```

### 4. Sample config (`config.sample.json`)

Add `success_message` to the demo task so the option is discoverable:

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

## Testing

Unit tests for `successMessage` covering both branches:

- Empty custom string → default `🎉 Correct! +N points! 🎉`.
- Whitespace-only custom string → treated as empty → default.
- Non-empty custom string → `<custom> +N points!`.

Optionally assert `config.Task` round-trips `success_message` through JSON
(mirrors the existing `TestTaskArchivedJSON`).

## Out of Scope / Non-Goals

- No database schema or migration changes — this is display-time only.
- No custom message for incorrect submissions.
- No per-message styling controls; the existing green/bold style is retained.
```