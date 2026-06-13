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
