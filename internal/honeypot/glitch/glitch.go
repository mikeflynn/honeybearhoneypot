package glitch

import (
	"math/rand"
	"regexp"
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

	// maxLines caps the working window so View() stays O(constant) per frame
	// regardless of how much scrollback the session has accumulated.
	maxLines = 200
)

var (
	corruptGlyphs = []rune("▓▒░@#%&*")
	flickerStyle  = lipgloss.NewStyle().Reverse(true)

	// ansiRe matches CSI escape sequences (the form lipgloss emits for color
	// and style). Used to mask rune positions so corruption never lands inside
	// an escape and breaks the terminal.
	ansiRe = regexp.MustCompile("\x1b\\[[0-9;]*[a-zA-Z]")
)

// Tick is the message type used to advance the glitch animation each frame.
type Tick struct{}

// New returns a glitch model initialized with the given base screen content.
// The base is split into lines once and capped at maxLines so subsequent
// frames are cheap regardless of session history length.
func New(base string) *Model {
	lines := strings.Split(base, "\n")
	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}
	return &Model{
		baseLines: lines,
		rng:       rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Model holds the snapshot of underlying screen content and elapsed time.
type Model struct {
	baseLines []string
	elapsed   time.Duration
	rng       *rand.Rand
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

// View renders one corrupted frame derived from the snapshot. A fresh copy of
// the base lines is used each frame so corruption never accumulates.
func (m *Model) View() string {
	if len(m.baseLines) == 0 {
		return ""
	}
	lines := make([]string, len(m.baseLines))
	copy(lines, m.baseLines)

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

	// Char corruption, ANSI-aware.
	for i, line := range lines {
		lines[i] = corruptLine(line, m.rng)
	}

	// Color flicker.
	for i, line := range lines {
		if m.rng.Float64() < flickerRate {
			lines[i] = flickerStyle.Render(line)
		}
	}

	return strings.Join(lines, "\n")
}

// corruptLine replaces a fraction of non-whitespace runes with random glyphs,
// skipping any byte ranges that belong to ANSI CSI escape sequences so the
// terminal styling stays intact.
func corruptLine(line string, rng *rand.Rand) string {
	if line == "" {
		return line
	}

	// Mark byte offsets that fall inside an ANSI escape; runes whose starting
	// byte is masked are left alone.
	var masked []bool
	if matches := ansiRe.FindAllStringIndex(line, -1); len(matches) > 0 {
		masked = make([]bool, len(line))
		for _, m := range matches {
			for b := m[0]; b < m[1]; b++ {
				masked[b] = true
			}
		}
	}

	var b strings.Builder
	b.Grow(len(line))
	byteIdx := 0
	for _, r := range line {
		size := len(string(r))
		skip := r == ' ' || r == '\t' || (masked != nil && masked[byteIdx])
		if !skip && rng.Float64() < corruptionRate {
			b.WriteRune(corruptGlyphs[rng.Intn(len(corruptGlyphs))])
		} else {
			b.WriteRune(r)
		}
		byteIdx += size
	}
	return b.String()
}

// Start returns the initial tick message for the glitch effect.
func Start() tea.Msg {
	return Tick{}
}
