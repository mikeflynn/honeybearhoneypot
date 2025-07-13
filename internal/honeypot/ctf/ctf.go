package ctf

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mikeflynn/honeybearhoneypot/internal/entity"
)

// Start returns a tea.Msg used to launch the CTF game.
func Start() tea.Msg { return startMsg{} }

// startMsg is used to bootstrap the game from the filesystem command.
type startMsg struct{}

// quitMsg is sent when the user wants to exit the CTF without quitting the host program.
type quitMsg struct{}

// Quit returns a message that signals the parent model to close the CTF view.
func Quit() tea.Msg { return quitMsg{} }

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

type item struct {
	task Task
}

func (i item) Title() string {
	return fmt.Sprintf("%s (%d pts)", i.task.Name, i.task.Points)
}
func (i item) Description() string { return i.task.Description }
func (i item) FilterValue() string { return i.task.Name }

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

	list list.Model

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

	items := []list.Item{}
	for _, t := range tasks {
		items = append(items, item{task: t})
	}
	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Challenges"

	width := 0
	height := 0

	return Model{
		state:         stateLogin,
		usernameInput: ti,
		passwordInput: pi,
		answerInput:   ai,
		list:          l,
		width:         width,
		height:        height,
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case startMsg:
		return m, nil
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width-4, msg.Height-10)
	}

	switch m.state {
	case stateLogin:
		return m.updateLogin(msg)
	case stateMenu:
		return m.updateMenu(msg)
	case stateAnswer:
		return m.updateAnswer(msg)
	case stateDone:
		return m, func() tea.Msg { return quitMsg{} }
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
		m.list, _ = m.list.Update(msg)
		return m, cmd
	}
	m.passwordInput, cmd = m.passwordInput.Update(msg)
	m.list, _ = m.list.Update(msg)
	return m, cmd
}

func (m Model) updateMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if it, ok := m.list.SelectedItem().(item); ok {
				m.selectedTask = &it.task
				m.state = stateAnswer
				m.answerInput.SetValue("")
				m.answerInput.Focus()
			}
			return m, nil
		case "q", "ctrl+c":
			m.state = stateDone
			return m, func() tea.Msg { return quitMsg{} }
		}
	}
	m.list, cmd = m.list.Update(msg)
	return m, cmd
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
			m.list.View(),
			m.errMsg,
			"username: "+m.usernameInput.View(),
			"password: "+m.passwordInput.View(),
		)
	case stateMenu:
		header := fmt.Sprintf("Honey Bear Honey Pot CTF - %s (%d pts)", m.user.Username, m.user.Points)
		m.list.Title = header
		return m.list.View()
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
