package honeypot

import (
	"fmt"
	"net"
	"os"
	"regexp"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/log/v2"
	"github.com/charmbracelet/ssh"
	"github.com/google/shlex"
	"github.com/mikeflynn/honeybearhoneypot/internal/entity"
	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot/filesystem"
)

func execMiddleware(next ssh.Handler) ssh.Handler {
	return func(s ssh.Session) {
		rawCmd := s.RawCommand()
		if len(rawCmd) > 0 {
			host := s.Context().RemoteAddr().String()
			ip, _, err := net.SplitHostPort(host)
			if err != nil {
				ip = host
			}

			if !GetRateLimiter().Check(ip) {
				log.Warn("Rate limit exceeded for IP", "ip", ip, "until", GetRateLimiter().limits[ip].bannedUntil)
				fmt.Fprintf(s, "Connection refused: rate limit exceeded\n")
				s.Exit(1)
				return
			}

			handleExec(s, rawCmd)
			return
		}
		next(s)
	}
}

func handleExec(s ssh.Session, rawCmd string) {
	// Basic context
	user := s.Context().User()
	host := s.Context().RemoteAddr().String()

	env := DefaultEnviron(user, "")

	// Log login event
	logEvent(user, host, "login", "Logged in via exec")

	// Log command event
	logEvent(user, host, "typed", rawCmd)
	log.Info(fmt.Sprintf("Exec command entered by %s:%s: %s", user, host, rawCmd))

	// Ensure filesystem is initialized (following pattern in pot.go)
	// Note: This might be race-prone as discussed, but consistency with existing code is key.
	filesystem.Initialize()

	// Initialize "current directory" to home (mimic login)
	currentDir := filesystem.HomeDir

	// Preserving backslashes that shlex would otherwise strip in double quotes
	// by escaping them for shlex.
	re := regexp.MustCompile(`\\([^$"\\` + "`" + `\n])`)
	escapedCommand := re.ReplaceAllString(rawCmd, `\\$0`)

	cmd, err := shlex.Split(escapedCommand)
	if err != nil {
		fmt.Fprintf(s, "Error parsing command: %s\n", err)
		return
	}

	if len(cmd) > 0 {
		// Expand environment variables
		for i := range cmd {
			cmd[i] = os.Expand(cmd[i], func(k string) string {
				return env[k]
			})
		}

		switch cmd[0] {
		case "exit":
			s.Exit(0)
			return
		default:
			runCommand(s, currentDir, cmd[0], cmd[1:], user, "default", env)
		}
	}
}

func runCommand(s ssh.Session, dir *filesystem.Node, bin string, args []string, user, group string, env map[string]string) {
	teaCmd, err := filesystem.RunNode(dir, bin, args, user, group, env)
	if err != nil {
		fmt.Fprintf(s, "%s\n", err)
		return
	}

	if teaCmd != nil {
		// teaCmd is a func() tea.Msg
		msg := (*teaCmd)()
		handleTeaMsg(s, msg)
	}
}

func handleTeaMsg(s ssh.Session, msg tea.Msg) {
	switch m := msg.(type) {
	case filesystem.OutputMsg:
		fmt.Fprintln(s, string(m))
	case filesystem.FileContentsMsg:
		s.Write([]byte(m))
	case filesystem.ClearOutputMsg:
		// Maybe print a bunch of newlines or ansi clear code?
		// For exec, usually we don't clear screen, but let's try ansi.
		fmt.Fprint(s, "\033[H\033[2J")
	case tea.BatchMsg:
		// It's a batch of commands. We need to iterate and execute them.
		// tea.BatchMsg is []tea.Cmd
		for _, cmd := range m {
			subMsg := cmd()
			handleTeaMsg(s, subMsg)
		}
	// Handle specific other messages if needed
	case filesystem.SetRunningCmd:
		// Ignore state changes for interactive mode
		if string(m) != "" {
			fmt.Fprintln(s, "Interactive mode not supported in exec session.")
		}
	case filesystem.ListActiveUsersMsg:
		// Replicate the formatting logic from model.go
		users := activeUsersSnapshot()
		output := fmt.Sprintf("04:25:58 up 10 days, 23:21,  %d users,  load average: 0.10, 0.18, 0.10\n", len(users))
		output += fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s\t%s\n", "USER", "TTY", "FROM", "LOGIN@", "IDLE", "JCPU", "PCPU WHAT")
		for i, u := range users {
			output += fmt.Sprintf("%s\tpts/%d\t%s\t%s\t%s\t%s\t%s\n", u, i, "--", "--", "--", "--", "--")
		}
		fmt.Fprint(s, output)

	case filesystem.ChangeDirMsg:
		// CD in exec is mostly useless as the session ends, but we can print it.
		fmt.Fprintf(s, "cd %s\n", m.Path)

	default:
		// Ignore unknown messages
	}
}

func logEvent(user, host, eventType, action string) {
	source := entity.EventSourceSystem
	if eventType == "typed" || eventType == "login" { // Assume 'typed' and 'login' are user events
		source = entity.EventSourceUser
	}

	event := &entity.Event{
		User:      user,
		Host:      host,
		App:       "ssh",
		Source:    source,
		Type:      eventType,
		Action:    action,
		Timestamp: time.Now(),
	}

	event.Publish()
	if err := event.Save(); err != nil {
		log.Error("Failed to save event", "error", err)
	}
}
