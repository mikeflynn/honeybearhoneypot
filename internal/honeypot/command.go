package honeypot

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/shlex"
	"github.com/mikeflynn/hardhat-honeybear/internal/honeypot/embedded"
	"github.com/muesli/reflow/wordwrap"
)

func globalCommandHandler(m *model) *tea.Cmd {
	command := m.textInput.Value()
	parts, err := shlex.Split(command)
	if err != nil {
		m.output += m.outputStyle.Render(fmt.Sprintf("\nError parsing command: %s\n", err))
		return nil
	}

	if len(parts) == 0 {
		return nil
	}

	switch parts[0] {
	case "clear":
		m.output = ""
	case "help":
		helpText, err := embedded.Files.ReadFile("help.txt")
		if err != nil {
			helpText = []byte("\nError reading file.\n")
		}

		m.output += m.outputStyle.Render("\n" + string(helpText))
	case "history":
		max := 10
		if len(m.history) < 10 {
			max = len(m.history)
		}

		list := m.history[:max]

		for i := len(list) - 1; i >= 0; i-- {
			m.output += m.outputStyle.Render(fmt.Sprintf("\n%d: %s", i, list[i]))
		}
	case "cat", "more", "less":
		m.runningCommand = "cat"

		var cmd tea.Cmd
		cmd = func() tea.Msg {
			fileData, err := embedded.Files.ReadFile(parts[1])
			if err != nil {
				return fileContentsMsg(m.outputStyle.Render(fmt.Sprintf("\n%s: No such file or directory\n", err)))
			}

			wrapper := wordwrap.NewWriter(m.width)
			wrapper.Newline = []rune{'\r'}
			wrapper.Breakpoints = []rune{':', ','}
			wrapper.Write(fileData)
			wrapper.Close()

			return fileContentsMsg(wrapper.String())
		}

		return &cmd
	default:
		m.output += m.outputStyle.Render(fmt.Sprintf("\ncommand not found: %s\n", parts[0]))
	}

	return nil
}
