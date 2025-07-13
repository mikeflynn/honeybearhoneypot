package ctf

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mikeflynn/honeybearhoneypot/internal/entity"
)

// Start returns a tea.Msg used to launch the CTF game.
func Start() tea.Msg { return startMsg{} }

// startMsg is used to bootstrap the game from the filesystem command.
type startMsg struct{}

// QuitMsg is sent when the user wants to exit the CTF without quitting the host
// program.
type QuitMsg struct{}

// Quit returns a message that signals the parent model to close the CTF view.
func Quit() tea.Msg { return QuitMsg{} }

// gameState indicates which screen we're showing.
type gameState int

const (
	stateLogin gameState = iota
	stateMenu
	stateAnswer
	stateDone
)

type Task struct {
	Name        string
	Description string
	Flag        string
	Points      int
}

type Model struct {
	state    gameState
	username string
	password string
	user     *entity.CTFUser

	width  int
	height int

	usernameInput textinput.Model
	passwordInput textinput.Model
	answerInput   textinput.Model

	tasks  []Task
	cursor int

	selectedTask *Task

	errMsg string
}

func InitialModel(tasks []Task) Model {
	ti := textinput.New()
	ti.Placeholder = "username"
	ti.Focus()
	ti.CharLimit = 32

	pi := textinput.New()
	pi.Placeholder = "password"
	pi.CharLimit = 32
	pi.EchoMode = textinput.EchoPassword

	ai := textinput.New()
	ai.Placeholder = "flag"
	ai.CharLimit = 256

	return Model{
		state:         stateLogin,
		usernameInput: ti,
		passwordInput: pi,
		answerInput:   ai,
		tasks:         tasks,
		width:         0,
		height:        0,
		cursor:        0,
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case startMsg:
		m.state = stateLogin
		m.errMsg = ""
		m.cursor = 0
		m.usernameInput.Focus()
		m.passwordInput.Blur()
		return m, nil
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	switch m.state {
	case stateLogin:
		return m.updateLogin(msg)
	case stateMenu:
		return m.updateMenu(msg)
	case stateAnswer:
		return m.updateAnswer(msg)
	case stateDone:
		return m, func() tea.Msg { return QuitMsg{} }
	}
	return m, nil
}

func (m *Model) authenticate() tea.Cmd {
	u := &entity.CTFUser{Username: m.usernameInput.Value()}
	err := u.Load()
	if err != nil {
		// create
		u.Username = m.usernameInput.Value()
		u.Password = m.passwordInput.Value()
		if err := u.Save(); err != nil {
			m.errMsg = err.Error()
			return nil
		}
	}
	if u.Password != m.passwordInput.Value() {
		m.errMsg = "invalid password"
		return nil
	}
	m.user = u
	m.username = u.Username
	m.password = u.Password
	m.state = stateMenu
	m.errMsg = ""
	return nil
}

func (m Model) updateLogin(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			return m, m.authenticate()
		case "tab", "down":
			if m.usernameInput.Focused() {
				m.usernameInput.Blur()
				m.passwordInput.Focus()
			} else {
				m.passwordInput.Blur()
				m.usernameInput.Focus()
			}
			return m, nil
		case "shift+tab", "up":
			if m.passwordInput.Focused() {
				m.passwordInput.Blur()
				m.usernameInput.Focus()
			} else {
				m.usernameInput.Blur()
				m.passwordInput.Focus()
			}
			return m, nil
		}
	}
	m.usernameInput, cmd = m.usernameInput.Update(msg)
	if m.usernameInput.Focused() {
		return m, cmd
	}
	m.passwordInput, cmd = m.passwordInput.Update(msg)
	return m, cmd
}

func (m Model) updateMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.tasks)-1 {
				m.cursor++
			}
		case "enter":
			if len(m.tasks) > 0 {
				m.selectedTask = &m.tasks[m.cursor]
				m.state = stateAnswer
				m.answerInput.SetValue("")
				m.answerInput.Focus()
			}
		case "q", "ctrl+c":
			m.state = stateDone
			return m, func() tea.Msg { return QuitMsg{} }
		}
	}
	return m, nil
}

func (m Model) updateAnswer(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			ans := strings.TrimSpace(m.answerInput.Value())
			if ans == m.selectedTask.Flag {
				if err := m.user.CompleteTask(m.selectedTask.Name, m.selectedTask.Points); err != nil {
					m.errMsg = err.Error()
				} else {
					m.errMsg = fmt.Sprintf("Correct! +%d points", m.selectedTask.Points)
				}
			} else {
				m.errMsg = "Incorrect flag"
			}
			m.state = stateMenu
			return m, nil
		case "esc":
			m.state = stateMenu
			return m, nil
		}
	}
	m.answerInput, cmd = m.answerInput.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	welcome := lipgloss.NewStyle().Foreground(lipgloss.Color("7")).Render("Welcome to the Honey Bear Honey Pot CTF!\n" +
		"Create an account by entering a new username and password or login with your existing credentials.")

	switch m.state {
	case stateLogin:
		return lipgloss.JoinVertical(lipgloss.Left,
			titleStyle.Render("Honey Bear Honey Pot CTF"),
			welcome,
			m.renderTasks(true),
			m.errMsg,
			"username: "+m.usernameInput.View(),
			"password: "+m.passwordInput.View(),
		)
	case stateMenu:
		header := fmt.Sprintf("Honey Bear Honey Pot CTF - %s (%d pts)", m.user.Username, m.user.Points)
		return lipgloss.JoinVertical(lipgloss.Left,
			titleStyle.Render(header),
			m.renderTasks(false),
			m.errMsg,
		)
	case stateAnswer:
		return lipgloss.JoinVertical(lipgloss.Left,
			titleStyle.Render(m.selectedTask.Name),
			m.selectedTask.Description,
			m.answerInput.View(),
			m.errMsg,
		)
	case stateDone:
		return "Goodbye"
	}
	return ""
}

func (m Model) renderTasks(showAllDesc bool) string {
	var b strings.Builder
	for i, t := range m.tasks {
		prefix := "  "
		if m.state == stateMenu && i == m.cursor {
			prefix = ">"
		}
		b.WriteString(fmt.Sprintf("%s %s (%d pts)\n", prefix, t.Name, t.Points))
		if showAllDesc || (m.state == stateMenu && i == m.cursor) {
			b.WriteString("    " + t.Description + "\n")
		}
	}
	return strings.TrimRight(b.String(), "\n")
}
