package glitch

import (
	"math/rand"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	TickInterval = 50 * time.Millisecond
	Duration     = 2500 * time.Millisecond

	corruptionRate = 0.12
	lineShiftRate  = 0.20
	flickerRate    = 0.05
)

var (
	corruptGlyphs = []rune("▓▒░@#%&*")
	flickerStyle  = lipgloss.NewStyle().Reverse(true)
)

// Tick is the message type used to advance the glitch animation each frame.
type Tick struct{}

// New returns a glitch model initialized with the given base screen content.
func New(base string) *Model {
	return &Model{
		base: base,
		rng:  rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Model holds the snapshot of underlying screen content and elapsed time.
type Model struct {
	base    string
	elapsed time.Duration
	rng     *rand.Rand
}

// Update advances the glitch by one tick. Returns the next tick command and a
// done flag that becomes true once the configured Duration is exceeded.
func (m *Model) Update(msg tea.Msg) (tea.Cmd, bool) {
	if _, ok := msg.(Tick); !ok {
		return nil, false
	}
	m.elapsed += TickInterval
	if m.elapsed >= Duration {
		return nil, true
	}
	return tea.Tick(TickInterval, func(time.Time) tea.Msg { return Tick{} }), false
}

// View renders one corrupted frame of the base content.
func (m *Model) View() string {
	if m.base == "" {
		return ""
	}
	lines := strings.Split(m.base, "\n")

	// Line shift.
	for i := range lines {
		if m.rng.Float64() < lineShiftRate {
			if i+1 < len(lines) && m.rng.Intn(2) == 0 {
				lines[i], lines[i+1] = lines[i+1], lines[i]
			} else {
				offset := strings.Repeat(" ", 1+m.rng.Intn(4))
				lines[i] = offset + lines[i]
			}
		}
	}

	// Char corruption.
	for i, line := range lines {
		runes := []rune(line)
		for j, r := range runes {
			if r == ' ' || r == '\t' {
				continue
			}
			if m.rng.Float64() < corruptionRate {
				runes[j] = corruptGlyphs[m.rng.Intn(len(corruptGlyphs))]
			}
		}
		lines[i] = string(runes)
	}

	// Color flicker.
	for i, line := range lines {
		if m.rng.Float64() < flickerRate {
			lines[i] = flickerStyle.Render(line)
		}
	}

	return strings.Join(lines, "\n")
}

// Start returns the initial tick message for the glitch effect.
func Start() tea.Msg {
	return Tick{}
}
