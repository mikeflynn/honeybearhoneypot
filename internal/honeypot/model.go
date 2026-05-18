package honeypot

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
	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot/glitch"
	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot/matrix"
	"github.com/muesli/reflow/wordwrap"
)

// Just a generic tea.Model to demo terminal information of ssh.
type model struct {
	// Session
	user           string
	password       string
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
	glitchModel   *glitch.Model
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

func (m model) Init() tea.Cmd {
	NewEvent(&m, true, "login", m.password)
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
		return m, doTick()
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width

		footerHeight := lipgloss.Height(m.quitStyle.Render("\n"))
		inputHeight := lipgloss.Height(m.textInput.View())
		verticalMargin := footerHeight + inputHeight

		if !m.viewportReady {
			m.viewport = viewport.New(viewport.WithWidth(msg.Width), viewport.WithHeight(msg.Height-verticalMargin))
			m.viewport.SetYOffset(0)
			//m.viewport.HighPerformanceRendering = false
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
	case glitch.Tick:
		if m.runningCommand == "glitch" {
			if m.glitchModel == nil {
				m.glitchModel = glitch.New(m.output)
			}
			cmd, done := m.glitchModel.Update(msg)
			if done {
				m.glitchModel = nil
			} else if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	case filesystem.ListActiveUsersMsg:
		users := activeUsersSnapshot()
		m.output += fmt.Sprintf("04:25:58 up 10 days, 23:21,  %d users,  load average: 0.10, 0.18, 0.10\n", len(users))
		m.output += fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s\t%s\n", "USER", "TTY", "FROM", "LOGIN@", "IDLE", "JCPU", "PCPU WHAT")
		for i, u := range users {
			m.output += fmt.Sprintf("%s\tpts/%d\t%s\t%s\t%s\t%s\t%s\n", u, i, "--", "--", "--", "--", "--")
		}
	case filesystem.ClearOutputMsg:
		m.output = ""
	case filesystem.SetEnvMsg:
		if m.environ == nil {
			m.environ = make(map[string]string)
		}
		m.environ[msg.Key] = msg.Value
	case filesystem.UnsetEnvMsg:
		if m.environ != nil {
			delete(m.environ, string(msg))
		}
	case filesystem.ChangeDirMsg:
		m.currentDir = msg.Node
		if m.environ == nil {
			m.environ = make(map[string]string)
		}
		m.environ["PWD"] = msg.Path
		if !msg.Silent {
			m.output += m.outputStyle.Render(fmt.Sprintf("\ncd %s\n", msg.Path))
		}
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
	case ctf.QuitMsg:
		m.viewport.SetContent("")
		m.runningCommand = ""
		m.ctf = ctf.InitialModel(convertTasks(config.Active.Tasks), m.user, m.host)
		return m, nil
	case tea.KeyPressMsg:
		switch msg.String() {
		case "enter":
			if m.runningCommand == "" {
				command := m.textInput.Value()
				log.Debug(fmt.Sprintf("Command entered by %s:%s: %s", m.user, m.host, command))
				m.historyIdx = 0
				m.SetEventTime("enter")
				m.output += m.historyStyle.Render(fmt.Sprintf("\n❯ %s\n", m.textInput.Value()))

				// Preserving backslashes that shlex would otherwise strip in double quotes
				// by escaping them for shlex.
				re := regexp.MustCompile(`\\([^$"\\` + "`" + `\n])`)
				escapedCommand := re.ReplaceAllString(command, `\\$0`)

				parts, err := shlex.Split(escapedCommand)
				if err != nil {
					m.output += m.outputStyle.Render(fmt.Sprintf("\nError parsing command: %s\n", err))
					return m, tea.Batch(cmds...)
				}

				if len(parts) > 0 {
					// Expand environment variables
					for i := range parts {
						parts[i] = os.Expand(parts[i], func(k string) string {
							return m.environ[k]
						})
					}

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
					default:
						newCmd, err := filesystem.RunNode(m.currentDir, parts[0], parts[1:], m.user, m.group, m.environ)
						if err != nil {
							m.output += m.outputStyle.Render(fmt.Sprintf("\n%s\n", err))
						} else if newCmd != nil {
							cmds = append(cmds, *newCmd)
						}
					}
				}

				m.textInput.Reset()
				return m, tea.Batch(cmds...)
			}
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
		case "tab":
			if m.runningCommand == "" {
				val := m.textInput.Value()
				if strings.TrimSpace(val) == "" {
					break
				}

				newVal, matches := filesystem.AutoComplete(m.currentDir, val)
				if len(matches) == 1 || (len(matches) > 1 && len(newVal) > len(val)) {
					m.textInput.SetValue(newVal)
					m.textInput.SetCursor(len(newVal))
				} else if len(matches) > 1 && len(matches) < 25 {
					m.output += m.outputStyle.Render(fmt.Sprintf("\n%s\n", strings.Join(matches, "  ")))
				}
			}
		case "ctrl+c":
			if m.runningCommand != "" {
				if m.runningCommand == "matrix" {
					m.matrix, _ = m.matrix.Update(matrix.MatrixStop{})
				}

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
		m.confetti, cmd = m.confetti.Update(msg)
		cmds = append(cmds, cmd)
	case "matrix":
		m.matrix, cmd = m.matrix.Update(msg)
		cmds = append(cmds, cmd)
	case "ctf":
		m.ctf, cmd = m.ctf.Update(msg)
		cmds = append(cmds, cmd)
	default:
		m.textInput, cmd = m.textInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() tea.View {
	if !m.viewportReady {
		return tea.NewView("\nInitializing...\n")
	}

	footerHeight := lipgloss.Height(m.quitStyle.Render("\n"))
	inputHeight := lipgloss.Height(m.textInput.View())
	contentHeight := m.height - footerHeight - inputHeight

	//s := fmt.Sprintf("Your term is %s\nYour window size is %dx%d\nBackground: %s", m.term, m.width, m.height, m.bg)

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
	} else if m.runningCommand == "glitch" {
		if m.glitchModel != nil {
			content = m.glitchModel.View()
		}
		help = ""
	} else if m.runningCommand == "ctf" {
		return tea.NewView("" +
			m.ctf.View() +
			"\n" +
			m.quitStyle.Render("esc to go back or ctrl + c to exit the ctf.\n"))
	}

	return tea.NewView(fmt.Sprintf("%s\n%s\n%s\n", content, m.textInput.View(), m.quitStyle.Render(help)))
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
