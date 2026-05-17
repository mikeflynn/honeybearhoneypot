package honeypot

import (
	"math/rand"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot/confetti"
	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot/filesystem"
	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot/glitch"
	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot/matrix"
)

var (
	sessionsMu sync.RWMutex
	sessions   = map[string]*tea.Program{}
)

func registerSession(id string, p *tea.Program) {
	sessionsMu.Lock()
	sessions[id] = p
	sessionsMu.Unlock()
}

func unregisterSession(id string) {
	sessionsMu.Lock()
	delete(sessions, id)
	sessionsMu.Unlock()
}

// SessionCount returns the number of registered interactive sessions.
func SessionCount() int {
	sessionsMu.RLock()
	defer sessionsMu.RUnlock()
	return len(sessions)
}

func broadcast(msgs ...tea.Msg) {
	sessionsMu.RLock()
	defer sessionsMu.RUnlock()
	for _, p := range sessions {
		for _, msg := range msgs {
			p.Send(msg)
		}
	}
}

// ActionKnock broadcasts the classic Matrix-style prompt to all sessions.
func ActionKnock() {
	broadcast(filesystem.OutputMsg("Knock, knock, Neo."))
	go func() {
		time.Sleep(10 * time.Second)
		broadcast(filesystem.OutputMsg("If you don't know what to do, type 'help' or 'ctf' to play a game."))
	}()
}

// ActionSystemNotice broadcasts a fake wall-style maintenance message.
func ActionSystemNotice() {
	broadcast(filesystem.OutputMsg("Broadcast message: System going down in 5 minutes for maintenance."))
}

// ActionFakeJoin broadcasts a fake "another user joined" line.
func ActionFakeJoin() {
	broadcast(filesystem.OutputMsg("user 'fozzie' has connected from 10.0.0.42"))
}

// ActionMatrix flips every active session into matrix mode.
func ActionMatrix() {
	broadcast(filesystem.SetRunningCmd("matrix"), matrix.MatrixTick{})
}

// ActionConfetti fires a single burst of confetti on every active session
// and auto-exits after a short window so the user doesn't have to dismiss it.
func ActionConfetti() {
	broadcast(filesystem.SetRunningCmd("confetti"), confetti.Burst())
	go func() {
		time.Sleep(5 * time.Second)
		broadcast(filesystem.SetRunningCmd(""), filesystem.ClearOutputMsg(""))
	}()
}

// ActionKickAll disconnects every active session.
func ActionKickAll() {
	broadcast(tea.Quit())
}

// ActionGlitch broadcasts a short visual glitch effect (~2.5s) to every active
// session, then restores normal rendering. Mirrors the lifecycle of
// ActionConfetti.
func ActionGlitch() {
	broadcast(filesystem.SetRunningCmd("glitch"), glitch.Tick{})
	go func() {
		time.Sleep(glitch.Duration + 100*time.Millisecond)
		broadcast(filesystem.SetRunningCmd(""), filesystem.ClearOutputMsg(""))
	}()
}

// ActionShuffleDirs silently relocates every active session to a randomly
// chosen directory in the fake filesystem. Each session gets an independent
// random destination.
func ActionShuffleDirs() {
	dirs := filesystem.AllDirectories()
	if len(dirs) == 0 {
		return
	}
	sessionsMu.RLock()
	defer sessionsMu.RUnlock()
	for _, p := range sessions {
		d := dirs[rand.Intn(len(dirs))]
		p.Send(filesystem.ChangeDirMsg{Node: d, Path: d.Path, Silent: true})
	}
}
