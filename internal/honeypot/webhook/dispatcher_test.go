package webhook

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

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
