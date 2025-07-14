package ctf

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mikeflynn/honeybearhoneypot/internal/entity"
	"github.com/muesli/reflow/wordwrap"
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
	Completed   bool
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

func (m *Model) loadCompleted() {
	if m.user == nil {
		return
	}
	tasks, err := m.user.CompletedTasks()
	if err != nil {
		m.errMsg = err.Error()
		return
	}
	done := map[string]struct{}{}
	for _, t := range tasks {
		done[t] = struct{}{}
	}
	for i := range m.tasks {
		if _, ok := done[m.tasks[i].Name]; ok {
			m.tasks[i].Completed = true
		}
	}
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
	ai.Prompt = "❯ "
	ai.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("5")).Bold(true)
	ai.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
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
	pass := strings.TrimSpace(m.passwordInput.Value())
	if pass == "" {
		m.errMsg = "password required"
		return nil
	}

	u := &entity.CTFUser{Username: strings.TrimSpace(m.usernameInput.Value())}
	err := u.Load()
	if err != nil {
		// create new account
		u.Username = m.usernameInput.Value()
		u.Password = pass
		if err := u.Save(); err != nil {
			m.errMsg = err.Error()
			return nil
		}
	}

	if u.Password != pass {
		m.errMsg = "invalid password"
		return nil
	}

	m.user = u
	m.username = u.Username
	m.password = u.Password
	m.state = stateMenu
	m.errMsg = ""
	m.loadCompleted()
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
				if m.tasks[m.cursor].Completed {
					return m, nil
				}
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
					m.errMsg = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true).Render(
						fmt.Sprintf("🎉 Correct! +%d points! 🎉", m.selectedTask.Points))
					m.selectedTask.Completed = true
				}
				m.state = stateMenu
				return m, nil
			}

			m.errMsg = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true).Render("Incorrect flag")
			return m, nil
		case "esc":
			m.state = stateMenu
			m.errMsg = ""
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
		desc := wordwrap.String(m.selectedTask.Description, uint(m.width-4))
		descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
		box := lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Padding(1, 2)
		content := lipgloss.JoinVertical(lipgloss.Left,
			titleStyle.Render(m.selectedTask.Name),
			descStyle.Render(desc),
			m.answerInput.View(),
			m.errMsg,
		)
		return box.Render(content)
	case stateDone:
		return "Goodbye"
	}
	return ""
}

func (m Model) renderTasks(showAllDesc bool) string {
	var b strings.Builder
	bullet := lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Render("•")
	doneBullet := lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render("✓")
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("0")).Background(lipgloss.Color("6")).Bold(true)
	normalStyle := lipgloss.NewStyle()
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("7"))

	for i, t := range m.tasks {
		blet := bullet
		style := normalStyle
		if t.Completed {
			blet = doneBullet
			style = normalStyle.Foreground(lipgloss.Color("8"))
		}
		line := fmt.Sprintf("%s %s (%d pts)", blet, t.Name, t.Points)
		if m.state == stateMenu && i == m.cursor {
			line = selectedStyle.Render(line)
		} else {
			line = style.Render(line)
		}
		b.WriteString(line + "\n")
		desc := t.Description
		if !showAllDesc {
			r := []rune(desc)
			limit := 60
			if len(r) > limit {
				desc = string(r[:limit-3]) + "..."
			}
			b.WriteString("  " + descStyle.Render(desc) + "\n")
		} else {
			b.WriteString("  " + descStyle.Render(desc) + "\n")
		}
	}
	return strings.TrimRight(b.String(), "\n")
}
