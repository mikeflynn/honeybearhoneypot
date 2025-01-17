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
	"github.com/mikeflynn/hardhat-honeybear/internal/db"
)

const (
	host     = "localhost"
	port     = "2222"
	maxUsers = 10
)

var (
	activeUsers int
)

type fileContentsMsg string

func StartHoneyPot() {
	activeUsers = 0

	s, err := wish.NewServer(
		wish.WithAddress(net.JoinHostPort(host, port)),
		wish.WithHostKeyPath(".ssh/id_ed25519"),
		wish.WithPasswordAuth(func(ctx ssh.Context, password string) bool {
			log.Info(fmt.Sprintf("Authorization used: %s, %s", ctx.User(), password))
			return true
		}),
		//wish.WithBannerHandler(func(ctx ssh.Context) string {
		//	return fmt.Sprintf(banner, ctx.User())
		//}),
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
	outputStyle := renderer.NewStyle().MaxHeight(pty.Window.Height).Foreground(lipgloss.AdaptiveColor{
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

	m := model{
		user:         s.Context().User(),
		host:         s.Context().RemoteAddr().String(),
		term:         pty.Term,
		profile:      renderer.ColorProfile().Name(),
		width:        pty.Window.Width,
		height:       pty.Window.Height,
		txtStyle:     txtStyle,
		quitStyle:    quitStyle,
		outputStyle:  outputStyle,
		historyStyle: historyStyle,
		ready:        false,
		textInput:    textinput,
		output:       "",
		helpText:     "Type 'help' to see some commands; Use up/down for history.",
		historyIdx:   0,
		history:      []string{},
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
	term           string
	profile        string
	width          int
	height         int
	runningCommand string
	// Styles
	txtStyle     lipgloss.Style
	quitStyle    lipgloss.Style
	historyStyle lipgloss.Style
	outputStyle  lipgloss.Style
	// UX
	textInput textinput.Model
	viewport  viewport.Model
	ready     bool
	helpText  string
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

		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-verticalMargin)
			m.viewport.YPosition = 0
			m.viewport.HighPerformanceRendering = false
			m.viewport.Style = m.outputStyle.Border(lipgloss.NormalBorder(), false, false, true, false)
			m.viewport.SetContent("")
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMargin
		}

		//cmds = append(cmds, viewport.Sync(m.viewport))
	case fileContentsMsg:
		m.viewport.SetContent(string(msg))
		m.viewport.GotoTop()
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			command := m.textInput.Value()
			m.historyIdx = 0
			m.output += m.historyStyle.Render(fmt.Sprintf("\n❯ %s\n", m.textInput.Value()))

			if command != "" {
				historyPush(&m, command)

				// Save an event log
				err := NewEvent(&m, true, "typed", command)
				if err != nil {
					log.Printf("Error saving event: %s", err)
				}

				if command == "exit" {
					return m, tea.Quit
				}

				newCmd := globalCommandHandler(&m)
				if newCmd != nil {
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

	// Allow viewport to track arrow keys
	m.viewport, vpCmd = m.viewport.Update(msg)
	cmds = append(cmds, vpCmd)

	// The rest of the text can go in to the input.q
	m.textInput, cmd = m.textInput.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if !m.ready {
		return "\nInitializing...\n"
	}

	footerHeight := lipgloss.Height(m.quitStyle.Render("\n"))
	inputHeight := lipgloss.Height(m.textInput.View())
	contentHeight := m.height - footerHeight - inputHeight

	//s := fmt.Sprintf("Your term is %s\nYour window size is %dx%d\nBackground: %s", m.term, m.width, m.height, m.bg)

	content := m.txtStyle.Width(m.width - 4).Height(contentHeight).Render(lipgloss.PlaceVertical(contentHeight-2, lipgloss.Top, m.output))

	if m.runningCommand == "cat" && m.ready {
		return "" +
			m.viewport.View() +
			"\n" +
			m.quitStyle.Render("ctrl + c to exit this file.\n")
	}

	return "" +
		content +
		"\n" +
		m.textInput.View() +
		"\n" +
		m.quitStyle.Render(m.helpText+"\n")
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
	source := db.EventSourceSystem
	if userEvent {
		source = db.EventSourceUser
	}

	event := &db.Event{
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
