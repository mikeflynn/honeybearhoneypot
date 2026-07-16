# CTF Arcade Success Screen — Design

**Date:** 2026-07-15
**Status:** Approved
**Supersedes:** the display portion of `2026-07-15-ctf-custom-success-message-design.md` (the custom-message config field stays; only how it is rendered changes).

## Summary

Replace the one-line green "Correct!" message + full-screen confetti takeover on a
correct CTF flag with a dedicated animated arcade "LEVEL COMPLETE" success screen
rendered inside the CTF model. The custom per-task success message (added earlier)
becomes the headline text on this screen.

## Motivation

The current success feedback is a single styled line plus a generic confetti burst.
An arcade-style success screen gives solving a task a satisfying, retro "wow"
moment and a more prominent points payoff.

## Behavior

On a correct flag submission:

1. The task is completed (DB write, leaderboard publish, GUI notification event) —
   unchanged from today.
2. The CTF view transitions to a new `stateSuccess` screen instead of returning
   straight to the menu and instead of firing `ConfettiMsg`.
3. The screen animates for as long as it is shown (see Animation).
4. Any keypress returns to `stateMenu`, where the task now shows as completed.

The confetti feature itself (the `celebrate` command and the confetti model) is
**untouched** — the CTF path simply no longer routes through it.

### Screen layout

```
        ·  ·  ✦  LEVEL COMPLETE  ✦  ·  ·

              🏆   🍯   🏆

     >>>  Nicely done — the vault is open.  <<<

       BONUS ................ +50
       SCORE ................ 250

          press any key to continue
```

- **Headline:** `✦ LEVEL COMPLETE ✦`, color-cycling through an 8-bit palette.
- **Trophy row:** `🏆   🍯   🏆`.
- **Message:** the task's custom success message if set, otherwise the default
  `Challenge Complete!`. No `+N points!` suffix here — points appear in the BONUS
  line.
- **BONUS:** dotted leader line showing points earned for this task, `+N`.
- **SCORE:** dotted leader line showing the player's running total.
- **Prompt:** blinking `press any key to continue`.

### Animation

Driven by an ~80ms `tea.Tick` (`successTickMsg`) that is issued when entering
`stateSuccess` and re-issued each tick **only while still in** `stateSuccess`, so
it stops on its own after the player leaves.

- **Color cycle:** headline foreground steps through an 8-bit palette
  (`9` red, `11` yellow, `10` green, `14` cyan, `13` magenta, `12` blue) indexed by
  `frame`.
- **Count-up tally:** `BONUS` ramps `0 → task points` and `SCORE` ramps
  `oldTotal → newTotal` over a fixed duration (`tallyFrames`, ~1s at 80ms/frame),
  then holds. Uses `tallyValue`.
- **Blink:** `press any key to continue` is shown when `(frame/blinkFrames)%2 == 0`.

## Changes

All changes are in `internal/honeypot/ctf/ctf.go` and its test file. No `model.go`,
`pot.go`, or `config.go` changes are required (the parent already routes messages to
and renders `m.ctf` while `runningCommand == "ctf"`, and the parent's esc/ctrl+c
footer remains).

### 1. State + model fields (`ctf.go`)

- Add `stateSuccess` to the `gameState` enum.
- Add fields to `Model` for the success snapshot captured at transition time:
  - `successFrame int`
  - `successBanner string` (the headline message text)
  - `successBonus int` (points earned this task)
  - `successOldTotal int`
  - `successNewTotal int`

### 2. Tick message + command (`ctf.go`)

```go
type successTickMsg struct{}

func successTick() tea.Cmd {
	return tea.Tick(80*time.Millisecond, func(time.Time) tea.Msg {
		return successTickMsg{}
	})
}
```

Handle `successTickMsg` in the top-level `Update` (like `leaderboardLoadedMsg`):
increment `m.successFrame` only when `m.state == stateSuccess`, and return
`successTick()` to keep animating while in that state; otherwise ignore.

### 3. Pure helpers (`ctf.go`)

Replace `successMessage(custom string, points int) string` (from the previous
change) with:

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
	fixed := len(label) + len(value) + 2 // spaces around the dots
	dots := width - fixed
	if dots < 1 {
		dots = 1
	}
	return fmt.Sprintf("%s %s %s", label, strings.Repeat(".", dots), value)
}
```

Note `dottedLeader` measures with `len()` on ASCII-only label/value (the values are
`+N`, integers, and fixed ASCII labels), which is safe here.

### 4. Transition on correct flag (`updateAnswer`, `ctf.go`)

In the correct-answer branch, after `m.user.CompleteTask(...)` succeeds:

- Set the snapshot: `m.successBanner = successBanner(m.selectedTask.SuccessMessage)`,
  `m.successBonus = m.selectedTask.Points`,
  `m.successNewTotal = m.user.Points`,
  `m.successOldTotal = m.user.Points - m.selectedTask.Points`,
  `m.successFrame = 0`.
- Keep: `m.selectedTask.Completed = true`, `publishLeaderboard("solve")`, the GUI
  notification goroutine, and the `partitionTasks` recompute + cursor clamp.
- Remove the `m.errMsg = ...` success line and the `celebrate` confetti batch.
- `m.state = stateSuccess` and `return m, successTick()`.

The incorrect-flag path and `esc` handling are unchanged.

### 5. State routing (`Update`, `ctf.go`)

Add `case stateSuccess: return m.updateSuccess(msg)` to the state switch.

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

### 6. View (`View` + `renderSuccess`, `ctf.go`)

Add `case stateSuccess: return m.renderSuccess()` to the `View` switch.
`renderSuccess` composes the layout above using `m.successFrame` for the color
cycle / blink and `tallyValue` for the BONUS/SCORE counts, centered within
`m.width`. The `ConfettiMsg` type and `Quit`/`QuitMsg` are retained (confetti type
stays exported for the `celebrate` command; it is simply no longer sent from the CTF
success path).

## Testing

Unit tests (no Bubble Tea driving required):

- `successBanner`: empty → `Challenge Complete!`; whitespace-only → default;
  custom → trimmed custom.
- `tallyValue`: `frame<=0` → start; `frame>=duration` → end; midpoint interpolates
  (e.g. `tallyValue(0, 50, 5, 10) == 25`); `duration<=0` → end.
- `dottedLeader`: contains label and value, total length == requested width for a
  width larger than the fixed content; minimum one dot when cramped.

## Out of Scope / Non-Goals

- No changes to the confetti model or the `celebrate` command.
- No sound, no per-task custom art beyond the message text.
- No config schema changes — this builds on the existing `success_message` field.
```