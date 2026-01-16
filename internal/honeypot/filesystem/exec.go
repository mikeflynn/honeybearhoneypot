package filesystem

import (
	"fmt"
	"math/rand/v2"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func bearSayExec(dir *Node, params []string) *tea.Cmd {
	cmds := []tea.Cmd{}

	cmds = append(cmds, tea.Cmd(func() tea.Msg {
		output := `
				  __         __
				 /  \.-"""-./  \
				\    -   -    /
				 |   o   o   |
				 \  .-'''-.  /
				  '-\__Y__/-'
				     '---'
				      (\__/)
				      (='.'=)
				      (")_(")

				%s

			`

		if len(params) == 0 {
			defaults := []string{
				"Hello, world!",
				"You're in!",
				"I don't play well with others",
				"Hack the planet!",
				"Its in the place that I put that thing that time.",
				"Stay curious!",
			}

			output = fmt.Sprintf(output, defaults[rand.IntN(len(defaults))])
		} else {
			output = fmt.Sprintf(output, strings.Join(params, " "))
		}

		return OutputMsg(output)
	}))

	batch := tea.Batch(cmds...)
	return &batch
}

func neofetchExec(dir *Node, params []string) *tea.Cmd {
	cmds := []tea.Cmd{}

	cmds = append(cmds, tea.Cmd(func() tea.Msg {
		logoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
		keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
		valStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("255"))

		logo := `
       __         __
      /  \.-"""-./  \
     \    -   -    /
      |   o   o   |
      \  .-'''-.  /
       '-\__Y__/-'
          '---'
`
		logoLines := strings.Split(logo, "\n")

		info := []struct {
			key string
			val string
		}{
			{"OS", "Hardhat Linux"},
			{"Host", "HoneyBear (1.0)"},
			{"Kernel", "6.22.0-81-generic"},
			{"Uptime", "10 days, 23 hours, 21 mins"},
			{"Packages", "1337 (dpkg)"},
			{"Shell", "bearshell"},
			{"Resolution", "1920x1080"},
			{"DE", "HoneyDesktop"},
			{"WM", "HoneyWindow"},
			{"Theme", "HoneyDark [GTK2/3]"},
			{"Icons", "HoneyIcons [GTK2/3]"},
			{"Terminal", "/dev/pts/0"},
			{"CPU", "HoneyBear CPU @ 4.0GHz"},
			{"GPU", "HoneyBear GPU"},
			{"Memory", "1337MiB / 2048MiB"},
		}

		var outputLines []string
		maxLines := len(logoLines)
		if len(info) > maxLines {
			maxLines = len(info)
		}

		// User@Host line
		userHost := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true).Render("root@honeybear")
		separator := "----------------"

		// Prepend empty lines to logo if it's shorter than info to center it roughly or top align
		// Actually neofetch usually top aligns.

		for i := 0; i < maxLines; i++ {
			logoLine := ""
			if i < len(logoLines) {
				logoLine = logoStyle.Render(logoLines[i])
			}

			// Pad logo line to a fixed width
			logoWidth := 30
			currentLen := 0
			if i < len(logoLines) {
				currentLen = lipgloss.Width(logoLines[i]) // Use raw string for length
			}
			if logoWidth < currentLen {
				logoWidth = currentLen
			}
			padding := strings.Repeat(" ", logoWidth-currentLen)
			logoPart := logoLine + padding

			infoPart := ""
			if i == 0 {
				infoPart = userHost
			} else if i == 1 {
				infoPart = valStyle.Render(separator)
			} else if i-2 < len(info) && i >= 2 {
				idx := i - 2
				infoPart = keyStyle.Render(info[idx].key+":") + " " + valStyle.Render(info[idx].val)
			}

			outputLines = append(outputLines, logoPart+infoPart)
		}

		// Color blocks at the bottom
		blocks := ""
		for i := 0; i < 8; i++ {
			blocks += lipgloss.NewStyle().Background(lipgloss.Color(fmt.Sprintf("%d", i))).Render("   ")
		}
		blocks += "\n"
		for i := 8; i < 16; i++ {
			blocks += lipgloss.NewStyle().Background(lipgloss.Color(fmt.Sprintf("%d", i))).Render("   ")
		}

		outputLines = append(outputLines, "", blocks)

		return OutputMsg("\n" + strings.Join(outputLines, "\n"))
	}))

	batch := tea.Batch(cmds...)
	return &batch
}

func catExec(dir *Node, params []string) *tea.Cmd {
	cmds := []tea.Cmd{}
	cmds = append(cmds, tea.Cmd(func() tea.Msg {
		return SetRunningCmd("cat")
	}))

	cmds = append(cmds, tea.Cmd(func() tea.Msg {
		if len(params) == 0 {
			return OutputMsg("cat: missing file operand")
		}

		target, err := GetNodeByPath(dir, params[0])
		if err != nil || target == nil {
			return OutputMsg(err.Error())
		}

		fileData, err := target.Open()
		if err != nil {
			return OutputMsg("cat: " + err.Error())
		}

		return FileContentsMsg(fileData)
	}))

	batch := tea.Batch(cmds...)
	return &batch
}

func idExec(dir *Node, params []string) *tea.Cmd {
	cmd := tea.Cmd(func() tea.Msg {
		return OutputMsg("uid=1000(you) gid=1000(you) groups=1000(you),27(sudo)")
	})
	return &cmd
}

func psExec(dir *Node, params []string) *tea.Cmd {
	cmd := tea.Cmd(func() tea.Msg {
		output := fmt.Sprintf("%-8s %-5s %-5s %-5s %-8s %-8s %-5s %s\n", "USER", "PID", "%CPU", "%MEM", "VSZ", "RSS", "TTY", "COMMAND")
		processes := []struct {
			user, pid, cpu, mem, vsz, rss, tty, cmd string
		}{
			{"root", "1", "0.0", "0.1", "168244", "12544", "?", "/sbin/init"},
			{"root", "2", "0.0", "0.0", "0", "0", "?", "[kthreadd]"},
			{"root", "3", "0.0", "0.0", "0", "0", "?", "[rcu_gp]"},
			{"you", "452", "0.1", "0.5", "23452", "8432", "pts/0", "-bash"},
			{"you", "1337", "4.2", "1.2", "104320", "24512", "pts/0", "./honeybear --no-gui"},
			{"root", "2048", "0.0", "0.2", "72432", "4124", "?", "/usr/sbin/sshd -D"},
		}

		for _, p := range processes {
			output += fmt.Sprintf("%-8s %-5s %-5s %-5s %-8s %-8s %-5s %s\n", p.user, p.pid, p.cpu, p.mem, p.vsz, p.rss, p.tty, p.cmd)
		}
		return OutputMsg(output)
	})
	return &cmd
}

func envExec(dir *Node, params []string) *tea.Cmd {
	cmd := tea.Cmd(func() tea.Msg {
		envVars := []string{
			"SHELL=/bin/bash",
			"PWD=/home/you",
			"LOGNAME=you",
			"HOME=/home/you",
			"LANG=en_US.UTF-8",
			"USER=you",
			"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
			"AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE",
			"AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			"_=/usr/bin/env",
		}
		return OutputMsg(strings.Join(envVars, "\n"))
	})
	return &cmd
}

func netstatExec(dir *Node, params []string) *tea.Cmd {
	cmd := tea.Cmd(func() tea.Msg {
		output := "Active Internet connections (only servers)\n"
		output += fmt.Sprintf("%-5s %-6s %-6s %-20s %-20s %-10s\n", "Proto", "Recv-Q", "Send-Q", "Local Address", "Foreign Address", "State")
		ports := []struct {
			proto, recv, send, local, foreign, state string
		}{
			{"tcp", "0", "0", "0.0.0.0:22", "0.0.0.0:*", "LISTEN"},
			{"tcp", "0", "0", "0.0.0.0:1337", "0.0.0.0:*", "LISTEN"},
			{"tcp", "0", "0", "127.0.0.1:3306", "0.0.0.0:*", "LISTEN"},
			{"tcp6", "0", "0", ":::80", ":::*", "LISTEN"},
		}

		for _, p := range ports {
			output += fmt.Sprintf("%-5s %-6s %-6s %-20s %-20s %-10s\n", p.proto, p.recv, p.send, p.local, p.foreign, p.state)
		}
		return OutputMsg(output)
	})
	return &cmd
}
