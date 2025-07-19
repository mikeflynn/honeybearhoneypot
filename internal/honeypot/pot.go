package honeypot

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"reflect"
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
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/elapsed"
	"github.com/charmbracelet/wish/logging"
	"github.com/google/shlex"
	"github.com/mikeflynn/honeybearhoneypot/internal/config"
	"github.com/mikeflynn/honeybearhoneypot/internal/entity"
	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot/confetti"
	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot/ctf"
	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot/embedded"
	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot/filesystem"
	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot/matrix"
)

const (
	host            = "0.0.0.0"
	defaultMaxUsers = 10
)

var (
	// State
	activeUsers      []string
	usersThisSession int = 0
	activeUsersMu    sync.Mutex
	tunnelActive     int = -1 // -1 = not configured, 0 = not connected, 1 = connected

	// Config
	potPort                 string          // Primary port the honey pot will answer on.
	potAdditionalListeners  []*net.Listener // Optional additional ports for the honey pot.
	tunnelUser              string          // Username for remote SSH server
	tunnelAddr              string          // Hostname or IP of remote SSH server
	tunnelKey               string          // Path to private SSH key for remote server
	tunnelPort              = "22"          // Port of remote SSH server
	tunnelRemoteBind        = "127.0.0.1"   // IP address to bind to on the *remote* server (0.0.0.0 for all)
	tunnelRemoteForwardPort = "8022"        // Port to open on the *remote* server for forwarding
	knownHostsPath          = ""            // Path to known hosts file.
)

func addActiveUser(user string) {
	activeUsersMu.Lock()
	activeUsers = append(activeUsers, user)
	activeUsersMu.Unlock()
}

func removeActiveUser(user string) {
	activeUsersMu.Lock()
	for i, u := range activeUsers {
		if u == user {
			activeUsers = append(activeUsers[:i], activeUsers[i+1:]...)
			break
		}
	}
	activeUsersMu.Unlock()
}

func activeUsersLen() int {
	activeUsersMu.Lock()
	defer activeUsersMu.Unlock()
	return len(activeUsers)
}

func activeUsersSnapshot() []string {
	activeUsersMu.Lock()
	defer activeUsersMu.Unlock()
	snapshot := make([]string, len(activeUsers))
	copy(snapshot, activeUsers)
	return snapshot
}

func incrementUsersThisSession() {
	activeUsersMu.Lock()
	usersThisSession++
	activeUsersMu.Unlock()
}

func usersThisSessionCount() int {
	activeUsersMu.Lock()
	defer activeUsersMu.Unlock()
	return usersThisSession
}

func SetPort(port string) error {
	potPort = port
	return nil
}

func AddListeners(additionalListeners ...*net.Listener) error {
	potAdditionalListeners = additionalListeners
	return nil
}

func SetTunnel(host *string, keyPath *string) error {
	if host == nil || *host == "" || keyPath == nil || *keyPath == "" {
		// Flags not set, tunnel not needed.
		return nil
	}

	tunnelActive = 0 // Not connected yet.

	parts := strings.Split(*host, "@")
	if len(parts) < 2 {
		return errors.New("Invalid remote host.")
	}

	tunnelUser = parts[0]

	hp := strings.Split(parts[1], ":")
	if len(hp) == 1 {
		tunnelAddr = parts[1]
	} else {
		tunnelAddr = hp[0]
		tunnelPort = hp[1]
	}

	tunnelKey = *keyPath

	return nil
}

func sessionMiddleware(next ssh.Handler) ssh.Handler {
	return func(s ssh.Session) {
		addActiveUser(s.User())
		defer removeActiveUser(s.User())

		// Record login event
		e := &entity.Event{
			User:      s.User(),
			Host:      s.RemoteAddr().String(),
			App:       "ssh",
			Source:    entity.EventSourceUser,
			Type:      "login",
			Action:    "Logged in!",
			Timestamp: time.Now(),
		}
		e.Publish()
		if err := e.Save(); err != nil {
			log.Error("error saving event", "error", err)
		}

		if cmd := s.RawCommand(); cmd != "" {
			execSession(s, cmd)
			return
		}

		if _, _, ok := s.Pty(); !ok {
			io.WriteString(s, "Requires an active PTY\n")
			return
		}

		next(s)
	}
}

func execSession(s ssh.Session, raw string) {
	output := executeCommand(raw, s.User(), s.RemoteAddr().String())
	io.WriteString(s, output)
}

func executeCommand(raw, user, host string) string {
	parts, err := shlex.Split(raw)
	if err != nil {
		return fmt.Sprintf("Error parsing command: %v\n", err)
	}

	evt := &entity.Event{
		User:      user,
		Host:      host,
		App:       "ssh",
		Source:    entity.EventSourceUser,
		Type:      "typed",
		Action:    raw,
		Timestamp: time.Now(),
	}
	evt.Publish()
	_ = evt.Save()

	filesystem.Initialize()
	currentDir := filesystem.HomeDir
	group := "default"

	if len(parts) == 0 {
		return ""
	}

	var cmd *tea.Cmd
	if parts[0] == "sudo" && len(parts) > 1 {
		cmd, err = filesystem.RunNode(currentDir, parts[1], parts[2:], "root", "root")
	} else {
		cmd, err = filesystem.RunNode(currentDir, parts[0], parts[1:], user, group)
	}
	if err != nil {
		return fmt.Sprintf("%s\n", err)
	}

	return runTeaCmd(&currentDir, cmd)
}

func runTeaCmd(currentDir **filesystem.Node, cmd *tea.Cmd) string {
	if cmd == nil {
		return ""
	}
	var out strings.Builder
	var process func(tea.Msg)

	process = func(msg tea.Msg) {
		switch m := msg.(type) {
		case filesystem.OutputMsg:
			out.WriteString(string(m))
			out.WriteByte('\n')
		case filesystem.FileContentsMsg:
			out.Write(m)
		case filesystem.ListActiveUsersMsg:
			users := activeUsersSnapshot()
			out.WriteString(fmt.Sprintf("04:25:58 up 10 days, 23:21,  %d users, load average: 0.10, 0.18, 0.10\n", len(users)))
			out.WriteString(fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s\t%s\n", "USER", "TTY", "FROM", "LOGIN@", "IDLE", "JCPU", "PCPU WHAT"))
			for i, u := range users {
				out.WriteString(fmt.Sprintf("%s\tpts/%d\t%s\t%s\t%s\t%s\t%s\n", u, i, "--", "--", "--", "--", "--"))
			}
		case filesystem.ChangeDirMsg:
			*currentDir = m.Node
		default:
			v := reflect.ValueOf(msg)
			if v.Kind() == reflect.Slice {
				for i := 0; i < v.Len(); i++ {
					if sub, ok := v.Index(i).Interface().(tea.Msg); ok {
						process(sub)
					}
				}
			}
		}
	}

	first := (*cmd)()
	process(first)
	return out.String()
}

func StartHoneyPot(appConfigDir string) {
	activeUsersMu.Lock()
	activeUsers = []string{}
	usersThisSession = 0
	activeUsersMu.Unlock()
	maxUsers := entity.OptionGetInt(entity.KeyPotMaxUsers)
	if maxUsers == 0 {
		maxUsers = defaultMaxUsers
	}

	s, err := wish.NewServer(
		wish.WithAddress(net.JoinHostPort(host, potPort)),
		wish.WithHostKeyPath(appConfigDir+"/.ssh/id_ed25519"),
		wish.WithPasswordAuth(func(ctx ssh.Context, password string) bool {
			if activeUsersLen()+1 > maxUsers {
				return false
			}

			log.Info(fmt.Sprintf("Authorization used: %s, %s", ctx.User(), password))
			incrementUsersThisSession()
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
			sessionMiddleware,
			bubbletea.Middleware(teaHandler),
			logging.Middleware(),
			elapsed.Middleware(),
		),
	)
	if err != nil {
		log.Error("Could not create server", "error", err)
		return
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	log.Info("Starting SSH server", "host", host, "port", potPort)

	go func() {
		if err = s.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			log.Error("Could not start server", "error", err)
			done <- nil
		}
	}()

	if len(potAdditionalListeners) > 0 {
		for _, additionalListener := range potAdditionalListeners {
			if additionalListener == nil {
				continue
			}

			go func() {
				if err = s.Serve(*additionalListener); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
					log.Error("Could not start server", "error", err)
					done <- nil
				}
			}()
		}
	}

	// SSH Reverse Tunnel
	if tunnelKey != "" || tunnelAddr != "" || tunnelUser != "" {
		go setupReverseTunnel(
			tunnelUser, tunnelAddr, tunnelPort,
			tunnelKey, knownHostsPath,
			fmt.Sprintf("localhost:%s", potPort), // The primary address of the honey pot.
			tunnelRemoteBind, tunnelRemoteForwardPort,
		)
	}

	<-done
	log.Info("Stopping SSH server")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer func() { cancel() }()
	if err := s.Shutdown(ctx); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		log.Error("Could not stop server", "error", err)
	}
}

func teaHandler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
	// Expect a PTY when running interactively.
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
		events: map[string]time.Time{
			"session_start": time.Now(),
		},
		confetti:   confetti.InitialModel(),
		matrix:     matrix.InitialModel(pty.Window.Width, pty.Window.Height),
		ctf:        ctf.InitialModel(convertTasks(config.Active.Tasks)),
		output:     "",
		helpText:   "Type 'help' to see some commands; Use up/down for history.",
		historyIdx: 0,
		history:    []string{},
	}

	return m, []tea.ProgramOption{
		//tea.WithAltScreen(),
	}
}

func convertTasks(t []config.Task) []ctf.Task {
	out := make([]ctf.Task, len(t))
	for i, task := range t {
		out[i] = ctf.Task{
			Name:        task.Name,
			Description: task.Description,
			Flag:        task.Flag,
			Points:      task.Points,
		}
	}
	return out
}

func doTick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return filesystem.TickMsg(t)
	})
}
