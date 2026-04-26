# Charmbracelet V2 Library Upgrade Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Upgrade bubbletea, bubbles, lipgloss, wish, and log from their GitHub v1 paths to `charm.land` v2 vanity imports with zero functionality changes.

**Architecture:** All five charmbracelet libraries move from `github.com/charmbracelet/*` to `charm.land/*/v2`. The main breaking changes are: `MakeRenderer` removal (styles become standalone), `View()` returns `tea.View` instead of `string`, `tea.KeyMsg` becomes `tea.KeyPressMsg`, viewport/textinput APIs switch to getter/setter methods, and `AdaptiveColor` is replaced by plain `Color` (all existing usages already have identical light/dark values). Sub-models (confetti, matrix, ctf) are changed from `tea.Model` interface to concrete types with `View() string` (matching the bubbles v2 component pattern), since they are embedded components not top-level programs.

**Tech Stack:** Go 1.24+, charm.land/bubbletea/v2, charm.land/bubbles/v2, charm.land/lipgloss/v2, charm.land/wish/v2, charm.land/log/v2

**Important:** The project will NOT compile until all tasks 1-9 are complete. The v1 and v2 import paths are mutually exclusive. Do not attempt `go build` or `go test` until after Task 9.

---

### Task 1: Update go.mod Dependencies

**Files:**
- Modify: `go.mod`

- [ ] **Step 1: Remove old charmbracelet dependencies and add v2 modules**

```bash
cd /Users/mikeflynn/Code/Personal/hardhat/honeybearhoneypot
go get charm.land/bubbletea/v2@latest
go get charm.land/bubbles/v2@latest
go get charm.land/lipgloss/v2@latest
go get charm.land/wish/v2@latest
go get charm.land/log/v2@latest
```

Note: This will fail or produce warnings until code imports are updated. That is expected. The go.mod will be fully resolved after all code changes and a final `go mod tidy` in Task 10.

---

### Task 2: Update Log-Only Files (Import Path Only)

These files only import `charmbracelet/log` and need only an import path change.

**Files:**
- Modify: `main.go`
- Modify: `internal/gui/window.go`
- Modify: `internal/gui/admin.go`
- Modify: `internal/gui/admin_stats.go`
- Modify: `internal/gui/bears.go`
- Modify: `internal/honeypot/stats.go`
- Modify: `internal/honeypot/tunnel.go`
- Modify: `internal/honeypot/ratelimit.go`
- Modify: `internal/honeypot/filesystem/extra.go`

- [ ] **Step 1: In every file listed above, replace the log import**

In each file, find:
```go
"github.com/charmbracelet/log"
```

Replace with:
```go
"charm.land/log/v2"
```

The `log` package alias stays the same. No other changes needed in these files.

---

### Task 3: Update simulation Package (Import Path Only)

**Files:**
- Modify: `internal/honeypot/simulation/simulation.go`

- [ ] **Step 1: Update lipgloss import**

Find:
```go
"github.com/charmbracelet/lipgloss"
```

Replace with:
```go
"charm.land/lipgloss/v2"
```

No other changes. The `lipgloss.Color` and `lipgloss.Style` types are used identically.

---

### Task 4: Update confetti Sub-Model

Remove `*lipgloss.Renderer` dependency. Change from `tea.Model` interface to concrete component pattern.

**Files:**
- Modify: `internal/honeypot/confetti/confetti.go`

- [ ] **Step 1: Update imports**

Replace the import block:
```go
import (
	"math/rand"
	"time"

	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot/simulation"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/harmonica"
	"github.com/charmbracelet/lipgloss"
)
```

With:
```go
import (
	"math/rand"
	"time"

	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot/simulation"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/harmonica"
	"charm.land/lipgloss/v2"
)
```

- [ ] **Step 2: Remove renderer from Model struct and update Spawn**

Replace the `Model` struct:
```go
type Model struct {
	system   *simulation.System
	renderer *lipgloss.Renderer
}
```

With:
```go
type Model struct {
	system *simulation.System
}
```

Replace the `Spawn` function signature and body — change `renderer *lipgloss.Renderer` parameter to nothing and use `lipgloss.NewStyle()` instead of `renderer.NewStyle()`:

```go
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
```

- [ ] **Step 3: Update InitialModel to drop renderer parameter**

Replace:
```go
func InitialModel(renderer *lipgloss.Renderer) Model {
	return Model{
		system: &simulation.System{
			Particles: []*simulation.Particle{},
			Frame:     simulation.Frame{},
		},
		renderer: renderer,
	}
}
```

With:
```go
func InitialModel() Model {
	return Model{
		system: &simulation.System{
			Particles: []*simulation.Particle{},
			Frame:     simulation.Frame{},
		},
	}
}
```

- [ ] **Step 4: Update Update method — change return type and remove renderer from Spawn calls**

Replace the full `Update` method:
```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		m.system.Particles = append(m.system.Particles, Spawn(m.renderer, m.system.Frame.Width, m.system.Frame.Height)...)

		return m, nil
	case frameMsg:
		m.system.Update()
		return m, animate()
	case burstMsg:
		if len(m.system.Particles) == 0 {
			m.system.Particles = Spawn(m.renderer, m.system.Frame.Width, m.system.Frame.Height)
		} else {
			m.system.Particles = append(m.system.Particles, Spawn(m.renderer, m.system.Frame.Width, m.system.Frame.Height)...)
		}

		return m, animate()
	case tea.WindowSizeMsg:
		if m.system.Frame.Width == 0 && m.system.Frame.Height == 0 {
			// For the first frameMsg spawn a system of particles
			m.system.Particles = Spawn(m.renderer, msg.Width, msg.Height)
		}
		m.system.Frame.Width = msg.Width
		m.system.Frame.Height = msg.Height
		return m, nil
	default:
		return m, nil
	}
}
```

With:
```go
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
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
			m.system.Particles = Spawn(msg.Width, msg.Height)
		}
		m.system.Frame.Width = msg.Width
		m.system.Frame.Height = msg.Height
		return m, nil
	default:
		return m, nil
	}
}
```

Key changes: return type `(Model, tea.Cmd)` instead of `(tea.Model, tea.Cmd)`, `tea.KeyMsg` -> `tea.KeyPressMsg`, removed renderer from `Spawn()` calls.

---

### Task 5: Update matrix Sub-Model

Remove `*lipgloss.Renderer` dependency. Change to concrete component return types.

**Files:**
- Modify: `internal/honeypot/matrix/matrix.go`

- [ ] **Step 1: Update imports**

Replace:
```go
import (
	"math/rand"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)
```

With:
```go
import (
	"math/rand"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)
```

- [ ] **Step 2: Remove renderer from Matrix struct and InitialModel**

Replace the `Matrix` struct:
```go
type Matrix struct {
	Speed  time.Duration
	Width  int
	Height int

	renderer *lipgloss.Renderer
	symbols  [][]string
	colors   [][]int
}
```

With:
```go
type Matrix struct {
	Speed  time.Duration
	Width  int
	Height int

	symbols [][]string
	colors  [][]int
}
```

Replace `InitialModel`:
```go
func InitialModel(renderer *lipgloss.Renderer, width int, height int) Matrix {
	m := Matrix{
		Speed:    time.Millisecond * 100,
		Width:    width,
		Height:   height,
		renderer: renderer,
	}

	return m.initSymbols()
}
```

With:
```go
func InitialModel(width int, height int) Matrix {
	m := Matrix{
		Speed:  time.Millisecond * 100,
		Width:  width,
		Height: height,
	}

	return m.initSymbols()
}
```

- [ ] **Step 3: Update Update method return type**

Replace:
```go
func (m Matrix) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
```

With:
```go
func (m Matrix) Update(msg tea.Msg) (Matrix, tea.Cmd) {
```

No other changes to the Update body are needed. The `tea.WindowSizeMsg` and internal message types are unchanged.

- [ ] **Step 4: Update View to use lipgloss.NewStyle() instead of renderer.NewStyle()**

Replace the first line of the View method:
```go
style := m.renderer.NewStyle().Background(matrixBg)
```

With:
```go
style := lipgloss.NewStyle().Background(matrixBg)
```

---

### Task 6: Update ctf Sub-Model

Remove `*lipgloss.Renderer`, update textinput style API, update key message type.

**Files:**
- Modify: `internal/honeypot/ctf/ctf.go`

- [ ] **Step 1: Update imports**

Replace:
```go
import (
	"fmt"
	"strings"
	"time"

	"unicode"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mikeflynn/honeybearhoneypot/internal/entity"
)
```

With:
```go
import (
	"fmt"
	"strings"
	"time"

	"unicode"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/mikeflynn/honeybearhoneypot/internal/entity"
)
```

- [ ] **Step 2: Remove renderer from Model struct**

In the `Model` struct, remove the `renderer` field:
```go
renderer *lipgloss.Renderer
```

- [ ] **Step 3: Update InitialModel — remove renderer parameter, update textinput style API**

Replace the full `InitialModel` function:
```go
func InitialModel(renderer *lipgloss.Renderer, tasks []Task, sshuser string, sshhost string) Model {
	ti := textinput.New()
	ti.Prompt = "Username: "
	//ti.Placeholder = ""
	ti.Focus()
	ti.CharLimit = 32
	ti.Cursor.Style = renderer.NewStyle().Blink(true)
	ti.Width = 16

	pi := textinput.New()
	pi.Prompt = "Password: "
	//pi.Placeholder = ""
	pi.CharLimit = 32
	pi.EchoMode = textinput.EchoPassword
	pi.Cursor.Style = renderer.NewStyle().Blink(true)
	pi.Width = 16

	ai := textinput.New()
	ai.Placeholder = "flag"
	ai.Prompt = "❯ "
	ai.PromptStyle = renderer.NewStyle().Foreground(lipgloss.Color("5")).Bold(true)
	ai.TextStyle = renderer.NewStyle().Foreground(lipgloss.Color("11"))
	ai.CharLimit = 256
	ai.Width = 16

	return Model{
		state:         stateLogin,
		usernameInput: ti,
		passwordInput: pi,
		answerInput:   ai,
		tasks:         tasks,
		width:         0,
		height:        0,
		cursor:        0,
		sshuser:       sshuser,
		sshhost:       sshhost,
		renderer:      renderer,
	}
}
```

With:
```go
func InitialModel(tasks []Task, sshuser string, sshhost string) Model {
	ti := textinput.New()
	ti.Prompt = "Username: "
	ti.CharLimit = 32
	ti.SetWidth(16)

	styles := ti.Styles()
	styles.Cursor.Blink = true
	ti.SetStyles(styles)
	ti.Focus()

	pi := textinput.New()
	pi.Prompt = "Password: "
	pi.CharLimit = 32
	pi.EchoMode = textinput.EchoPassword
	pi.SetWidth(16)

	pStyles := pi.Styles()
	pStyles.Cursor.Blink = true
	pi.SetStyles(pStyles)

	ai := textinput.New()
	ai.Placeholder = "flag"
	ai.Prompt = "❯ "
	ai.CharLimit = 256
	ai.SetWidth(16)

	aStyles := ai.Styles()
	aStyles.Focused.Prompt = lipgloss.NewStyle().Foreground(lipgloss.Color("5")).Bold(true)
	aStyles.Focused.Text = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	aStyles.Blurred.Prompt = lipgloss.NewStyle().Foreground(lipgloss.Color("5")).Bold(true)
	aStyles.Blurred.Text = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	ai.SetStyles(aStyles)

	return Model{
		state:         stateLogin,
		usernameInput: ti,
		passwordInput: pi,
		answerInput:   ai,
		tasks:         tasks,
		width:         0,
		height:        0,
		cursor:        0,
		sshuser:       sshuser,
		sshhost:       sshhost,
	}
}
```

- [ ] **Step 4: Update Update method return type**

Replace:
```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
```

With:
```go
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
```

- [ ] **Step 5: Update all tea.KeyMsg to tea.KeyPressMsg**

In `updateLogin`, `updateMenu`, and `updateAnswer`, replace every:
```go
case tea.KeyMsg:
```

With:
```go
case tea.KeyPressMsg:
```

These occur at lines 226, 260, and 290. The `msg.String()` calls remain valid.

Also update the return types of `updateLogin`, `updateMenu`, and `updateAnswer`:

Replace each:
```go
func (m Model) updateLogin(msg tea.Msg) (tea.Model, tea.Cmd) {
func (m Model) updateMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
func (m Model) updateAnswer(msg tea.Msg) (tea.Model, tea.Cmd) {
```

With:
```go
func (m Model) updateLogin(msg tea.Msg) (Model, tea.Cmd) {
func (m Model) updateMenu(msg tea.Msg) (Model, tea.Cmd) {
func (m Model) updateAnswer(msg tea.Msg) (Model, tea.Cmd) {
```

- [ ] **Step 6: Update View and renderTasks — replace renderer.NewStyle() with lipgloss.NewStyle()**

In the `View()` method, replace every `m.renderer.NewStyle()` with `lipgloss.NewStyle()`. There are 6 occurrences in `View()`:

```go
titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
welcome := lipgloss.NewStyle().Foreground(lipgloss.Color("7")).Render(...)
```

And in `stateAnswer`:
```go
descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
box := lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Padding(1, 2)
```

In `updateAnswer`, replace the two `m.renderer.NewStyle()` calls:
```go
m.errMsg = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true).Render(...)
m.errMsg = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true).Render("Incorrect flag")
```

In `renderTasks`, replace all `m.renderer.NewStyle()` calls (6 occurrences) with `lipgloss.NewStyle()`:
```go
bullet := lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Render("🍯")
doneBullet := lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render("✓")
selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("0")).Background(lipgloss.Color("6")).Bold(true)
normalStyle := lipgloss.NewStyle()
descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
```

---

### Task 7: Update filesystem Packages

**Files:**
- Modify: `internal/honeypot/filesystem/filesystem.go`
- Modify: `internal/honeypot/filesystem/exec.go`
- Modify: `internal/honeypot/filesystem/node.go`

- [ ] **Step 1: Update imports in node.go**

Replace:
```go
tea "github.com/charmbracelet/bubbletea"
```

With:
```go
tea "charm.land/bubbletea/v2"
```

- [ ] **Step 2: Update imports in filesystem.go**

Replace:
```go
tea "github.com/charmbracelet/bubbletea"
"github.com/charmbracelet/lipgloss"
```

With:
```go
tea "charm.land/bubbletea/v2"
"charm.land/lipgloss/v2"
```

- [ ] **Step 3: Update imports in exec.go**

Replace:
```go
tea "github.com/charmbracelet/bubbletea"
"github.com/charmbracelet/lipgloss"
```

With:
```go
tea "charm.land/bubbletea/v2"
"charm.land/lipgloss/v2"
```

No other changes needed in exec.go. The `lipgloss.NewStyle()` calls (not renderer-bound) are already in the correct form.

---

### Task 8: Update Main Honeypot Model

Update viewport API, View return type, key message type, and sub-model types.

**Files:**
- Modify: `internal/honeypot/model.go`

- [ ] **Step 1: Update imports**

Replace:
```go
import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/google/shlex"
	"github.com/mikeflynn/honeybearhoneypot/internal/config"
	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot/confetti"
	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot/ctf"
	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot/filesystem"
	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot/matrix"
	"github.com/muesli/reflow/wordwrap"
)
```

With:
```go
import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"charm.land/log/v2"
	"github.com/google/shlex"
	"github.com/mikeflynn/honeybearhoneypot/internal/config"
	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot/confetti"
	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot/ctf"
	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot/filesystem"
	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot/matrix"
	"github.com/muesli/reflow/wordwrap"
)
```

- [ ] **Step 2: Update model struct — remove renderer, change sub-model types**

Replace the model struct:
```go
type model struct {
	// Session
	user           string
	host           string
	group          string
	term           string
	profile        string
	width          int
	height         int
	runningCommand string
	currentDir     *filesystem.Node
	// Styles
	renderer     *lipgloss.Renderer
	txtStyle     lipgloss.Style
	quitStyle    lipgloss.Style
	historyStyle lipgloss.Style
	outputStyle  lipgloss.Style
	// UX & Sub-Models
	textInput     textinput.Model
	viewport      viewport.Model
	viewportReady bool
	confetti      tea.Model
	matrix        tea.Model
	ctf           tea.Model
	helpText      string
	events        map[string]time.Time
	// Data
	output string
	// History
	historyIdx int
	history    []string
	// Environment
	environ map[string]string
}
```

With:
```go
type model struct {
	// Session
	user           string
	host           string
	group          string
	term           string
	width          int
	height         int
	runningCommand string
	currentDir     *filesystem.Node
	// Styles
	txtStyle     lipgloss.Style
	quitStyle    lipgloss.Style
	historyStyle lipgloss.Style
	outputStyle  lipgloss.Style
	// UX & Sub-Models
	textInput     textinput.Model
	viewport      viewport.Model
	viewportReady bool
	confetti      confetti.Model
	matrix        matrix.Matrix
	ctf           ctf.Model
	helpText      string
	events        map[string]time.Time
	// Data
	output string
	// History
	historyIdx int
	history    []string
	// Environment
	environ map[string]string
}
```

Removed: `profile`, `renderer`. Changed: `confetti`, `matrix`, `ctf` from `tea.Model` to concrete types.

- [ ] **Step 3: Update the Update method — viewport API, key messages, sub-model calls**

In the `tea.WindowSizeMsg` case, replace the viewport initialization and field access:

Replace:
```go
case tea.WindowSizeMsg:
	m.height = msg.Height
	m.width = msg.Width

	footerHeight := lipgloss.Height(m.quitStyle.Render("\n"))
	inputHeight := lipgloss.Height(m.textInput.View())
	verticalMargin := footerHeight + inputHeight

	if !m.viewportReady {
		m.viewport = viewport.New(msg.Width, msg.Height-verticalMargin)
		m.viewport.YPosition = 0
		//m.viewport.HighPerformanceRendering = false
		m.viewport.Style = m.outputStyle.Border(lipgloss.NormalBorder(), false, false, true, false)
		m.viewport.SetContent("")
		m.viewportReady = true
	} else {
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - verticalMargin
	}

	m.confetti.Update(msg)
	m.matrix.Update(msg)
	m.ctf.Update(msg)
```

With:
```go
case tea.WindowSizeMsg:
	m.height = msg.Height
	m.width = msg.Width

	footerHeight := lipgloss.Height(m.quitStyle.Render("\n"))
	inputHeight := lipgloss.Height(m.textInput.View())
	verticalMargin := footerHeight + inputHeight

	if !m.viewportReady {
		m.viewport = viewport.New(viewport.WithWidth(msg.Width), viewport.WithHeight(msg.Height-verticalMargin))
		m.viewport.SetYOffset(0)
		m.viewport.Style = m.outputStyle.Border(lipgloss.NormalBorder(), false, false, true, false)
		m.viewport.SetContent("")
		m.viewportReady = true
	} else {
		m.viewport.SetWidth(msg.Width)
		m.viewport.SetHeight(msg.Height - verticalMargin)
	}

	m.confetti, _ = m.confetti.Update(msg)
	m.matrix, _ = m.matrix.Update(msg)
	m.ctf, _ = m.ctf.Update(msg)
```

- [ ] **Step 4: Update the ctf QuitMsg handler**

Replace:
```go
case ctf.QuitMsg:
	m.viewport.SetContent("")
	m.runningCommand = ""
	m.ctf = ctf.InitialModel(m.renderer, convertTasks(config.Active.Tasks), m.user, m.host)
	return m, nil
```

With:
```go
case ctf.QuitMsg:
	m.viewport.SetContent("")
	m.runningCommand = ""
	m.ctf = ctf.InitialModel(convertTasks(config.Active.Tasks), m.user, m.host)
	return m, nil
```

- [ ] **Step 5: Update tea.KeyMsg to tea.KeyPressMsg**

Replace:
```go
case tea.KeyMsg:
```

With:
```go
case tea.KeyPressMsg:
```

The `msg.String()` calls and all key string matching (`"enter"`, `"up"`, `"down"`, `"tab"`, `"ctrl+c"`) remain valid.

- [ ] **Step 6: Update sub-model Update calls in the runningCommand switch**

Replace the confetti case:
```go
case "confetti":
	np, cmd := m.confetti.Update(msg)
	cp, ok := np.(confetti.Model)
	if !ok {
		return m, tea.Quit
	}
	m.confetti = cp
	cmds = append(cmds, cmd)
```

With:
```go
case "confetti":
	m.confetti, cmd = m.confetti.Update(msg)
	cmds = append(cmds, cmd)
```

Replace the matrix case:
```go
case "matrix":
	matrixModel, cmd := m.matrix.Update(msg)
	mm, ok := matrixModel.(matrix.Matrix)
	if !ok {
		return m, tea.Quit
	}
	m.matrix = mm

	cmds = append(cmds, tea.Batch(
		//m.matrix.Init(),
		cmd,
	))
```

With:
```go
case "matrix":
	m.matrix, cmd = m.matrix.Update(msg)
	cmds = append(cmds, cmd)
```

Replace the ctf case:
```go
case "ctf":
	np, cmd := m.ctf.Update(msg)
	m.ctf = np
	cmds = append(cmds, cmd)
```

With:
```go
case "ctf":
	m.ctf, cmd = m.ctf.Update(msg)
	cmds = append(cmds, cmd)
```

- [ ] **Step 7: Update View() to return tea.View**

Replace the entire `View()` method:
```go
func (m model) View() string {
	if !m.viewportReady {
		return "\nInitializing...\n"
	}

	footerHeight := lipgloss.Height(m.quitStyle.Render("\n"))
	inputHeight := lipgloss.Height(m.textInput.View())
	contentHeight := m.height - footerHeight - inputHeight

	content := m.txtStyle.Width(m.width - 4).Height(contentHeight).Render(lipgloss.PlaceVertical(contentHeight-2, lipgloss.Top, m.output))
	help := m.helpText

	if m.runningCommand == "cat" && m.viewportReady {
		m.viewport.Height = m.height - footerHeight

		return "" +
			m.viewport.View() +
			"\n" +
			m.quitStyle.Render("ctrl + c to exit this file.\n")
	} else if m.runningCommand == "confetti" {
		content = m.confetti.View()
		help = "Press 'q' to quit or any other key to make more confetti."
	} else if m.runningCommand == "matrix" {
		content = m.matrix.View()
		help = "Press 'ctrl + c' to quit."
	} else if m.runningCommand == "ctf" {
		return "" +
			m.ctf.View() +
			"\n" +
			m.quitStyle.Render("esc to go back or ctrl + c to exit the ctf.\n")
	}

	return fmt.Sprintf("%s\n%s\n%s\n", content, m.textInput.View(), m.quitStyle.Render(help))
}
```

With:
```go
func (m model) View() tea.View {
	if !m.viewportReady {
		return tea.NewView("\nInitializing...\n")
	}

	footerHeight := lipgloss.Height(m.quitStyle.Render("\n"))
	inputHeight := lipgloss.Height(m.textInput.View())
	contentHeight := m.height - footerHeight - inputHeight

	content := m.txtStyle.Width(m.width - 4).Height(contentHeight).Render(lipgloss.PlaceVertical(contentHeight-2, lipgloss.Top, m.output))
	help := m.helpText

	if m.runningCommand == "cat" && m.viewportReady {
		m.viewport.SetHeight(m.height - footerHeight)

		return tea.NewView("" +
			m.viewport.View() +
			"\n" +
			m.quitStyle.Render("ctrl + c to exit this file.\n"))
	} else if m.runningCommand == "confetti" {
		content = m.confetti.View()
		help = "Press 'q' to quit or any other key to make more confetti."
	} else if m.runningCommand == "matrix" {
		content = m.matrix.View()
		help = "Press 'ctrl + c' to quit."
	} else if m.runningCommand == "ctf" {
		return tea.NewView("" +
			m.ctf.View() +
			"\n" +
			m.quitStyle.Render("esc to go back or ctrl + c to exit the ctf.\n"))
	}

	return tea.NewView(fmt.Sprintf("%s\n%s\n%s\n", content, m.textInput.View(), m.quitStyle.Render(help)))
}
```

Key changes: return type `tea.View`, all return statements wrapped in `tea.NewView()`, `m.viewport.Height = x` -> `m.viewport.SetHeight(x)`. Sub-model `.View()` calls return `string` (concrete types), so string concatenation still works.

---

### Task 9: Update pot.go — Handler, Renderer Removal, Styles

**Files:**
- Modify: `internal/honeypot/pot.go`

- [ ] **Step 1: Update imports**

Replace:
```go
import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/activeterm"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/elapsed"
	"github.com/charmbracelet/wish/logging"
	"github.com/mikeflynn/honeybearhoneypot/internal/config"
	"github.com/mikeflynn/honeybearhoneypot/internal/entity"
	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot/confetti"
	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot/ctf"
	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot/embedded"
	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot/filesystem"
	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot/matrix"
)
```

With:
```go
import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"charm.land/log/v2"
	"github.com/charmbracelet/ssh"
	"charm.land/wish/v2"
	"charm.land/wish/v2/activeterm"
	"charm.land/wish/v2/bubbletea"
	"charm.land/wish/v2/elapsed"
	"charm.land/wish/v2/logging"
	"github.com/mikeflynn/honeybearhoneypot/internal/config"
	"github.com/mikeflynn/honeybearhoneypot/internal/entity"
	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot/confetti"
	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot/ctf"
	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot/embedded"
	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot/filesystem"
	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot/matrix"
)
```

Note: `github.com/charmbracelet/ssh` stays on its GitHub path (wish v2 uses it from GitHub too).

- [ ] **Step 2: Update teaHandler — remove MakeRenderer, replace AdaptiveColor, update textinput API**

Replace the entire `teaHandler` function:
```go
func teaHandler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
	// This should never fail, as we are using the activeterm middleware.
	pty, _, _ := s.Pty()

	renderer := bubbletea.MakeRenderer(s)

	txtStyle := renderer.NewStyle().Foreground(lipgloss.AdaptiveColor{
		Light: "10", // Light green
		Dark:  "10", // Light green
	})
	outputStyle := renderer.NewStyle().Foreground(lipgloss.AdaptiveColor{
		Light: "8",   // Light grey
		Dark:  "246", // Dark grey
	})
	historyStyle := renderer.NewStyle().Bold(true).Foreground(lipgloss.AdaptiveColor{
		Light: "#c33", // C33 Red
		Dark:  "#c33", // C33 Red
	})
	quitStyle := renderer.NewStyle().Foreground(lipgloss.AdaptiveColor{
		Light: "246", // Dark grey
		Dark:  "8",   // Light grey
	})

	textinput := textinput.New()
	textinput.Placeholder = ""
	textinput.Focus()
	textinput.CharLimit = 200
	textinput.Width = 50
	textinput.Prompt = "you@hbhphh.hhb.com $ "
	textinput.Cursor.Style = txtStyle.Background(lipgloss.Color("10"))
	textinput.PromptStyle = txtStyle
	textinput.TextStyle = txtStyle

	initialOutput := ""
	if out, err := embedded.Files.ReadFile("initial-output.txt"); err == nil {
		initialOutput = string(out)
	}

	filesystem.Initialize()

	user := s.Context().User()

	m := model{
		user:          user,
		host:          s.Context().RemoteAddr().String(),
		group:         "default",
		term:          pty.Term,
		currentDir:    filesystem.HomeDir,
		profile:       renderer.ColorProfile().Name(),
		width:         pty.Window.Width,
		height:        pty.Window.Height,
		renderer:      renderer,
		txtStyle:      txtStyle,
		quitStyle:     quitStyle,
		outputStyle:   outputStyle,
		historyStyle:  historyStyle,
		viewportReady: false,
		textInput:     textinput,
		events: map[string]time.Time{
			"session_start": time.Now(),
		},
		environ: DefaultEnviron(user, pty.Term),
		confetti: confetti.InitialModel(renderer),
		matrix:   matrix.InitialModel(renderer, pty.Window.Width, pty.Window.Height),
		ctf: ctf.InitialModel(
			renderer,
			convertTasks(config.Active.Tasks),
			s.Context().User(),
			s.Context().RemoteAddr().String(),
		),
		output:     initialOutput,
		helpText:   "Type 'help' to see some commands; Use up/down for history.",
		historyIdx: 0,
		history:    []string{},
	}

	return m, []tea.ProgramOption{
		//tea.WithAltScreen(),
	}
}
```

With:
```go
func teaHandler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
	// This should never fail, as we are using the activeterm middleware.
	pty, _, _ := s.Pty()

	txtStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	outputStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("246"))
	historyStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#c33"))
	quitStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	ti := textinput.New()
	ti.Placeholder = ""
	ti.CharLimit = 200
	ti.SetWidth(50)
	ti.Prompt = "you@hbhphh.hhb.com $ "

	tiStyles := ti.Styles()
	tiStyles.Cursor.Color = lipgloss.Color("10")
	tiStyles.Focused.Prompt = txtStyle
	tiStyles.Focused.Text = txtStyle
	tiStyles.Blurred.Prompt = txtStyle
	tiStyles.Blurred.Text = txtStyle
	ti.SetStyles(tiStyles)
	ti.Focus()

	initialOutput := ""
	if out, err := embedded.Files.ReadFile("initial-output.txt"); err == nil {
		initialOutput = string(out)
	}

	filesystem.Initialize()

	user := s.Context().User()

	m := model{
		user:          user,
		host:          s.Context().RemoteAddr().String(),
		group:         "default",
		term:          pty.Term,
		currentDir:    filesystem.HomeDir,
		width:         pty.Window.Width,
		height:        pty.Window.Height,
		txtStyle:      txtStyle,
		quitStyle:     quitStyle,
		outputStyle:   outputStyle,
		historyStyle:  historyStyle,
		viewportReady: false,
		textInput:     ti,
		events: map[string]time.Time{
			"session_start": time.Now(),
		},
		environ:  DefaultEnviron(user, pty.Term),
		confetti: confetti.InitialModel(),
		matrix:   matrix.InitialModel(pty.Window.Width, pty.Window.Height),
		ctf: ctf.InitialModel(
			convertTasks(config.Active.Tasks),
			s.Context().User(),
			s.Context().RemoteAddr().String(),
		),
		output:     initialOutput,
		helpText:   "Type 'help' to see some commands; Use up/down for history.",
		historyIdx: 0,
		history:    []string{},
	}

	return m, bubbletea.MakeOptions(s)
}
```

Key changes:
- `MakeRenderer(s)` removed entirely
- `AdaptiveColor{Light: "10", Dark: "10"}` simplified to `lipgloss.Color("10")` (values were always identical)
- For `outputStyle`, chose `"246"` (the dark variant, appropriate for typical terminal backgrounds)
- For `quitStyle`, chose `"8"` (the dark variant)
- `renderer.NewStyle()` -> `lipgloss.NewStyle()`
- `textinput.Width = 50` -> `ti.SetWidth(50)`
- `textinput.Cursor.Style`, `textinput.PromptStyle`, `textinput.TextStyle` -> `SetStyles()` API
- Removed `profile` and `renderer` from model init
- Sub-model constructors no longer take renderer
- Return `bubbletea.MakeOptions(s)` instead of empty options (handles SSH I/O)

---

### Task 10: Update exec.go

**Files:**
- Modify: `internal/honeypot/exec.go`

- [ ] **Step 1: Update imports**

Replace:
```go
import (
	"fmt"
	"net"
	"os"
	"regexp"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/google/shlex"
	"github.com/mikeflynn/honeybearhoneypot/internal/entity"
	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot/filesystem"
)
```

With:
```go
import (
	"fmt"
	"net"
	"os"
	"regexp"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/log/v2"
	"github.com/charmbracelet/ssh"
	"github.com/google/shlex"
	"github.com/mikeflynn/honeybearhoneypot/internal/entity"
	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot/filesystem"
)
```

No other changes needed. The `tea.BatchMsg` type still exists in v2 with the same `[]tea.Cmd` definition.

---

### Task 11: Resolve Dependencies and Verify Build

- [ ] **Step 1: Run go mod tidy to clean up dependencies**

```bash
cd /Users/mikeflynn/Code/Personal/hardhat/honeybearhoneypot
go mod tidy
```

This will remove old `github.com/charmbracelet/{bubbletea,bubbles,lipgloss,wish,log}` entries and resolve the new `charm.land/*` dependencies.

Expected: completes without errors. If there are unresolved import issues, the error messages will indicate which files still reference old paths.

- [ ] **Step 2: Build the project**

```bash
go build ./...
```

Expected: clean build with no errors. If there are compilation errors, they will point to API mismatches that need fixing.

- [ ] **Step 3: Run all tests**

```bash
go test ./...
```

Expected output (3 packages with tests should pass):
```
ok  	github.com/mikeflynn/honeybearhoneypot/internal/db/export
ok  	github.com/mikeflynn/honeybearhoneypot/internal/honeypot
ok  	github.com/mikeflynn/honeybearhoneypot/internal/honeypot/filesystem
```

- [ ] **Step 4: Run the application to verify it starts**

```bash
go run main.go -no-gui -ssh-port 2222 -log-level debug &
sleep 2
# Test SSH connection
ssh -o StrictHostKeyChecking=no -p 2222 test@localhost
```

Verify: the app starts, the SSH banner appears, you can type commands (`ls`, `help`, `ctf`), and ctrl+c exits cleanly.

- [ ] **Step 5: Commit all changes**

```bash
git add -A
git commit -m "Upgrade charmbracelet libraries to v2 (charm.land vanity imports)"
```

---

## Notes

- **AdaptiveColor simplification:** Every `AdaptiveColor` usage in this codebase had identical Light and Dark values (e.g., `{Light: "10", Dark: "10"}`). These are replaced with plain `lipgloss.Color("10")`. For `outputStyle` and `quitStyle` which had different values, we picked the dark-theme variant since most terminal users use dark themes and the SSH honeypot targets technical users.
- **Sub-model pattern change:** `confetti.Model`, `matrix.Matrix`, and `ctf.Model` are no longer stored as `tea.Model` interface. They return their concrete types from `Update()` and `string` from `View()`. This matches the bubbles v2 component pattern and eliminates type assertions.
- **muesli/reflow:** This dependency is unchanged. It has no coupling to the charmbracelet v2 migration.
- **charmbracelet/ssh:** Stays on its GitHub import path. Wish v2 also imports it from GitHub.
- **charmbracelet/harmonica:** Stays on its GitHub import path. No v2 migration needed.
