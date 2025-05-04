package filesystem

import (
	"fmt"
	"math/rand/v2"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func bearSayExec(dir *Node, params []string) *tea.Cmd {
	cmds := []tea.Cmd{}
	cmds = append(cmds, tea.Cmd(func() tea.Msg {
		return SetRunningCmd("bearsay")
	}))

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
