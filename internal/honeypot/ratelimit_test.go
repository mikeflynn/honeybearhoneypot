package honeypot

import (
	"testing"
	"time"

	"github.com/mikeflynn/honeybearhoneypot/internal/config"
)

func TestRateLimiter(t *testing.T) {
	// Mock config
	config.Active = &config.Config{
		RateLimitWindow: 1,
		RateLimitMax:    2,
		RateLimitBan:    2,
	}

	// Just directly create a limiter for testing to avoid global state mess if possible,
	// but GetRateLimiter uses global config. So we set config above.
	// Let's just create a new RateLimiter manually for testing logic.
	
	rl := &RateLimiter{
		limits:      make(map[string]*clientState),
		window:      1 * time.Second,
		maxRequests: 2,
		banDuration: 2 * time.Second,
	}

	ip := "127.0.0.1"

	// 1. First request - Allowed
	if !rl.Check(ip) {
		t.Errorf("First request should be allowed")
	}

	// 2. Second request - Allowed
	if !rl.Check(ip) {
		t.Errorf("Second request should be allowed")
	}

	// 3. Third request - Blocked (Max is 2)
	if rl.Check(ip) {
		t.Errorf("Third request should be blocked")
	}

	// 4. Fourth request - Still Blocked
	if rl.Check(ip) {
		t.Errorf("Fourth request should be blocked")
	}

	// Wait for ban to expire
	time.Sleep(2100 * time.Millisecond)

	// 5. Fifth request - Allowed (Ban expired)
	if !rl.Check(ip) {
		t.Errorf("Request after ban should be allowed")
	}
}

func TestRateLimiterWindowReset(t *testing.T) {
	rl := &RateLimiter{
		limits:      make(map[string]*clientState),
		window:      1 * time.Second,
		maxRequests: 2,
		banDuration: 5 * time.Second,
	}

	ip := "192.168.1.1"

	// 1. Request
	rl.Check(ip)
	// 2. Request
	rl.Check(ip)

	// Wait for window to expire (but not ban, as we didn't hit limit yet)
	time.Sleep(1100 * time.Millisecond)

	// 3. Request - Should be allowed and reset count
	if !rl.Check(ip) {
		t.Errorf("Request after window should be allowed")
	}

	// Internal check
	rl.mu.Lock()
	if rl.limits[ip].count != 1 {
		t.Errorf("Count should be reset to 1, got %d", rl.limits[ip].count)
	}
	rl.mu.Unlock()
}
