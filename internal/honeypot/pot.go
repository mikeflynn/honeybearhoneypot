package honeypot

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

func SetTunnel(host *string, keyPath *string, bindAddr *string, remotePort *string) error {
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

	if bindAddr != nil && *bindAddr != "" {
		tunnelRemoteBind = *bindAddr
	}

	if remotePort != nil && *remotePort != "" {
		tunnelRemoteForwardPort = *remotePort
	}

	return nil
}

func withProxyProtocol() ssh.Option {
	return ssh.WrapConn(func(ctx ssh.Context, conn net.Conn) net.Conn {
		// Set short deadline to peek for PROXY header
		conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))

		header := make([]byte, 6)
		n, err := io.ReadFull(conn, header)

		// Reset deadline
		conn.SetReadDeadline(time.Time{})

		if err != nil {
			// If timeout, it means client sent nothing (waiting for banner).
			// So it's not PROXY protocol (which sends immediately).
			if n > 0 {
				return &proxyConn{
					Conn:   conn,
					reader: io.MultiReader(bytes.NewReader(header[:n]), conn),
				}
			}
			return conn
		}

		prefix := string(header)
		if prefix == "PROXY " {
			// Read the rest of the line
			lineBuf := []byte(prefix)
			byteBuf := make([]byte, 1)
			for {
				_, err := conn.Read(byteBuf)
				if err != nil {
					break
				}
				lineBuf = append(lineBuf, byteBuf[0])
				if byteBuf[0] == '\n' {
					break
				}
				if len(lineBuf) > 107 { // Max header length per spec
					break
				}
			}

			// Parse lineBuf
			// "PROXY TCP4 1.2.3.4 5.6.7.8 1234 5678\r\n"
			parts := strings.Split(strings.TrimSpace(string(lineBuf)), " ")
			if len(parts) >= 6 {
				srcIP := parts[2]
				srcPort := parts[4]
				if port, err := strconv.Atoi(srcPort); err == nil {
					newAddr := &net.TCPAddr{
						IP:   net.ParseIP(srcIP),
						Port: port,
					}
					return &proxyConn{
						Conn:   conn,
						reader: conn, // The header is consumed.
						remote: newAddr,
					}
				}
			}
			// If parse fails, return conn (header consumed, might fail SSH but we tried)
			return conn
		}

		// Not PROXY header (e.g. "SSH-2.").
		return &proxyConn{
			Conn:   conn,
			reader: io.MultiReader(bytes.NewReader(header), conn),
		}
	})
}

type proxyConn struct {
	net.Conn
	reader io.Reader
	remote net.Addr
}

func (c *proxyConn) Read(p []byte) (n int, err error) {
	return c.reader.Read(p)
}

func (c *proxyConn) RemoteAddr() net.Addr {
	if c.remote != nil {
		return c.remote
	}
	return c.Conn.RemoteAddr()
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
		withProxyProtocol(),
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
				return string(banner)
			}

			return ""
		}),
		wish.WithMiddleware(
			bubbletea.Middleware(teaHandler),
			func(next ssh.Handler) ssh.Handler {
				return func(s ssh.Session) {
					addActiveUser(s.User())

					next(s)

					removeActiveUser(s.User())
				}
			},
			activeterm.Middleware(), // Bubble Tea apps usually require a PTY.
			execMiddleware,
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
		output:     initialOutput,
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
