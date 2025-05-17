package honeypot

import (
	"github.com/mikeflynn/honeybearhoneypot/internal/entity"
)

func StatActiveUsers() int {
	return len(activeUsers)
}

func StatUsersThisSession() int {
	return usersThisSession
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
