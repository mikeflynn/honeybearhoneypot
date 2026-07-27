package ctf

import (
	"fmt"
	"strings"
	"time"

	"unicode"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/mikeflynn/honeybearhoneypot/internal/entity"
	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot/webhook"
)

// leaderboardPublisher is set by main.go at startup. When nil or when its
// underlying URL is empty, the publish call is a no-op.
var leaderboardPublisher *webhook.Dispatcher

// SetLeaderboardPublisher wires the webhook dispatcher used after successful
// flag submissions. Safe to call once at startup.
func SetLeaderboardPublisher(d *webhook.Dispatcher) {
	leaderboardPublisher = d
}

func publishLeaderboard(event string) {
	if leaderboardPublisher != nil {
		leaderboardPublisher.Publish(event)
	}
}

// Start returns a tea.Msg used to launch the CTF game.
func Start() tea.Msg { return startMsg{} }

// startMsg is used to bootstrap the game from the filesystem command.
type startMsg struct{}

// QuitMsg is sent when the user wants to exit the CTF without quitting the host
// program.
type QuitMsg struct{}

// Quit returns a message that signals the parent model to close the CTF view.
func Quit() tea.Msg { return QuitMsg{} }

// ConfettiMsg is sent when the player solves a CTF task. The parent model is
// responsible for translating this into the appropriate confetti animation
// commands (filesystem.SetRunningCmd + confetti.Burst).
// Active=true means "start confetti"; Active=false means "stop confetti".
type ConfettiMsg struct{ Active bool }

// leaderboardLoadedMsg carries the result of an async leaderboard query so
// the SQLite read stays off the Bubble Tea event loop.
type leaderboardLoadedMsg struct {
	users    []entity.CTFUser
	myRank   int
	myPoints int
	err      error
}

func loadLeaderboardCmd(username string) tea.Cmd {
	return func() tea.Msg {
		users, err := entity.Leaderboard(10)
		if err != nil {
			return leaderboardLoadedMsg{err: err}
		}
		var rank, points int
		if username != "" {
			if r, p, found, rerr := entity.RankFor(username); rerr != nil {
				return leaderboardLoadedMsg{err: rerr}
			} else if found {
				rank, points = r, p
			}
		}
		return leaderboardLoadedMsg{users: users, myRank: rank, myPoints: points}
	}
}

// successTickMsg drives the arcade success-screen animation.
type successTickMsg struct{}

func successTick() tea.Cmd {
	return tea.Tick(80*time.Millisecond, func(time.Time) tea.Msg {
		return successTickMsg{}
	})
}

// gameState indicates which screen we're showing.
type gameState int

const (
	stateLogin gameState = iota
	stateMenu
	stateAnswer
	stateLeaderboard
	stateSuccess
	stateDone
)

type Task struct {
	Name           string
	Description    string
	Flag           string
	Points         int
	Completed      bool
	Archived       bool
	SuccessMessage string
}

type Model struct {
	state    gameState
	username string
	password string
	user     *entity.CTFUser

	sshuser string
	sshhost string

	width  int
	height int

	usernameInput textinput.Model
	passwordInput textinput.Model
	answerInput   textinput.Model

	tasks        []Task
	activeTasks  []Task
	archivedDone []Task
	cursor       int

	selectedTask *Task

	leaderboard []entity.CTFUser
	// myRank/myPoints let a player below the visible top-N still see their own
	// standing. myRank is 0 when the current user has no ranked score.
	myRank   int
	myPoints int
	errMsg   string

	// success-screen animation snapshot
	successFrame    int
	successMsg      string
	successBonus    int
	successOldTotal int
	successNewTotal int
}

// partitionTasks splits tasks into (active, archivedDone).
// Active = not archived (regardless of completion state).
// archivedDone = archived AND completed by the current user.
// Archived-and-not-completed tasks are excluded entirely.
func partitionTasks(tasks []Task) (active, archivedDone []Task) {
	for _, t := range tasks {
		switch {
		case !t.Archived:
			active = append(active, t)
		case t.Archived && t.Completed:
			archivedDone = append(archivedDone, t)
		}
	}
	return active, archivedDone
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
	m.activeTasks, m.archivedDone = partitionTasks(m.tasks)
	if m.cursor >= len(m.activeTasks) {
		m.cursor = 0
	}
}

// wordWrap returns text wrapped to the given width. It tries to break on
// whitespace characters when possible.
func wordWrap(text string, width int) string {
	if width <= 0 {
		return text
	}

	var result strings.Builder
	for len(text) > 0 {
		if len(text) <= width {
			result.WriteString(text)
			break
		}

		cut := strings.LastIndexFunc(text[:width+1], unicode.IsSpace)
		if cut <= 0 {
			cut = width
		}

		line := strings.TrimRightFunc(text[:cut], unicode.IsSpace)
		result.WriteString(line)
		result.WriteByte('\n')
		text = strings.TrimLeftFunc(text[cut:], unicode.IsSpace)
	}

	return result.String()
}

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

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
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
	case leaderboardLoadedMsg:
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			m.state = stateMenu
		} else {
			m.leaderboard = msg.users
			m.myRank = msg.myRank
			m.myPoints = msg.myPoints
		}
		return m, nil
	case successTickMsg:
		if m.state == stateSuccess {
			m.successFrame++
			return m, successTick()
		}
		return m, nil
	}

	switch m.state {
	case stateLogin:
		return m.updateLogin(msg)
	case stateMenu:
		return m.updateMenu(msg)
	case stateAnswer:
		return m.updateAnswer(msg)
	case stateLeaderboard:
		return m.updateLeaderboard(msg)
	case stateSuccess:
		return m.updateSuccess(msg)
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

func (m Model) updateLogin(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
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

func (m Model) updateMenu(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.activeTasks)-1 {
				m.cursor++
			}
		case "enter":
			if len(m.activeTasks) > 0 {
				if m.activeTasks[m.cursor].Completed {
					return m, nil
				}
				// Find the master-list pointer so CompleteTask updates the right element.
				for i := range m.tasks {
					if m.tasks[i].Name == m.activeTasks[m.cursor].Name {
						m.selectedTask = &m.tasks[i]
						break
					}
				}
				m.state = stateAnswer
				m.answerInput.SetValue("")
				m.answerInput.Focus()
			}
		case "l", "L":
			m.state = stateLeaderboard
			m.leaderboard = nil
			m.myRank = 0
			m.myPoints = 0
			username := ""
			if m.user != nil {
				username = m.user.Username
			}
			return m, loadLeaderboardCmd(username)
		case "q", "ctrl+c":
			m.state = stateDone
			return m, func() tea.Msg { return QuitMsg{} }
		}
	}
	return m, nil
}

// successBanner returns the headline shown on the success screen: the custom
// message when set, otherwise a default.
func successBanner(custom string) string {
	if strings.TrimSpace(custom) == "" {
		return "Challenge Complete!"
	}
	return strings.TrimSpace(custom)
}

// tallyValue returns the animated count-up value at the given frame, ramping
// linearly from start to end over duration frames, then holding at end.
func tallyValue(start, end, frame, duration int) int {
	if duration <= 0 || frame >= duration {
		return end
	}
	if frame <= 0 {
		return start
	}
	return start + (end-start)*frame/duration
}

// dottedLeader renders "LABEL ....... value" padded to width with a dotted run.
func dottedLeader(label, value string, width int) string {
	dots := width - (len(label) + len(value) + 2)
	if dots < 1 {
		dots = 1
	}
	return fmt.Sprintf("%s %s %s", label, strings.Repeat(".", dots), value)
}

func (m Model) updateAnswer(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "enter":
			ans := strings.TrimSpace(m.answerInput.Value())
			if ans == m.selectedTask.Flag {
				var next tea.Cmd
				if err := m.user.CompleteTask(m.selectedTask.Name, m.selectedTask.Points); err != nil {
					m.errMsg = err.Error()
					m.state = stateMenu
				} else {
					m.selectedTask.Completed = true
					publishLeaderboard("solve")

					// Create a GUI notification for task completion
					go func(taskName string, points int) {
						event := &entity.Event{
							User:      m.sshuser,
							Host:      m.sshhost,
							App:       "ctf",
							Source:    entity.EventSourceUser,
							Type:      "taskCompleted",
							Action:    fmt.Sprintf("Completed CTF task: %s (+%d pts)", taskName, points),
							Timestamp: time.Now(),
						}
						event.Publish()
						event.Save()
					}(m.selectedTask.Name, m.selectedTask.Points)

					// Recompute partition so activeTasks reflects the newly-completed task.
					m.activeTasks, m.archivedDone = partitionTasks(m.tasks)
					if m.cursor >= len(m.activeTasks) {
						m.cursor = 0
					}

					// Snapshot data for the arcade success screen.
					m.successMsg = successBanner(m.selectedTask.SuccessMessage)
					m.successBonus = m.selectedTask.Points
					m.successNewTotal = m.user.Points
					m.successOldTotal = m.user.Points - m.selectedTask.Points
					m.successFrame = 0
					m.errMsg = ""
					m.state = stateSuccess
					next = successTick()
				}
				return m, next
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

func (m Model) updateSuccess(msg tea.Msg) (Model, tea.Cmd) {
	if _, ok := msg.(tea.KeyPressMsg); ok {
		m.state = stateMenu
		m.errMsg = ""
		return m, nil
	}
	return m, nil
}

func (m Model) updateLeaderboard(msg tea.Msg) (Model, tea.Cmd) {
	if _, ok := msg.(tea.KeyPressMsg); ok {
		m.state = stateMenu
		return m, nil
	}
	return m, nil
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
			m.errMsg,
			m.usernameInput.View(),
			m.passwordInput.View(),
		)
	case stateMenu:
		header := fmt.Sprintf("Honey Bear Honey Pot CTF - %s (%d pts)", m.user.Username, m.user.Points)
		footer := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("press L for leaderboard, q to quit")
		return lipgloss.JoinVertical(lipgloss.Left,
			titleStyle.Render(header),
			m.renderTasks(false),
			m.errMsg,
			footer,
		)
	case stateAnswer:
		desc := wordWrap(m.selectedTask.Description, m.width-4)
		descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
		box := lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Padding(1, 2)
		content := lipgloss.JoinVertical(lipgloss.Left,
			titleStyle.Render(m.selectedTask.Name),
			descStyle.Render(desc),
			m.answerInput.View(),
			m.errMsg,
		)
		return box.Render(content)
	case stateLeaderboard:
		box := lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Padding(1, 2)
		return box.Render(m.renderLeaderboard())
	case stateSuccess:
		return m.renderSuccess()
	case stateDone:
		return "Goodbye"
	}
	return ""
}

func (m Model) renderSuccess() string {
	palette := []string{"9", "11", "10", "14", "13", "12"}
	const tallyFrames = 12
	const blinkFrames = 6

	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(palette[(m.successFrame/2)%len(palette)])).Bold(true)
	msgStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Bold(true)
	scoreStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("7"))

	bonus := tallyValue(0, m.successBonus, m.successFrame, tallyFrames)
	score := tallyValue(m.successOldTotal, m.successNewTotal, m.successFrame, tallyFrames)

	const leaderWidth = 30
	bonusLine := dottedLeader("BONUS", fmt.Sprintf("+%d", bonus), leaderWidth)
	scoreLine := dottedLeader("SCORE", fmt.Sprintf("%d", score), leaderWidth)

	// Wrap the message; only decorate with >>> <<< when it stays on one line.
	maxMsg := 44
	if m.width > 0 && m.width-12 < maxMsg {
		maxMsg = m.width - 12
	}
	if maxMsg < 10 {
		maxMsg = 10
	}
	wrapped := wordWrap(m.successMsg, maxMsg)
	var msgLine string
	if strings.Contains(wrapped, "\n") {
		msgLine = msgStyle.Render(wrapped)
	} else {
		msgLine = msgStyle.Render(">>>  " + wrapped + "  <<<")
	}

	prompt := " "
	if (m.successFrame/blinkFrames)%2 == 0 {
		prompt = "press any key to continue"
	}

	body := lipgloss.JoinVertical(lipgloss.Center,
		titleStyle.Render("·  ·  ✦  LEVEL COMPLETE  ✦  ·  ·"),
		"",
		"🏆   🍯   🏆",
		"",
		msgLine,
		"",
		scoreStyle.Render(bonusLine),
		scoreStyle.Render(scoreLine),
		"",
		dimStyle.Render(prompt),
	)

	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 4)
	inner := box.Render(body)
	if m.width > 0 {
		return lipgloss.PlaceHorizontal(m.width, lipgloss.Center, inner)
	}
	return inner
}

func (m Model) renderTasks(showAllDesc bool) string {
	var b strings.Builder
	bullet := lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Render("🍯")
	doneBullet := lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render("✓")
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("0")).Background(lipgloss.Color("6")).Bold(true)
	normalStyle := lipgloss.NewStyle()
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	archivedHeader := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("── Archived ──")

	renderOne := func(t Task, i int, selectable bool) {
		blet := bullet
		style := normalStyle
		if t.Completed {
			blet = doneBullet
			style = normalStyle.Foreground(lipgloss.Color("8"))
		}

		marker := " "
		if selectable && m.state == stateMenu && i == m.cursor {
			marker = ">"
		}

		line := fmt.Sprintf("%s %s %s (%d pts)", marker, blet, t.Name, t.Points)
		if selectable && m.state == stateMenu && i == m.cursor {
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
		}
		b.WriteString("  " + descStyle.Render(desc) + "\n")
	}

	if len(m.activeTasks) == 0 && len(m.archivedDone) == 0 {
		return descStyle.Render("No tasks available.")
	}

	for i, t := range m.activeTasks {
		renderOne(t, i, true)
	}

	if len(m.activeTasks) == 0 {
		b.WriteString(descStyle.Render("All active tasks complete!") + "\n")
	}

	if len(m.archivedDone) > 0 {
		b.WriteString("\n" + archivedHeader + "\n")
		for i, t := range m.archivedDone {
			renderOne(t, i, false)
		}
	}

	return strings.TrimRight(b.String(), "\n")
}

func (m Model) renderLeaderboard() string {
	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	rowStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	meStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true)
	footerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	var b strings.Builder
	b.WriteString(titleStyle.Render("Leaderboard") + "\n\n")

	if m.leaderboard == nil {
		b.WriteString(rowStyle.Render("Loading…") + "\n")
	} else if len(m.leaderboard) == 0 {
		b.WriteString(rowStyle.Render("No scores yet.") + "\n")
	} else {
		inList := false
		rank := 0
		for i, u := range m.leaderboard {
			// Competition ranking: ties share a rank (1, 2, 2, 4...), matching
			// entity.RankFor so a player's number is the same in-list or in the
			// self-rank footer below.
			if i == 0 || u.Points != m.leaderboard[i-1].Points {
				rank = i + 1
			}
			line := fmt.Sprintf("%2d. %s — %d pts", rank, u.Username, u.Points)
			if m.user != nil && u.Username == m.user.Username {
				inList = true
				b.WriteString(meStyle.Render(line) + "\n")
			} else {
				b.WriteString(rowStyle.Render(line) + "\n")
			}
		}
		// If the current player scored but sits below the visible top-N, append
		// their own standing so they can always find their score.
		if !inList && m.user != nil && m.myRank > 0 && m.myPoints > 0 {
			line := fmt.Sprintf("%2d. %s — %d pts", m.myRank, m.user.Username, m.myPoints)
			b.WriteString(footerStyle.Render("     ⋮") + "\n")
			b.WriteString(meStyle.Render(line) + "\n")
		}
	}

	b.WriteString("\n" + footerStyle.Render("press any key to return"))
	return b.String()
}
