# CTF Updates: Archive, Celebrate, Leaderboard

Date: 2026-05-18

## Goal

Three additions to the CTF subsystem (`internal/honeypot/ctf/`, `internal/entity/ctf_user.go`, `internal/config/config.go`):

1. Allow tasks to be archived in config so they remain visible (with credit) to users who already solved them, but disappear for everyone else.
2. Fire a confetti burst inside the CTF TUI when a user submits a correct flag.
3. Add a leaderboard view to the CTF menu.

No admin-GUI work, no DB migrations, no changes to scoring math.

## 1. Archive task

### Config

Add `Archived bool` to `config.Task` (`internal/config/config.go:12`):

```go
type Task struct {
    Name        string `json:"name"`
    Description string `json:"description"`
    Flag        string `json:"flag"`
    Points      int    `json:"points"`
    Archived    bool   `json:"archived,omitempty"`
}
```

Propagate the field through `convertTasks` in `internal/honeypot/pot.go:421` to a matching new field on `ctf.Task` (`internal/honeypot/ctf/ctf.go:39`).

### Filtering

After `loadCompleted()` in `ctf.go`, partition `m.tasks` into two derived slices held on the model:

- `activeTasks`: tasks where `!Archived`.
- `archivedDone`: tasks where `Archived && Completed`.

Archived tasks the user did not complete are excluded from both slices and never rendered. `m.tasks` (master list) is retained so reloads still work; the derived slices are recomputed in `loadCompleted` and after a successful answer.

### Rendering

`renderTasks` is restructured:

1. Render `activeTasks` exactly as today (cursor, bullets, `Completed` styling for users who finished an active task).
2. If `len(archivedDone) > 0`, render a dimmed separator `── Archived ──` and then each archived-completed task in greyed style with the `✓` bullet. No cursor marker, not selectable.

### Navigation

`updateMenu` cursor bounds change from `len(m.tasks)-1` to `len(m.activeTasks)-1`. `enter` selects from `activeTasks`. If `activeTasks` is empty, `enter` is a no-op and the menu shows "All active tasks complete!" above the archived list (if any).

### Behavior summary

| User state for task | Task active | Task archived |
|---|---|---|
| Not completed | Visible, selectable | Hidden |
| Completed | Visible, ✓, not selectable | Visible in Archived section, ✓ |

Points already credited are never altered. Flag submission paths cannot reach archived tasks because they're not on the menu.

## 2. Celebrate on correct flag

In `ctf.go:298–323`, the correct-flag branch currently sets `m.errMsg` and returns `m, nil`. Change it to return a `tea.Batch` that triggers the existing top-level confetti overlay:

```go
return m, tea.Batch(
    func() tea.Msg { return filesystem.SetRunningCmd("confetti") },
    func() tea.Msg { return confetti.Burst() },
    tea.Tick(4*time.Second, func(time.Time) tea.Msg { return filesystem.SetRunningCmd("") }),
)
```

Imports added to `ctf.go`: `confetti` and `filesystem` packages from `internal/honeypot/`.

The parent honeypot model (`internal/honeypot/model.go:267,305`) already routes `confetti.Burst()` messages to the top-level `confetti.Model` and swaps the view to the confetti overlay while `runningCommand == "confetti"`. The CTF model stays at `stateMenu` underneath; when the 4-second tick fires and clears `runningCmd`, the menu reappears with the green "🎉 Correct! +N points! 🎉" message already set on `m.errMsg`, and the task marked `Completed`.

The 4-second timing matches the existing `celebrate` filesystem command (`filesystem.go:594`) so behavior is consistent across the two entry points. No changes are required in `model.go` or `confetti/`.

The background `entity.Event` publish for `taskCompleted` remains unchanged.

## 3. Leaderboard view

Add a fourth `gameState`:

```go
stateLeaderboard
```

### Trigger

In `updateMenu`, bind `l` / `L` to set `m.state = stateLeaderboard` and load the top 10 via `entity.Leaderboard(10)` (already exists, `ctf_user.go:113`) into `m.leaderboard []entity.CTFUser` on the model. Errors set `m.errMsg` and leave the state on `stateMenu`.

### Rendering

`View()` gets a `stateLeaderboard` case that renders a bordered box (same style as `stateAnswer`) with:

- Title: "Leaderboard".
- Numbered list of up to 10 entries: `1. <username> — <points> pts`. The viewing user's row is highlighted (bold + accent color) if present.
- Empty state: "No scores yet."
- Footer: "press any key to return".

### Exit

`updateLeaderboard` returns to `stateMenu` on any keypress.

## Out of scope

- Admin-GUI archive toggle (config-only per user decision).
- DB migration for archive state (not needed; archive is config-side).
- First-blood bonuses, hints, categories, rate limiting, achievements, MOTD, activity feed — explicitly deferred.

## Testing

- Manual: launch with `go run main.go`, SSH in, start `ctf`, register, answer a task correctly → confetti fires, menu returns with green confirmation, task shows ✓.
- Manual: mark a completed task `"archived": true` in config, restart; for the original solver the task appears under "── Archived ──"; for a fresh user the task is absent.
- Manual: press `L` in the menu, confirm leaderboard renders, any key returns.
- `go test ./...` should still pass (no existing CTF tests to update; none added — these are TUI/integration changes).
- `go fmt ./...` and `go build` must succeed.

## Files touched

- `internal/config/config.go` — add `Archived` field.
- `internal/honeypot/pot.go` — pass `Archived` through `convertTasks`.
- `internal/honeypot/ctf/ctf.go` — `Task.Archived`, derived slices, render changes, leaderboard state, confetti dispatch, new imports.
