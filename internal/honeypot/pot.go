package honeypot

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
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
	"github.com/mikeflynn/honeybearhoneypot/internal/entity"
	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot/confetti"
	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot/embedded"
	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot/filesystem"
)

const (
	host            = "0.0.0.0"
	defaultMaxUsers = 10
)

var (
	activeUsers            int
	potPort                string
	potAdditionalListeners []*net.Listener
	tunnelUser             string
	tunnelAddr             string
	tunnelKey              string
	tunnelRemotePort       = "22"
	tunnelLocalHost        = "127.0.0.1"
	tunnelLocalPort        = "2222"
)

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
		tunnelRemotePort = hp[1]
	}

	tunnelKey = *keyPath

	return nil
}

func StartHoneyPot() {
	activeUsers = 0
	maxUsers := entity.OptionGetInt(entity.KeyPotMaxUsers)
	if maxUsers == 0 {
		maxUsers = defaultMaxUsers
	}

	s, err := wish.NewServer(
		wish.WithAddress(net.JoinHostPort(host, potPort)),
		wish.WithHostKeyPath(".ssh/id_ed25519"),
		wish.WithPasswordAuth(func(ctx ssh.Context, password string) bool {
			if activeUsers+1 > maxUsers {
				return false
			}

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
			activeterm.Middleware(), // Bubble Tea apps usually require a PTY.
			//accesscontrol.Middleware(),
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
		events: map[string]time.Time{
			"session_start": time.Now(),
		},
		confetti:   confetti.InitialModel(),
		output:     "",
		helpText:   "Type 'help' to see some commands; Use up/down for history.",
		historyIdx: 0,
		history:    []string{},
	}

	return m, []tea.ProgramOption{
		//tea.WithAltScreen(),
	}
}

func doTick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return filesystem.TickMsg(t)
	})
}
