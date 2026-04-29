package honeypot

import (
	"sync"
	"time"

	"charm.land/log/v2"
	"github.com/mikeflynn/honeybearhoneypot/internal/entity"
)

var (
	usersAllTimeCacheValue int
	usersAllTimeCacheTTL   time.Time
	usersAllTimeMu         sync.Mutex
	usersAllTimeUpdating   bool
)

func StatActiveUsers() int {
	return activeUsersLen()
}

func StatUsersThisSession() int {
	return usersThisSessionCount()
}

func StatUsersAllTime() int {
	usersAllTimeMu.Lock()
	val := usersAllTimeCacheValue
	ttl := usersAllTimeCacheTTL
	updating := usersAllTimeUpdating
	usersAllTimeMu.Unlock()

	now := time.Now()
	if now.After(ttl) && !updating {
		usersAllTimeMu.Lock()
		usersAllTimeUpdating = true
		usersAllTimeMu.Unlock()

		go func() {
			defer func() {
				usersAllTimeMu.Lock()
				usersAllTimeUpdating = false
				usersAllTimeMu.Unlock()
			}()

			data, err := entity.EventCountQuery(`
				SELECT
					"logins" AS Value,
					COUNT(*) AS Count
				FROM events
				WHERE
					events.type = 'login'
					AND events.source = 'user'
			`)

			usersAllTimeMu.Lock()
			defer usersAllTimeMu.Unlock()

			if err != nil {
				log.Error("statUsersAllTime", "error", err)
				// extend TTL slightly on error to avoid rapid retry
				usersAllTimeCacheTTL = time.Now().Add(5 * time.Second)
				return
			}

			if len(data) == 0 {
				usersAllTimeCacheValue = 0
			} else {
				usersAllTimeCacheValue = data[0].Count
			}
			usersAllTimeCacheTTL = time.Now().Add(10 * time.Second)
		}()
	}

	return val
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
