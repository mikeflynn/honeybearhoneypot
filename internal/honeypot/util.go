package honeypot

import (
	"time"

	"github.com/mikeflynn/hardhat-honeybear/internal/entity"
)

func historyPush(m *model, command string) {
	// Prepends a command to the history slice.
	m.history = append([]string{command}, m.history...)
}

func historyPeek(m *model) string {
	if m.historyIdx >= len(m.history) {
		return ""
	}

	return m.history[m.historyIdx]
}

func historyIdxInc(m *model) {
	if m.historyIdx >= len(m.history)-1 {
		return
	}

	m.historyIdx++
}

func historyIdxDec(m *model) {
	if m.historyIdx == 0 {
		return
	}

	m.historyIdx--
}

func NewEvent(m *model, userEvent bool, eventType string, eventAction string) error {
	source := entity.EventSourceSystem
	if userEvent {
		source = entity.EventSourceUser
	}

	event := &entity.Event{
		User:      m.user,
		Host:      m.host,
		App:       "ssh",
		Source:    source,
		Type:      eventType,
		Action:    eventAction,
		Timestamp: time.Now(),
	}

	return event.Save()
}
