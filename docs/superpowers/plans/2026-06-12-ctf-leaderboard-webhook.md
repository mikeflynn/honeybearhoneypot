# CTF Leaderboard Webhook Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Publish the CTF leaderboard to an operator-configured external URL via JSON webhook on every solve, with optional heartbeat.

**Architecture:** A generic `webhook.Dispatcher` (goroutine + 1-slot mailbox + priority coalescing + ticker + HTTP POST) under `internal/honeypot/webhook`, plus a thin `leaderboard` subpackage that wires it to `entity.Leaderboard`. The CTF model fires `Publish("solve")` after a successful `CompleteTask`; `main.go` fires `Publish("startup")` once and defers `Close()`.

**Tech Stack:** Go 1.24, stdlib `net/http`, `charm.land/log/v2` (existing project logger), `httptest` for tests. No new dependencies.

**Reference spec:** `docs/superpowers/specs/2026-06-12-ctf-leaderboard-webhook-design.md`

**Note on logging:** The spec mentions `*slog.Logger`, but this codebase uses package-level `charm.land/log/v2` (`log.Info`, `log.Warn`, etc.) everywhere — main.go, honeypot, entity. The plan follows the existing convention: dispatcher calls package logger directly, no logger parameter.

---

## File Structure

**New:**
- `internal/honeypot/webhook/dispatcher.go` — generic dispatcher
- `internal/honeypot/webhook/dispatcher_test.go`
- `internal/honeypot/webhook/leaderboard/leaderboard.go` — leaderboard payload + constructor
- `internal/honeypot/webhook/leaderboard/leaderboard_test.go`

**Modified:**
- `internal/config/config.go` — add `Webhooks` block, CLI flags, merge logic
- `internal/honeypot/ctf/ctf.go` — package-level publisher accessor + `Publish("solve")` after `CompleteTask`
- `main.go` — construct dispatcher, fire startup event, defer Close

---

## Task 1: Config types and merge

**Files:**
- Modify: `internal/config/config.go`

- [ ] **Step 1: Add config types and field**

Add above the `Config` struct definition:

```go
type Webhooks struct {
	Leaderboard *LeaderboardWebhook `json:"leaderboard,omitempty"`
}

type LeaderboardWebhook struct {
	URL              string `json:"url,omitempty"`
	Limit            int    `json:"limit,omitempty"`
	HeartbeatSeconds int    `json:"heartbeat_seconds,omitempty"`
	TimeoutSeconds   int    `json:"timeout_seconds,omitempty"`
}
```

Add field inside `Config` struct (after `CurlResponses` / `NmapHosts`, near the bottom):

```go
	Webhooks *Webhooks `json:"webhooks,omitempty"`
```

- [ ] **Step 2: Add CLI flag declarations**

Inside the `var (...)` block where other flags are declared (after `exportTypesFlag`):

```go
	leaderboardWebhookURLFlag       = flag.String("leaderboard-webhook-url", "", "URL to POST CTF leaderboard snapshots to (enables leaderboard webhook)")
	leaderboardWebhookHeartbeatFlag = flag.Int("leaderboard-webhook-heartbeat", 0, "Heartbeat interval in seconds for leaderboard webhook (0 disables)")
```

- [ ] **Step 3: Wire CLI flags into Parse**

In `Parse()`, after the existing export-flag block, add:

```go
	if *leaderboardWebhookURLFlag != "" {
		if cfg.Webhooks == nil {
			cfg.Webhooks = &Webhooks{}
		}
		if cfg.Webhooks.Leaderboard == nil {
			cfg.Webhooks.Leaderboard = &LeaderboardWebhook{}
		}
		cfg.Webhooks.Leaderboard.URL = *leaderboardWebhookURLFlag
	}
	if *leaderboardWebhookHeartbeatFlag != 0 {
		if cfg.Webhooks == nil {
			cfg.Webhooks = &Webhooks{}
		}
		if cfg.Webhooks.Leaderboard == nil {
			cfg.Webhooks.Leaderboard = &LeaderboardWebhook{}
		}
		cfg.Webhooks.Leaderboard.HeartbeatSeconds = *leaderboardWebhookHeartbeatFlag
	}
```

- [ ] **Step 4: Add merge logic**

Inside `merge(...)`, after the existing `NmapHosts` block, add:

```go
	if src.Webhooks != nil {
		if dst.Webhooks == nil {
			dst.Webhooks = &Webhooks{}
		}
		if src.Webhooks.Leaderboard != nil {
			if dst.Webhooks.Leaderboard == nil {
				dst.Webhooks.Leaderboard = &LeaderboardWebhook{}
			}
			lb := src.Webhooks.Leaderboard
			dlb := dst.Webhooks.Leaderboard
			if lb.URL != "" {
				dlb.URL = lb.URL
			}
			if lb.Limit != 0 {
				dlb.Limit = lb.Limit
			}
			if lb.HeartbeatSeconds != 0 {
				dlb.HeartbeatSeconds = lb.HeartbeatSeconds
			}
			if lb.TimeoutSeconds != 0 {
				dlb.TimeoutSeconds = lb.TimeoutSeconds
			}
		}
	}
```

- [ ] **Step 5: Verify build**

Run: `go build ./...`
Expected: no errors.

- [ ] **Step 6: Commit**

```bash
git add internal/config/config.go
git commit -m "feat(config): add webhooks.leaderboard config block and flags"
```

---

## Task 2: Generic Dispatcher — type, constructor, no-op behavior

**Files:**
- Create: `internal/honeypot/webhook/dispatcher.go`
- Test: `internal/honeypot/webhook/dispatcher_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/honeypot/webhook/dispatcher_test.go`:

```go
package webhook

import "testing"

func TestNewDispatcher_NoopWhenURLEmpty(t *testing.T) {
	d := NewDispatcher(Config{
		URL:     "",
		Payload: func(string) (any, error) { return nil, nil },
	})
	if d == nil {
		t.Fatal("expected non-nil dispatcher")
	}
	d.Publish("solve") // must not panic
	d.Close()
}

func TestNewDispatcher_NoopWhenPayloadNil(t *testing.T) {
	d := NewDispatcher(Config{URL: "http://example.invalid"})
	d.Publish("solve")
	d.Close()
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/honeypot/webhook/ -run TestNewDispatcher -v`
Expected: FAIL with "package webhook is not in std" or "undefined: NewDispatcher".

- [ ] **Step 3: Implement Dispatcher skeleton**

Create `internal/honeypot/webhook/dispatcher.go`:

```go
// Package webhook provides a generic, fire-and-forget HTTP POST dispatcher with
// coalescing and optional heartbeat. Payload construction is delegated to the
// caller so the same machinery can serve different webhook types.
package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"charm.land/log/v2"
)

const defaultTimeoutSeconds = 5

// PayloadFunc builds the JSON-serializable payload for a given event name.
type PayloadFunc func(event string) (any, error)

// Config configures a Dispatcher. URL and Payload are required to do real work;
// if either is missing the dispatcher is a no-op.
type Config struct {
	URL              string
	TimeoutSeconds   int
	HeartbeatSeconds int
	Payload          PayloadFunc
	Version          string // included in User-Agent
	// HTTPClient is optional; if nil, one is built using TimeoutSeconds.
	HTTPClient *http.Client
}

// Dispatcher posts payloads to a URL on demand. It is safe for concurrent use.
type Dispatcher struct {
	cfg     Config
	enabled bool
	host    string

	mu      sync.Mutex
	pending string // "", "heartbeat", "startup", "solve"
	wake    chan struct{}

	stop   chan struct{}
	doneWG sync.WaitGroup
}

// NewDispatcher returns a Dispatcher. If URL is empty or Payload is nil, the
// returned Dispatcher accepts Publish/Close calls but performs no I/O.
func NewDispatcher(cfg Config) *Dispatcher {
	d := &Dispatcher{cfg: cfg}
	if cfg.URL == "" || cfg.Payload == nil {
		return d
	}
	u, err := url.Parse(cfg.URL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		log.Error("Invalid webhook URL; disabling", "err", err)
		return d
	}
	d.host = u.Host

	if cfg.TimeoutSeconds <= 0 {
		d.cfg.TimeoutSeconds = defaultTimeoutSeconds
	}
	if cfg.HTTPClient == nil {
		d.cfg.HTTPClient = &http.Client{Timeout: time.Duration(d.cfg.TimeoutSeconds) * time.Second}
	}

	d.enabled = true
	d.wake = make(chan struct{}, 1)
	d.stop = make(chan struct{})

	d.doneWG.Add(1)
	go d.worker()

	if cfg.HeartbeatSeconds > 0 {
		d.doneWG.Add(1)
		go d.heartbeat(time.Duration(cfg.HeartbeatSeconds) * time.Second)
	}

	log.Info("Webhook dispatcher enabled", "host", d.host, "heartbeat_seconds", cfg.HeartbeatSeconds)
	return d
}

// Publish queues an event for delivery. Non-blocking. Higher-priority events
// (solve > startup > heartbeat) overwrite lower-priority pending events.
func (d *Dispatcher) Publish(event string) {
	if !d.enabled {
		return
	}
	d.mu.Lock()
	if eventPriority(event) > eventPriority(d.pending) {
		d.pending = event
	}
	d.mu.Unlock()
	select {
	case d.wake <- struct{}{}:
	default:
	}
}

// Close stops worker goroutines and waits for them to exit.
func (d *Dispatcher) Close() {
	if !d.enabled {
		return
	}
	close(d.stop)
	d.doneWG.Wait()
}

func eventPriority(e string) int {
	switch e {
	case "solve":
		return 3
	case "startup":
		return 2
	case "heartbeat":
		return 1
	default:
		return 0
	}
}

func (d *Dispatcher) heartbeat(interval time.Duration) {
	defer d.doneWG.Done()
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-d.stop:
			return
		case <-t.C:
			d.Publish("heartbeat")
		}
	}
}

func (d *Dispatcher) worker() {
	defer d.doneWG.Done()
	for {
		select {
		case <-d.stop:
			return
		case <-d.wake:
		}
		for {
			d.mu.Lock()
			ev := d.pending
			d.pending = ""
			d.mu.Unlock()
			if ev == "" {
				break
			}
			d.deliver(ev)
		}
	}
}

func (d *Dispatcher) deliver(event string) {
	payload, err := d.cfg.Payload(event)
	if err != nil {
		log.Warn("Webhook payload build failed", "event", event, "host", d.host, "err", err)
		return
	}
	body, err := json.Marshal(payload)
	if err != nil {
		log.Warn("Webhook payload marshal failed", "event", event, "host", d.host, "err", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(d.cfg.TimeoutSeconds)*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.cfg.URL, bytes.NewReader(body))
	if err != nil {
		log.Warn("Webhook request build failed", "event", event, "host", d.host, "err", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", fmt.Sprintf("honeybear-webhook/%s", d.cfg.Version))

	resp, err := d.cfg.HTTPClient.Do(req)
	if err != nil {
		log.Warn("Webhook POST failed", "event", event, "host", d.host, "err", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Warn("Webhook POST non-2xx", "event", event, "host", d.host, "status", resp.StatusCode)
		return
	}
	log.Debug("Webhook POST ok", "event", event, "host", d.host, "status", resp.StatusCode)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/honeypot/webhook/ -run TestNewDispatcher -v`
Expected: PASS for both tests.

- [ ] **Step 5: Commit**

```bash
git add internal/honeypot/webhook/dispatcher.go internal/honeypot/webhook/dispatcher_test.go
git commit -m "feat(webhook): add generic dispatcher with no-op fallback"
```

---

## Task 3: Dispatcher — single POST end-to-end

**Files:**
- Modify: `internal/honeypot/webhook/dispatcher_test.go`

- [ ] **Step 1: Write the failing test**

Append to `dispatcher_test.go`:

```go
import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestDispatcher_PublishesPayload(t *testing.T) {
	var got struct {
		Body   []byte
		Method string
		CT     string
		UA     string
	}
	done := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.Method = r.Method
		got.CT = r.Header.Get("Content-Type")
		got.UA = r.Header.Get("User-Agent")
		got.Body, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		close(done)
	}))
	defer srv.Close()

	d := NewDispatcher(Config{
		URL:     srv.URL,
		Version: "test",
		Payload: func(event string) (any, error) {
			return map[string]any{"event": event, "n": 1}, nil
		},
	})
	defer d.Close()

	d.Publish("solve")

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("did not receive POST in time")
	}

	if got.Method != http.MethodPost {
		t.Errorf("method = %q, want POST", got.Method)
	}
	if got.CT != "application/json" {
		t.Errorf("content-type = %q", got.CT)
	}
	if got.UA != "honeybear-webhook/test" {
		t.Errorf("user-agent = %q", got.UA)
	}
	var parsed map[string]any
	if err := json.Unmarshal(got.Body, &parsed); err != nil {
		t.Fatalf("body not json: %v", err)
	}
	if parsed["event"] != "solve" {
		t.Errorf("event = %v, want solve", parsed["event"])
	}
}
```

Replace the existing minimal `import "testing"` line at the top with the full import block above (consolidate, do not duplicate).

- [ ] **Step 2: Run test**

Run: `go test ./internal/honeypot/webhook/ -run TestDispatcher_PublishesPayload -v`
Expected: PASS (dispatcher code already implemented in Task 2).

- [ ] **Step 3: Commit**

```bash
git add internal/honeypot/webhook/dispatcher_test.go
git commit -m "test(webhook): verify dispatcher POST shape and headers"
```

---

## Task 4: Dispatcher — coalescing and priority

**Files:**
- Modify: `internal/honeypot/webhook/dispatcher_test.go`

- [ ] **Step 1: Write the failing test**

Append to `dispatcher_test.go`:

```go
func TestDispatcher_CoalescesBurstWhilePOSTInFlight(t *testing.T) {
	release := make(chan struct{})
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		if atomic.LoadInt32(&hits) == 1 {
			<-release // hold the first request open
		}
		_, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	d := NewDispatcher(Config{
		URL:     srv.URL,
		Version: "test",
		Payload: func(event string) (any, error) { return map[string]any{"event": event}, nil },
	})
	defer d.Close()

	d.Publish("heartbeat") // starts in-flight request, server blocks
	// Give the worker a moment to pick it up and start the POST.
	time.Sleep(50 * time.Millisecond)
	for i := 0; i < 20; i++ {
		d.Publish("solve")
	}
	close(release)

	// Wait for the second POST.
	deadline := time.After(2 * time.Second)
	for atomic.LoadInt32(&hits) < 2 {
		select {
		case <-deadline:
			t.Fatalf("expected 2 POSTs, got %d", atomic.LoadInt32(&hits))
		case <-time.After(10 * time.Millisecond):
		}
	}
	// Confirm we didn't fan out one POST per Publish call.
	time.Sleep(100 * time.Millisecond)
	if h := atomic.LoadInt32(&hits); h != 2 {
		t.Fatalf("expected exactly 2 POSTs after coalescing, got %d", h)
	}
}

func TestDispatcher_SolveOverridesPendingHeartbeat(t *testing.T) {
	release := make(chan struct{})
	events := make(chan string, 4)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var p map[string]any
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &p)
		events <- p["event"].(string)
		if len(events) == 1 {
			<-release
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	d := NewDispatcher(Config{
		URL:     srv.URL,
		Version: "test",
		Payload: func(event string) (any, error) { return map[string]any{"event": event}, nil },
	})
	defer d.Close()

	d.Publish("heartbeat") // becomes the first in-flight POST
	time.Sleep(50 * time.Millisecond)
	d.Publish("heartbeat") // queued
	d.Publish("solve")     // should overwrite the queued heartbeat
	close(release)

	first := <-events
	second := <-events
	if first != "heartbeat" || second != "solve" {
		t.Fatalf("got events %q then %q, want heartbeat then solve", first, second)
	}
}
```

- [ ] **Step 2: Run tests**

Run: `go test ./internal/honeypot/webhook/ -run "TestDispatcher_Coalesces|TestDispatcher_Solve" -v`
Expected: PASS for both.

- [ ] **Step 3: Commit**

```bash
git add internal/honeypot/webhook/dispatcher_test.go
git commit -m "test(webhook): verify coalescing and priority override"
```

---

## Task 5: Dispatcher — heartbeat ticker fires events

**Files:**
- Modify: `internal/honeypot/webhook/dispatcher_test.go`

- [ ] **Step 1: Write the failing test**

Append to `dispatcher_test.go`:

```go
func TestDispatcher_HeartbeatFires(t *testing.T) {
	events := make(chan string, 4)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var p map[string]any
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &p)
		events <- p["event"].(string)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// HeartbeatSeconds is in seconds; we override via a test-only client that
	// fires fast. Since the dispatcher uses seconds, use 1 here and tolerate
	// the timing.
	d := NewDispatcher(Config{
		URL:              srv.URL,
		Version:          "test",
		HeartbeatSeconds: 1,
		Payload:          func(event string) (any, error) { return map[string]any{"event": event}, nil },
	})
	defer d.Close()

	select {
	case ev := <-events:
		if ev != "heartbeat" {
			t.Fatalf("first event = %q, want heartbeat", ev)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("no heartbeat POST received within 3s")
	}
}
```

- [ ] **Step 2: Run test**

Run: `go test ./internal/honeypot/webhook/ -run TestDispatcher_HeartbeatFires -v -timeout 10s`
Expected: PASS within ~1-2 seconds.

- [ ] **Step 3: Commit**

```bash
git add internal/honeypot/webhook/dispatcher_test.go
git commit -m "test(webhook): verify heartbeat ticker fires events"
```

---

## Task 6: Dispatcher — timeout does not block worker

**Files:**
- Modify: `internal/honeypot/webhook/dispatcher_test.go`

- [ ] **Step 1: Write the failing test**

Append to `dispatcher_test.go`:

```go
func TestDispatcher_TimeoutDoesNotBlockNextEvent(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&hits, 1)
		if n == 1 {
			time.Sleep(2 * time.Second) // exceeds 1s timeout
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	d := NewDispatcher(Config{
		URL:            srv.URL,
		Version:        "test",
		TimeoutSeconds: 1,
		Payload:        func(event string) (any, error) { return map[string]any{"event": event}, nil },
	})
	defer d.Close()

	d.Publish("solve")
	// Wait for the timeout to elapse, then publish again.
	time.Sleep(1500 * time.Millisecond)
	d.Publish("solve")

	deadline := time.After(3 * time.Second)
	for atomic.LoadInt32(&hits) < 2 {
		select {
		case <-deadline:
			t.Fatalf("expected 2 POST attempts, got %d", atomic.LoadInt32(&hits))
		case <-time.After(20 * time.Millisecond):
		}
	}
}
```

- [ ] **Step 2: Run test**

Run: `go test ./internal/honeypot/webhook/ -run TestDispatcher_TimeoutDoesNotBlockNextEvent -v -timeout 10s`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add internal/honeypot/webhook/dispatcher_test.go
git commit -m "test(webhook): verify timeout does not block subsequent events"
```

---

## Task 7: Leaderboard subpackage

**Files:**
- Create: `internal/honeypot/webhook/leaderboard/leaderboard.go`
- Test: `internal/honeypot/webhook/leaderboard/leaderboard_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/honeypot/webhook/leaderboard/leaderboard_test.go`:

```go
package leaderboard

import (
	"encoding/json"
	"testing"

	"github.com/mikeflynn/honeybearhoneypot/internal/entity"
)

func TestBuildPayload_ProjectsRanks(t *testing.T) {
	src := func(limit int) ([]entity.CTFUser, error) {
		return []entity.CTFUser{
			{Username: "alice", Points: 150},
			{Username: "bob", Points: 90},
		}, nil
	}
	p := newPayloadFunc(src, 10)
	got, err := p("solve")
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	var parsed struct {
		Event       string `json:"event"`
		Source      string `json:"source"`
		Timestamp   string `json:"timestamp"`
		Leaderboard []struct {
			Rank     int    `json:"rank"`
			Username string `json:"username"`
			Points   int    `json:"points"`
		} `json:"leaderboard"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("payload not json: %v", err)
	}
	if parsed.Event != "solve" {
		t.Errorf("event = %q", parsed.Event)
	}
	if parsed.Source != "honeybear" {
		t.Errorf("source = %q", parsed.Source)
	}
	if parsed.Timestamp == "" {
		t.Error("timestamp empty")
	}
	if len(parsed.Leaderboard) != 2 {
		t.Fatalf("len = %d, want 2", len(parsed.Leaderboard))
	}
	if parsed.Leaderboard[0].Rank != 1 || parsed.Leaderboard[0].Username != "alice" || parsed.Leaderboard[0].Points != 150 {
		t.Errorf("row 0 = %+v", parsed.Leaderboard[0])
	}
	if parsed.Leaderboard[1].Rank != 2 || parsed.Leaderboard[1].Username != "bob" {
		t.Errorf("row 1 = %+v", parsed.Leaderboard[1])
	}
}

func TestBuildPayload_EmptyLeaderboardEmitsEmptyArray(t *testing.T) {
	src := func(limit int) ([]entity.CTFUser, error) { return nil, nil }
	p := newPayloadFunc(src, 10)
	got, err := p("heartbeat")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(got)
	var parsed map[string]json.RawMessage
	_ = json.Unmarshal(body, &parsed)
	if string(parsed["leaderboard"]) != "[]" {
		t.Errorf("leaderboard = %s, want []", string(parsed["leaderboard"]))
	}
}

func TestBuildPayload_LimitDefaultsTo10WhenZero(t *testing.T) {
	var gotLimit int
	src := func(limit int) ([]entity.CTFUser, error) {
		gotLimit = limit
		return nil, nil
	}
	p := newPayloadFunc(src, 0)
	if _, err := p("solve"); err != nil {
		t.Fatal(err)
	}
	if gotLimit != 10 {
		t.Errorf("limit passed = %d, want 10", gotLimit)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/honeypot/webhook/leaderboard/ -v`
Expected: FAIL with build errors (`undefined: newPayloadFunc`).

- [ ] **Step 3: Implement leaderboard package**

Create `internal/honeypot/webhook/leaderboard/leaderboard.go`:

```go
// Package leaderboard wires the generic webhook dispatcher to the CTF
// leaderboard. It owns the entity.Leaderboard call and the on-the-wire payload
// shape.
package leaderboard

import (
	"time"

	"github.com/mikeflynn/honeybearhoneypot/internal/config"
	"github.com/mikeflynn/honeybearhoneypot/internal/entity"
	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot/webhook"
)

const defaultLimit = 10

type row struct {
	Rank     int    `json:"rank"`
	Username string `json:"username"`
	Points   int    `json:"points"`
}

type snapshot struct {
	Event       string `json:"event"`
	Timestamp   string `json:"timestamp"`
	Source      string `json:"source"`
	Leaderboard []row  `json:"leaderboard"`
}

// leaderboardSource matches entity.Leaderboard; broken out for testability.
type leaderboardSource func(limit int) ([]entity.CTFUser, error)

func newPayloadFunc(src leaderboardSource, limit int) webhook.PayloadFunc {
	if limit <= 0 {
		limit = defaultLimit
	}
	return func(event string) (any, error) {
		users, err := src(limit)
		if err != nil {
			return nil, err
		}
		rows := make([]row, 0, len(users))
		for i, u := range users {
			rows = append(rows, row{Rank: i + 1, Username: u.Username, Points: u.Points})
		}
		return snapshot{
			Event:       event,
			Timestamp:   time.Now().UTC().Format(time.RFC3339),
			Source:      "honeybear",
			Leaderboard: rows,
		}, nil
	}
}

// New returns a dispatcher configured for the leaderboard webhook. If cfg is
// nil or cfg.URL is empty, returns a no-op dispatcher.
func New(cfg *config.LeaderboardWebhook, version string) *webhook.Dispatcher {
	if cfg == nil {
		return webhook.NewDispatcher(webhook.Config{})
	}
	return webhook.NewDispatcher(webhook.Config{
		URL:              cfg.URL,
		TimeoutSeconds:   cfg.TimeoutSeconds,
		HeartbeatSeconds: cfg.HeartbeatSeconds,
		Version:          version,
		Payload:          newPayloadFunc(entity.Leaderboard, cfg.Limit),
	})
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/honeypot/webhook/leaderboard/ -v`
Expected: all three tests PASS.

- [ ] **Step 5: Run full webhook test suite**

Run: `go test ./internal/honeypot/webhook/...`
Expected: all PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/honeypot/webhook/leaderboard/
git commit -m "feat(webhook): add leaderboard payload builder and constructor"
```

---

## Task 8: CTF integration — package-level publisher accessor

**Files:**
- Modify: `internal/honeypot/ctf/ctf.go`

The CTF model is value-typed (`func (m Model) updateAnswer(...)`) and not constructed with a publisher dependency in the existing code path. A small package-level setter mirrors how the rest of the package picks up config (e.g. `m.user` comes from session state, not DI). This keeps the surface change minimal.

- [ ] **Step 1: Add publisher accessor near the top of ctf.go**

After the existing imports in `internal/honeypot/ctf/ctf.go`, add an import for the webhook package and a package-level publisher:

```go
import (
	// ... existing imports ...
	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot/webhook"
)
```

Add this block somewhere near the top of the file, after imports but before the existing top-level declarations:

```go
// leaderboardPublisher is set by main.go at startup. When nil or when its
// underlying URL is empty, the publish call is a no-op.
var leaderboardPublisher *webhook.Dispatcher

// SetLeaderboardPublisher wires the webhook dispatcher used after successful
// flag submissions. Safe to call once at startup.
func SetLeaderboardPublisher(d *webhook.Dispatcher) {
	leaderboardPublisher = d
}

func publishLeaderboard(event string) {
	if leaderboardPublisher != nil {
		leaderboardPublisher.Publish(event)
	}
}
```

- [ ] **Step 2: Call publishLeaderboard after CompleteTask succeeds**

Locate the existing solve block in `ctf.go` (around line 364–372):

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
```

Immediately after `m.selectedTask.Completed = true`, add:

```go
						publishLeaderboard("solve")
```

- [ ] **Step 3: Verify build and existing tests still pass**

Run: `go build ./... && go test ./internal/honeypot/ctf/...`
Expected: build clean, existing CTF tests PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/honeypot/ctf/ctf.go
git commit -m "feat(ctf): publish leaderboard webhook after successful solve"
```

---

## Task 9: Wire dispatcher in main.go

**Files:**
- Modify: `main.go`

- [ ] **Step 1: Add imports and construct the dispatcher**

Add to the import block in `main.go`:

```go
	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot/ctf"
	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot/webhook/leaderboard"
```

In `main()`, after the line `log.Info("Starting Honey Bear Honey Pot...")` and before `filesystem.SetNoFun(...)`, add:

```go
	var lbCfg *config.LeaderboardWebhook
	if cfg.Webhooks != nil {
		lbCfg = cfg.Webhooks.Leaderboard
	}
	lbDispatcher := leaderboard.New(lbCfg, strings.TrimSpace(versionFile))
	defer lbDispatcher.Close()
	ctf.SetLeaderboardPublisher(lbDispatcher)
	lbDispatcher.Publish("startup")
```

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: clean build.

- [ ] **Step 3: Smoke test end-to-end with a local sink**

In a separate terminal:

```bash
# Tiny stdlib sink that prints any POST body it receives.
cat > /tmp/sink.go <<'EOF'
package main
import ("fmt"; "io"; "net/http")
func main() {
  http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    b, _ := io.ReadAll(r.Body)
    fmt.Printf("%s %s\n%s\n---\n", r.Method, r.URL.Path, string(b))
    w.WriteHeader(200)
  })
  http.ListenAndServe(":18080", nil)
}
EOF
go run /tmp/sink.go
```

In the project terminal:

```bash
go run main.go -no-gui -ssh-port 2222 -log-level debug -leaderboard-webhook-url http://127.0.0.1:18080/lb -leaderboard-webhook-heartbeat 5
```

Expected: sink prints a `startup` payload within ~1s, then a `heartbeat` payload every ~5s. Ctrl-C to stop both processes.

- [ ] **Step 4: Commit**

```bash
git add main.go
git commit -m "feat(main): wire leaderboard webhook dispatcher at startup"
```

---

## Task 10: Update sample config

**Files:**
- Modify: `config.sample.json`

- [ ] **Step 1: Add a webhooks block to the sample config**

Open `config.sample.json` and add a `webhooks` block alongside the other top-level keys (placement matters for JSON validity; ensure a trailing comma on the previous field and no trailing comma after `webhooks`):

```json
  "webhooks": {
    "leaderboard": {
      "url": "https://example.com/honeybear/leaderboard",
      "limit": 10,
      "heartbeat_seconds": 300,
      "timeout_seconds": 5
    }
  }
```

- [ ] **Step 2: Validate JSON parses**

Run: `python3 -c 'import json,sys; json.load(open("config.sample.json"))' && echo ok`
Expected: `ok`.

- [ ] **Step 3: Verify Parse accepts the sample config**

Run: `go run main.go -config config.sample.json -no-gui -ssh-port 2223 -log-level debug` for ~3 seconds, then Ctrl-C.
Expected: startup logs include `"Webhook dispatcher enabled" host=example.com heartbeat_seconds=300`. No fatal errors during parse.

- [ ] **Step 4: Commit**

```bash
git add config.sample.json
git commit -m "docs(config): document webhooks.leaderboard block in sample"
```

---

## Task 11: Final verification

- [ ] **Step 1: Full test suite**

Run: `go test ./...`
Expected: all tests PASS.

- [ ] **Step 2: Build**

Run: `go build ./...`
Expected: clean.

- [ ] **Step 3: Format**

Run: `go fmt ./...`
Expected: no files reformatted (or commit any formatting fixes separately).
