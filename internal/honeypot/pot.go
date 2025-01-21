package honeypot

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/accesscontrol"
	"github.com/charmbracelet/wish/activeterm"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/elapsed"
	"github.com/charmbracelet/wish/logging"
	"github.com/google/shlex"
	"github.com/mikeflynn/hardhat-honeybear/internal/honeypot/confetti"
	"github.com/mikeflynn/hardhat-honeybear/internal/honeypot/embedded"
	"github.com/mikeflynn/hardhat-honeybear/internal/honeypot/filesystem"
	"github.com/muesli/reflow/wordwrap"
)

const (
	host     = "localhost"
	port     = "2222"
	maxUsers = 10
)

var (
	activeUsers int
)

func StartHoneyPot() {
	activeUsers = 0

	s, err := wish.NewServer(
		wish.WithAddress(net.JoinHostPort(host, port)),
		wish.WithHostKeyPath(".ssh/id_ed25519"),
		wish.WithPasswordAuth(func(ctx ssh.Context, password string) bool {
			log.Info(fmt.Sprintf("Authorization used: %s, %s", ctx.User(), password))
			return true
		}),
		wish.WithBannerHandler(func(ctx ssh.Context) string {
			banner, err := embedded.Files.ReadFile("banner.txt")
			if err == nil || banner != nil {
				return fmt.Sprintf(string(banner), ctx.User())
			}

			return ""
		}),
		wish.WithMiddleware(
			bubbletea.Middleware(teaHandler),
			activeterm.Middleware(), // Bubble Tea apps usually require a PTY.
			accesscontrol.Middleware(),
			logging.Middleware(),
			func(next ssh.Handler) ssh.Handler {
				return func(s ssh.Session) {
					activeUsers++
					next(s)
					activeUsers--
					if activeUsers < 0 {
						activeUsers = 0
					}

				}
			},
			elapsed.Middleware(),
		),
	)
	if err != nil {
		log.Error("Could not start server", "error", err)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	log.Info("Starting SSH server", "host", host, "port", port)
	go func() {
		if err = s.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			log.Error("Could not start server", "error", err)
			done <- nil
		}
	}()

	<-done
	log.Info("Stopping SSH server")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer func() { cancel() }()
	if err := s.Shutdown(ctx); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		log.Error("Could not stop server", "error", err)
	}
}

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
	textinput.Placeholder = "Enter your command."
	textinput.Focus()
	textinput.CharLimit = 200
	textinput.Width = 50
	textinput.Prompt = "$ "
	textinput.Cursor.Style = txtStyle.Background(lipgloss.Color("10"))
	textinput.PromptStyle = txtStyle
	textinput.TextStyle = txtStyle

	filesystem.Initialize()

	m := model{
		user:          s.Context().User(),
		host:          s.Context().RemoteAddr().String(),
		group:         "default",
		term:          pty.Term,
		currentDir:    filesystem.HomeDir,
		profile:       renderer.ColorProfile().Name(),
		width:         pty.Window.Width,
		height:        pty.Window.Height,
		txtStyle:      txtStyle,
		quitStyle:     quitStyle,
		outputStyle:   outputStyle,
		historyStyle:  historyStyle,
		viewportReady: false,
		textInput:     textinput,
		confetti:      confetti.InitialModel(),
		output:        "",
		helpText:      "Type 'help' to see some commands; Use up/down for history.",
		historyIdx:    0,
		history:       []string{},
	}

	return m, []tea.ProgramOption{
		//tea.WithAltScreen(),
	}
}

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
	// Data
	output string
	// History
	historyIdx int
	history    []string
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd   tea.Cmd
		vpCmd tea.Cmd
		cmds  []tea.Cmd
	)

	switch msg := msg.(type) {
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
			m.historyIdx = 0
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

				if parts[0] == "exit" {
					return m, tea.Quit
				}

				newCmd, err := filesystem.RunNode(m.currentDir, parts[0], parts[1:], m.user, m.group)
				if err != nil {
					m.output += m.outputStyle.Render(fmt.Sprintf("\n%s\n", err))
				} else if newCmd != nil {
					cmds = append(cmds, *newCmd)
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
