package honeypot

import (
	"strconv"
	"sync"
	"time"

	"charm.land/log/v2"
	"github.com/mikeflynn/honeybearhoneypot/internal/entity"
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
		limiter = &RateLimiter{
			limits: make(map[string]*clientState),
		}
		limiter.Reload()
	})
	return limiter
}

func (r *RateLimiter) Reload() {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Window
	windowStr := entity.OptionGet(entity.KeyRateLimitWindow)
	var window time.Duration
	if val, err := strconv.Atoi(windowStr); err == nil && val > 0 {
		window = time.Duration(val) * time.Second
	} else {
		window = 1 * time.Second
	}

	// Ban Duration
	banStr := entity.OptionGet(entity.KeyRateLimitBan)
	var banDuration time.Duration
	if val, err := strconv.Atoi(banStr); err == nil && val > 0 {
		banDuration = time.Duration(val) * time.Second
	} else {
		banDuration = 1 * time.Second
	}

	// Max Requests
	maxStr := entity.OptionGet(entity.KeyRateLimitMax)
	var maxRequests int
	if maxStr != "" {
		maxRequests, _ = strconv.Atoi(maxStr)
	}
	if maxRequests == 0 {
		maxRequests = 999999
	}

	r.window = window
	r.maxRequests = maxRequests
	r.banDuration = banDuration

	log.Info("Rate limiter configuration reloaded", "window", window, "max", maxRequests, "ban", banDuration)
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
