package honeypot

import (
	"time"

	"github.com/mikeflynn/honeybearhoneypot/internal/entity"
)

func DefaultEnviron(user, term string) map[string]string {
	home := "/home/" + user
	if user == "root" {
		home = "/root"
	}

	env := map[string]string{
		"USER":    user,
		"LOGNAME": user,
		"HOME":    home,
		"SHELL":   "/bin/bash",
		"PATH":    "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		"PWD":     home,
		"LANG":    "en_US.UTF-8",
	}

	if term != "" {
		env["TERM"] = term
	}

	return env
}

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

	event.Publish()
	return event.Save()
}
