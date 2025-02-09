package honeypot

import (
	"github.com/mikeflynn/hardhat-honeybear/internal/entity"
)

func StatActiveUsers() int {
	return activeUsers
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
