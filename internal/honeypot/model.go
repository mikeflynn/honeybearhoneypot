package honeypot

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/google/shlex"
	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot/confetti"
	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot/filesystem"
	"github.com/muesli/reflow/wordwrap"
)

// Just a generic tea.Model to demo terminal information of ssh.
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
	txtStyle     lipgloss.Style
	quitStyle    lipgloss.Style
	historyStyle lipgloss.Style
	outputStyle  lipgloss.Style
	// UX & Sub-Models
	textInput     textinput.Model
	viewport      viewport.Model
	viewportReady bool
	confetti      tea.Model
	helpText      string
	events        map[string]time.Time
	// Data
	output string
	// History
	historyIdx int
	history    []string
}

func (m model) Init() tea.Cmd {
	NewEvent(&m, true, "login", "Logged in!")
	return doTick()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd   tea.Cmd
		vpCmd tea.Cmd
		cmds  []tea.Cmd
	)

	switch msg := msg.(type) {
	case filesystem.TickMsg:
		cmds := []tea.Cmd{}
		if time.Since(*m.EventTime("session_start")) > 3*time.Minute && m.EventTime("knock") == nil {
			m.SetEventTime("knock")
			cmds = append(
				cmds,
				tea.Cmd(func() tea.Msg {
					return filesystem.ClearOutputMsg("")
				}),
				tea.Cmd(func() tea.Msg {
					time.Sleep(time.Second * 1)
					return filesystem.OutputMsg("Knock, knock, Neo.")
				}),
			)
		}

		cmds = append(cmds, doTick())
		return m, tea.Batch(cmds...)
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width

		footerHeight := lipgloss.Height(m.quitStyle.Render("\n"))
		inputHeight := lipgloss.Height(m.textInput.View())
		verticalMargin := footerHeight + inputHeight

		if !m.viewportReady {
			m.viewport = viewport.New(msg.Width, msg.Height-verticalMargin)
			m.viewport.YPosition = 0
			m.viewport.HighPerformanceRendering = false
			m.viewport.Style = m.outputStyle.Border(lipgloss.NormalBorder(), false, false, true, false)
			m.viewport.SetContent("")
			m.viewportReady = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMargin
		}

		m.confetti.Update(msg)

		//cmds = append(cmds, viewport.Sync(m.viewport))
	case filesystem.FileContentsMsg:
		wrapper := wordwrap.NewWriter(m.width - 10)
		wrapper.Newline = []rune{'\r'}
		wrapper.Breakpoints = []rune{':', ','}
		wrapper.Write(msg)
		wrapper.Close()

		m.viewport.SetContent(m.outputStyle.Render(wrapper.String()))
		m.viewport.GotoTop()
	case filesystem.SetRunningCmd:
		m.runningCommand = string(msg)
	case filesystem.OutputMsg:
		m.output += m.outputStyle.Render("\n" + string(msg) + "\n")
	case filesystem.ListActiveUsersMsg:
		m.output += fmt.Sprintf("04:25:58 up 10 days, 23:21,  %d users,  load average: 0.10, 0.18, 0.10\n", len(activeUsers))
		m.output += fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s\t%s\n", "USER", "TTY", "FROM", "LOGIN@", "IDLE", "JCPU", "PCPU WHAT")
		for i, u := range activeUsers {
			m.output += fmt.Sprintf("%s\tpts/%d\t%s\t%s\t%s\t%s\t%s\n", u, i, "--", "--", "--", "--", "--")
		}
	case filesystem.ClearOutputMsg:
		m.output = ""
	case filesystem.ChangeDirMsg:
		m.currentDir = msg.Node
		m.output += m.outputStyle.Render(fmt.Sprintf("\ncd %s\n", msg.Path))
	case filesystem.HistoryListMsg:
		max := 10
		if len(m.history) < 10 {
			max = len(m.history)
		}

		list := m.history[:max]

		for i := len(list) - 1; i >= 0; i-- {
			m.output += m.outputStyle.Render(fmt.Sprintf("\n%d: %s", i, list[i]))
		}

		m.output += m.outputStyle.Render("\n")
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			command := m.textInput.Value()
			log.Debug(fmt.Sprintf("Command entered by %s:%s: %s", m.user, m.host, command))
			m.historyIdx = 0
			m.SetEventTime("enter")
			m.output += m.historyStyle.Render(fmt.Sprintf("\n❯ %s\n", m.textInput.Value()))

			parts, err := shlex.Split(command)
			if err != nil {
				m.output += m.outputStyle.Render(fmt.Sprintf("\nError parsing command: %s\n", err))
				return m, tea.Batch(cmds...)
			}

			if len(parts) > 0 {
				// Add to history
				historyPush(&m, command)
				// Save an event log
				err := NewEvent(&m, true, "typed", command)
				if err != nil {
					log.Printf("Error saving event: %s", err)
				}

				switch parts[0] {
				case "exit":
					return m, tea.Quit
				case "whoami":
					m.output += m.outputStyle.Render(fmt.Sprintf("\n%s\n", m.user))
				case "sudo":
					if len(parts) > 1 {
						newCmd, err := filesystem.RunNode(m.currentDir, parts[1], parts[2:], "root", "root")
						if err != nil {
							m.output += m.outputStyle.Render(fmt.Sprintf("\n%s\n", err))
						} else if newCmd != nil {
							cmds = append(cmds, *newCmd)
						}
					}
				default:
					newCmd, err := filesystem.RunNode(m.currentDir, parts[0], parts[1:], m.user, m.group)
					if err != nil {
						m.output += m.outputStyle.Render(fmt.Sprintf("\n%s\n", err))
					} else if newCmd != nil {
						cmds = append(cmds, *newCmd)
					}
				}
			}

			m.textInput.Reset()
			return m, tea.Batch(cmds...)
		case "up":
			if m.runningCommand == "" {
				m.textInput.SetValue(historyPeek(&m))
				historyIdxInc(&m)
			}
		case "down":
			if m.runningCommand == "" {
				m.textInput.SetValue(historyPeek(&m))
				historyIdxDec(&m)
			}
		case "ctrl+c":
			if m.runningCommand != "" {
				m.viewport.SetContent("")
				m.runningCommand = ""
			}

			return m, tea.Batch(cmds...)
		}
	}

	switch m.runningCommand {
	case "cat":
		m.viewport, vpCmd = m.viewport.Update(msg)
		cmds = append(cmds, vpCmd)
	case "confetti":
		np, cmd := m.confetti.Update(msg)
		cp, ok := np.(confetti.Model)
		if !ok {
			return m, tea.Quit
		}
		m.confetti = cp
		cmds = append(cmds, cmd)
	default:
		m.textInput, cmd = m.textInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if !m.viewportReady {
		return "\nInitializing...\n"
	}

	footerHeight := lipgloss.Height(m.quitStyle.Render("\n"))
	inputHeight := lipgloss.Height(m.textInput.View())
	contentHeight := m.height - footerHeight - inputHeight

	//s := fmt.Sprintf("Your term is %s\nYour window size is %dx%d\nBackground: %s", m.term, m.width, m.height, m.bg)

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
	}

	return fmt.Sprintf("%s\n%s\n%s\n", content, m.textInput.View(), m.quitStyle.Render(help))
}

func (m model) EventTime(event string) *time.Time {
	if t, ok := m.events[event]; ok {
		return &t
	}

	return nil
}

func (m model) SetEventTime(event string) {
	m.events[event] = time.Now()
}
