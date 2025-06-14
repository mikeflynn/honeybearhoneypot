package honeypot

import (
	"time"

	"github.com/charmbracelet/log"
	"github.com/mikeflynn/honeybearhoneypot/internal/entity"
)

var (
	usersAllTimeCacheValue int
	usersAllTimeCacheTTL   time.Time
)

func StatActiveUsers() int {
	return activeUsersLen()
}

func StatUsersThisSession() int {
	return usersThisSessionCount()
}

func StatUsersAllTime() int {
	now := time.Now()
	if now.Before(usersAllTimeCacheTTL) {
		return usersAllTimeCacheValue
	}

	data, err := entity.EventCountQuery(`
		SELECT
			"logins" AS Value,
			COUNT(*) AS Count
		FROM events
		WHERE
			events.type = 'login'
			AND events.source = 'user'
	`)
	if err != nil {
		log.Error("statUsersAllTime", "error", err)
		return usersAllTimeCacheValue // fallback to last cached value
	}

	if len(data) == 0 {
		usersAllTimeCacheValue = 0
	} else {
		usersAllTimeCacheValue = data[0].Count
	}
	usersAllTimeCacheTTL = now.Add(10 * time.Second)
	return usersAllTimeCacheValue
}

func StatMaxUsers() int {
	// This should up updated to subscribe to the options change
	// and update the value on change rather than querying the database each time.
	maxUsers := entity.OptionGetInt(entity.KeyPotMaxUsers)

	if maxUsers == 0 {
		return defaultMaxUsers
	}

	return maxUsers
}

func StatTunnelActive() int {
	return tunnelActive
}
