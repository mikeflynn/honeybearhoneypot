package confetti

import (
	"math/rand"
	"time"

	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot/simulation"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/harmonica"
	"github.com/charmbracelet/lipgloss"
)

const (
	framesPerSecond = 30.0
	numParticles    = 75
)

var (
	// Using 256-color palette equivalents for broader terminal support.
	colors     = []string{"141", "81", "119", "211", "228"}
	characters = []string{"█", "▓", "▒", "░", "▄", "▀"}
)

type (
	frameMsg time.Time
	burstMsg time.Time
)

func Burst() tea.Msg {
	return burstMsg(time.Now())
}

func animate() tea.Cmd {
	return tea.Tick(time.Second/framesPerSecond, func(t time.Time) tea.Msg {
		return frameMsg(t)
	})
}

// Confetti model
type Model struct {
	system *simulation.System
}

func Spawn(width, height int) []*simulation.Particle {
	particles := []*simulation.Particle{}
	for i := 0; i < numParticles; i++ {
		x := float64(width / 2)
		y := float64(0)

		p := simulation.Particle{
			Physics: harmonica.NewProjectile(
				harmonica.FPS(framesPerSecond),
				harmonica.Point{X: x + (float64(width/4) * (rand.Float64() - 0.5)), Y: y, Z: 0},
				harmonica.Vector{X: (rand.Float64() - 0.5) * 100, Y: rand.Float64() * 50, Z: 0},
				harmonica.TerminalGravity,
			),
			Char: lipgloss.NewStyle().
				Foreground(lipgloss.Color(arraySample(colors))).
				Render(arraySample(characters)),
		}

		particles = append(particles, &p)
	}
	return particles
}

func InitialModel() Model {
	return Model{system: &simulation.System{
		Particles: []*simulation.Particle{},
		Frame:     simulation.Frame{},
	}}
}

// Init initializes the confetti after a small delay
func (m Model) Init() tea.Cmd {
	return animate()
}

// Update updates the model every frame, it handles the animation loop and
// updates the particle physics every frame
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		m.system.Particles = append(m.system.Particles, Spawn(m.system.Frame.Width, m.system.Frame.Height)...)

		return m, nil
	case frameMsg:
		m.system.Update()
		return m, animate()
	case burstMsg:
		if len(m.system.Particles) == 0 {
			m.system.Particles = Spawn(m.system.Frame.Width, m.system.Frame.Height)
		} else {
			m.system.Particles = append(m.system.Particles, Spawn(m.system.Frame.Width, m.system.Frame.Height)...)
		}

		return m, animate()
	case tea.WindowSizeMsg:
		if m.system.Frame.Width == 0 && m.system.Frame.Height == 0 {
			// For the first frameMsg spawn a system of particles
			m.system.Particles = Spawn(msg.Width, msg.Height)
		}
		m.system.Frame.Width = msg.Width
		m.system.Frame.Height = msg.Height
		return m, nil
	default:
		return m, nil
	}
}

// View displays all the particles on the screen
func (m Model) View() string {
	return m.system.Render()
}

// Sample returns a random element from a generic array
func arraySample[T any](arr []T) T {
	return arr[rand.Intn(len(arr))]
}
