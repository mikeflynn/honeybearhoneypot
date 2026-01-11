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
