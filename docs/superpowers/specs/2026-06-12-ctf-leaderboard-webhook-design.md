# CTF Leaderboard Webhook — Design

**Date:** 2026-06-12
**Status:** Draft, pending implementation plan

## Goal

Allow the honeypot to publish its current CTF leaderboard to an externally-hosted site by POSTing a JSON snapshot whenever the leaderboard changes, plus an optional heartbeat so receivers that miss a POST self-heal on the next tick or solve.

## Non-goals

- HMAC signing, shared-secret auth, or any per-request authentication. The URL itself is treated as the secret.
- Retries on transient receiver failure. The next solve or heartbeat overwrites the previous snapshot, so a dropped POST repairs itself.
- An embedded HTTP server on the honeypot, or static-file output. The receiver is an external site the operator already runs.
- Per-user registration events, activity-feed style payloads, or anything beyond the leaderboard snapshot.
- A general-purpose webhook framework. Extensibility for future webhook types (e.g. event logging) is structural only — see Extensibility below.

## Configuration

Webhooks are namespaced under a single `webhooks` block in `Config` so future webhook types can be added without reshaping the config:

```json
{
  "webhooks": {
    "leaderboard": {
      "url": "https://example.com/honeybear/leaderboard",
      "limit": 10,
      "heartbeat_seconds": 300,
      "timeout_seconds": 5
    }
  }
}
```

Go types in `internal/config`:

```go
type Webhooks struct {
    Leaderboard *LeaderboardWebhook `json:"leaderboard,omitempty"`
}

type LeaderboardWebhook struct {
    URL              string `json:"url,omitempty"`               // required to enable
    Limit            int    `json:"limit,omitempty"`             // top-N, default 10
    HeartbeatSeconds int    `json:"heartbeat_seconds,omitempty"` // 0 disables; recommended 300
    TimeoutSeconds   int    `json:"timeout_seconds,omitempty"`   // default 5
}
```

Added to `Config` as `Webhooks *Webhooks` (pointer so omitting the block fully disables webhooks). Empty `URL` also disables.

CLI flags for the common operator knobs:

- `-leaderboard-webhook-url`
- `-leaderboard-webhook-heartbeat` (seconds)

`limit` and `timeout_seconds` are config-file only. The existing `${ENV_VAR}` expansion handled by `config.Parse` applies to `url`.

Defaults:

- `Limit`: 10
- `HeartbeatSeconds`: 0 (disabled — operator must opt in)
- `TimeoutSeconds`: 5

## Payload

A single payload shape, always a full snapshot:

```json
{
  "event": "solve",
  "timestamp": "2026-06-12T18:04:11Z",
  "source": "honeybear",
  "leaderboard": [
    { "rank": 1, "username": "alice", "points": 150 },
    { "rank": 2, "username": "bob",   "points": 90 }
  ]
}
```

- `event` is one of `"solve"`, `"heartbeat"`, `"startup"`. Informational only — the receiver does not need to keep state.
- `timestamp` is RFC3339 UTC.
- `source` is the literal string `"honeybear"` so a receiver hosting multiple data sources can route.
- `leaderboard` is the top-N as returned by `entity.Leaderboard(limit)`, projected to `{rank, username, points}` only. Internal fields like `ID`, `CreatedAt`, and `Password` are not exposed.
- An empty leaderboard sends `"leaderboard": []` (not omitted).

HTTP request:

- Method: `POST`
- `Content-Type: application/json`
- `User-Agent: honeybear-webhook/<version>`
- Body: JSON above, no trailing newline required

## Architecture

### Package layout

```
internal/honeypot/webhook/
    dispatcher.go        // generic Dispatcher: goroutine, channel, coalescing, heartbeat, POST
    dispatcher_test.go
    leaderboard/
        leaderboard.go   // builds the leaderboard payload, owns entity.Leaderboard call
        leaderboard_test.go
```

The split exists so a future webhook type (e.g. event logging) can reuse `Dispatcher` with a different payload builder. We are not building that second type now; the split is the only forward-looking work.

### Dispatcher (generic)

```go
package webhook

type PayloadFunc func(event string) (any, error)

type Dispatcher struct { /* unexported */ }

type Config struct {
    URL              string
    TimeoutSeconds   int
    HeartbeatSeconds int
    Payload          PayloadFunc
    Logger           *slog.Logger
    Version          string // for User-Agent
}

func NewDispatcher(cfg Config) *Dispatcher
func (d *Dispatcher) Publish(event string)  // non-blocking, coalescing
func (d *Dispatcher) Close()
```

Internals:

- One worker goroutine owns a buffered slot (capacity 1) protected by a mutex. `Publish` takes the lock, compares the incoming event to whatever is already pending using the priority `solve` > `startup` > `heartbeat`, keeps the higher-priority one, and signals the worker. Effectively, a burst of solves becomes one POST after the in-flight one completes.
- Worker loop: read event → call `cfg.Payload(event)` → marshal JSON → POST with `http.Client{Timeout: ...}` → log result → loop.
- If `HeartbeatSeconds > 0`, a ticker goroutine calls `Publish("heartbeat")` on each tick.
- On startup, the caller fires one `Publish("startup")` so a fresh receiver gets state immediately.
- `Close()` stops the ticker, drains/cancels the pending event, and waits for the worker to exit.
- If `URL` is empty or `Payload` is nil, `NewDispatcher` returns a no-op dispatcher (so callers don't need to nil-check at every `Publish` site).

### Leaderboard subpackage

```go
package leaderboard

func New(cfg *config.LeaderboardWebhook, version string, logger *slog.Logger) *webhook.Dispatcher
```

`New` wires a `webhook.Dispatcher` with a `PayloadFunc` that calls `entity.Leaderboard(cfg.Limit)`, projects to `{rank, username, points}`, and returns the snapshot struct. Returns a no-op dispatcher if `cfg` is nil or `cfg.URL` is empty.

### Integration points

1. `main.go` — after DB init, construct the leaderboard dispatcher and store it on a small package-level accessor in `internal/honeypot/ctf` (e.g. `ctf.SetLeaderboardPublisher(d)`). Call `d.Publish("startup")`. Defer `d.Close()`.
2. `internal/honeypot/ctf/ctf.go` — immediately after the successful `m.user.CompleteTask(...)` call (currently around line 366), call the configured publisher's `Publish("solve")`. Publish failure must not affect the SSH session; the call is fire-and-forget by contract.
3. No other call sites.

### Failure handling

- Non-2xx response, network error, or timeout: log at `warn` with the response status (if any), the URL host (not the full URL, to avoid leaking any path-embedded secret a future operator might add), and the event type. Drop the event. The next solve or heartbeat repairs receiver state.
- DB error inside the payload builder: log at `warn`, skip the POST.
- Invalid URL at startup: log at `error`, return a no-op dispatcher. App keeps running.

### Logging

Use the existing `slog` logger. Levels:

- `info` once at startup when a dispatcher is enabled, including URL host and heartbeat interval.
- `debug` on each successful POST.
- `warn` on failed POSTs or DB errors during payload build.
- `error` on startup misconfiguration.

## Testing

Unit tests in `internal/honeypot/webhook`:

- Publishes a single POST with the expected JSON shape against an `httptest.Server`.
- Three rapid `Publish` calls during a stalled in-flight POST result in exactly one follow-up POST (coalescing).
- Heartbeat ticker fires `heartbeat` events at the configured interval.
- `solve` overrides a pending `heartbeat` in the channel.
- Timeout: server that sleeps past the configured timeout is logged as a failure and does not block the worker.
- Disabled dispatcher (empty URL or nil payload) accepts `Publish` calls without making any HTTP request.

Unit tests in `internal/honeypot/webhook/leaderboard`:

- Payload builder returns the expected projection given a seeded DB (using existing test DB helpers).
- Honors `Limit`.
- Empty leaderboard produces `"leaderboard": []`, not a missing key.

No new integration test in the CTF flow; the call site is a single line and is covered by the publisher unit tests plus the existing CTF tests verifying solve behavior.

## Extensibility

The dispatcher is generic over its payload. To add a second webhook type later (e.g. event logging):

1. Add a new subpackage under `internal/honeypot/webhook/`.
2. Add a new field to `config.Webhooks` (e.g. `Events *EventsWebhook`).
3. Build a dispatcher with that subpackage's `PayloadFunc`.
4. Call `Publish(...)` from the new event sites.

Note: event logging will likely want different semantics — per-event payloads, no coalescing, possibly retries — which the current `Dispatcher` does not provide. Expect to either extend `Dispatcher` with options at that point or introduce a sibling type. That decision belongs to that future change, not this one.
