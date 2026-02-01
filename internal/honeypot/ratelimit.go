package honeypot

import (
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/mikeflynn/honeybearhoneypot/internal/config"
)

type clientState struct {
	count       int
	firstSeen   time.Time
	bannedUntil time.Time
}

type RateLimiter struct {
	mu          sync.Mutex
	limits      map[string]*clientState
	window      time.Duration
	maxRequests int
	banDuration time.Duration
}

var limiter *RateLimiter
var once sync.Once

func GetRateLimiter() *RateLimiter {
	once.Do(func() {
		window, _ := time.ParseDuration(config.Active.RateLimitWindow)
		if window == 0 {
			window = 60 * time.Second
		}
		
		banDuration, _ := time.ParseDuration(config.Active.RateLimitBan)
		if banDuration == 0 {
			banDuration = 5 * time.Minute
		}

		maxRequests := config.Active.RateLimitMax
		if maxRequests == 0 {
			maxRequests = 5
		}

		limiter = &RateLimiter{
			limits:      make(map[string]*clientState),
			window:      window,
			maxRequests: maxRequests,
			banDuration: banDuration,
		}
		
		log.Info("Rate limiter initialized", "window", window, "max", maxRequests, "ban", banDuration)
	})
	return limiter
}

// Check returns true if the request is allowed, false if blocked/banned
func (r *RateLimiter) Check(ip string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	state, exists := r.limits[ip]
	now := time.Now()

	if !exists {
		r.limits[ip] = &clientState{
			count:     1,
			firstSeen: now,
		}
		return true
	}

	// Check if currently banned
	if now.Before(state.bannedUntil) {
		return false
	}

	// Check if window expired, reset if so
	if now.Sub(state.firstSeen) > r.window {
		state.count = 1
		state.firstSeen = now
		state.bannedUntil = time.Time{} // Clear ban
		return true
	}

	// Increment count
	state.count++

	// Check if limit exceeded
	if state.count > r.maxRequests {
		state.bannedUntil = now.Add(r.banDuration)
		log.Warn("IP banned by rate limiter", "ip", ip, "duration", r.banDuration)
		return false
	}

	return true
}
