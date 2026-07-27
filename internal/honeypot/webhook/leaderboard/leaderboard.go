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
	// A non-positive limit means "send the whole board". entity.Leaderboard
	// forwards it to SQLite, where LIMIT -1 returns every row uncapped.
	if limit <= 0 {
		limit = -1
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
