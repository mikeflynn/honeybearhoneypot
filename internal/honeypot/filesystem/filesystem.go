package filesystem

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot/confetti"
	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot/embedded"
	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot/matrix"
)

var (
	SystemRoot *Node
	SystemPath []string
	HomeDir    *Node
)

type (
	FileContentsMsg    []byte
	OutputMsg          string
	ClearOutputMsg     string
	HistoryListMsg     string
	SetRunningCmd      string
	ListActiveUsersMsg string
	ChangeDirMsg       struct {
		Path string
		Node *Node
	}
	TickMsg time.Time
)

func newDirectory(path string, children ...*Node) *Node {
	parts := strings.Split(path, "/")
	return &Node{
		Name:      parts[len(parts)-1],
		Path:      path,
		Owner:     "root",
		Group:     "root",
		Mode:      0755,
		Directory: true,
		Children:  children,
	}
}

func newFile(path string, content []byte, mode int) *Node {
	parts := strings.Split(path, "/")
	return &Node{
		Name:      parts[len(parts)-1],
		Path:      path,
		Owner:     "root",
		Group:     "root",
		Directory: false,
		Content:   func() []byte { return content },
		Mode:      mode,
	}
}

func Initialize() {
	SystemPath = []string{
		"/usr/bin/",
	}

	HomeDir = &Node{
		Name:      "you",
		Path:      "/home/you",
		Directory: true,
		Children: []*Node{
			{
				Name:      "test.txt",
				Path:      "/home/you/test.txt",
				Directory: false,
				Owner:     "you",
				Group:     "default",
				Mode:      0644,
				Content: func() []byte {
					//return []byte("This is a note.")

					fileData, err := embedded.Files.ReadFile("test.txt")
					if err != nil {
						return FileContentsMsg(fmt.Sprintf("\n%s: Error reading file.\n", err))
					}

					return fileData
				},
			},
			newDirectory("/home/you/.ssh"),
		},
		Owner: "you",
		Group: "default",
	}

	catHelp := "Usage: cat [FILE]\n Displays the contents of a file."

	SystemRoot = &Node{
		Name:      "",
		Path:      "/",
		Directory: true,
		Children: []*Node{
			newDirectory("/opt"),
			newDirectory("/root"),
			newDirectory("/var"),
			newDirectory("/tmp"),
			newDirectory(
				"/etc",
				newFile(
					"/etc/os-release",
					[]byte("PRETTY_NAME=\"Hardhat Linux\"\nNAME=\"Hardhat Linux\"\nID=hardhat\nID_LIKE=debian\nVERSION_ID=\"1.0\"\nVERSION=\"1.0\"\nVERSION_CODENAME=\"hardhat\"\n"),
					0644,
				),
			),
			{
				Name:      "usr",
				Path:      "/usr",
				Directory: true,
				Children: []*Node{
					{
						Name:      "bin",
						Path:      "/usr/bin",
						Directory: true,
						Children: []*Node{
							{
								Name:      "ls",
								Path:      "/usr/bin/ls",
								Directory: false,
								Owner:     "root",
								Group:     "root",
								Mode:      0711,
								HelpText:  "Usage: ls [OPTION]... [FILE]...\n List information about the FILEs (the current directory by default).",
								Exec: func(dir *Node, params []string) *tea.Cmd {
									cmd := tea.Cmd(func() tea.Msg {
										output := ""
										separater := "\t"
										targetPath := "."
										showHidden := false

										if len(params) > 0 {
											for _, param := range params {
												if strings.HasPrefix(param, "-") {
													if strings.Contains(param, "l") {
														separater = "\n"
													}

													if strings.Contains(param, "a") {
														showHidden = true
													}
												} else {
													targetPath = param
												}
											}
										}

										targetNode, err := GetNodeByPath(dir, targetPath)
										if err != nil {
											return OutputMsg("ls: cannot access '" + targetPath + "': No such file or directory")
										}

										if targetNode.IsDirectory() && len(targetNode.Children) > 0 {
											for _, child := range targetNode.Children {
												if !showHidden && strings.HasPrefix(child.Name, ".") {
													continue
												}

												output += child.Name + separater
											}
										}

										return OutputMsg(output)
									})

									return &cmd
								},
							},
							{
								Name:      "w",
								Path:      "/usr/bin/w",
								Directory: false,
								Owner:     "root",
								Group:     "root",
								Mode:      0711,
								HelpText:  "w - Show who is logged on and what they are doing.",
								Exec: func(dir *Node, params []string) *tea.Cmd {
									cmd := tea.Cmd(func() tea.Msg {
										return ListActiveUsersMsg("")
									})

									return &cmd
								},
							},
							{
								Name:      "clear",
								Path:      "/usr/bin/clear",
								Directory: false,
								Owner:     "root",
								Group:     "root",
								Mode:      0711,
								HelpText:  "Usage: clear\n Clear the terminal screen.",
								Exec: func(dir *Node, params []string) *tea.Cmd {
									var cmd tea.Cmd
									cmd = func() tea.Msg {
										return ClearOutputMsg("")
									}

									return &cmd
								},
							},
							{
								Name:      "bearsay",
								Path:      "/usr/bin/bearsay",
								Directory: false,
								Owner:     "root",
								Group:     "root",
								Mode:      0711,
								HelpText:  "Usage: clear\n Clear the terminal screen.",
								Exec:      bearSayExec,
							},
							{
								Name:      "cowsay",
								Path:      "/usr/bin/cowsay",
								Directory: false,
								Owner:     "root",
								Group:     "root",
								Mode:      0711,
								HelpText:  "configurable speaking/thinking bear (and a bit more)",
								Exec:      bearSayExec,
							},
							{
								Name:      "echo",
								Path:      "/usr/bin/echo",
								Directory: false,
								Owner:     "root",
								Group:     "root",
								Mode:      0711,
								HelpText:  "configurable speaking/thinking bear (and a bit more)",
								Exec: func(dir *Node, params []string) *tea.Cmd {
									var cmd tea.Cmd
									cmd = func() tea.Msg {
										return OutputMsg(strings.Join(params, " "))
									}

									return &cmd
								},
							},
							{
								Name:      "ping",
								Path:      "/usr/bin/ping",
								Directory: false,
								Owner:     "root",
								Group:     "root",
								Mode:      0711,
								HelpText:  "Usage: ping\n Send a ping, get a pong.",
								Exec: func(dir *Node, params []string) *tea.Cmd {
									var cmd tea.Cmd
									cmd = func() tea.Msg {
										return OutputMsg("Pong!")
									}

									return &cmd
								},
							},
							{
								Name:      "man",
								Path:      "/usr/bin/man",
								Directory: false,
								Owner:     "root",
								Group:     "root",
								Mode:      0711,
								HelpText:  "Usage: man [COMMAND]\n Display the manual page for a command.",
								Exec: func(dir *Node, params []string) *tea.Cmd {
									var cmd tea.Cmd
									cmd = func() tea.Msg {
										return OutputMsg("No man. Just use -h or --help on the command you want to learn about.")
									}

									return &cmd
								},
							},
							{
								Name:      "help",
								Path:      "/usr/bin/help",
								Directory: false,
								Owner:     "root",
								Group:     "root",
								Mode:      0711,
								HelpText:  "Usage: help\n Display this help text.",
								Exec: func(dir *Node, params []string) *tea.Cmd {
									cmds := []tea.Cmd{}
									cmds = append(cmds, tea.Cmd(func() tea.Msg {
										return SetRunningCmd("cat")
									}))

									cmds = append(cmds, tea.Cmd(func() tea.Msg {
										helpText, err := embedded.Files.ReadFile("help.txt")
										if err != nil {
											helpText = []byte("\nError reading file.\n")
										}

										return FileContentsMsg(helpText)
									}))

									batch := tea.Batch(cmds...)
									return &batch
								},
							},
							{
								Name:      "pwd",
								Path:      "/usr/bin/pwd",
								Directory: false,
								Owner:     "root",
								Group:     "root",
								Mode:      0711,
								HelpText:  "Usage: pwd\n Print the name of the current working directory.",
								Exec: func(dir *Node, params []string) *tea.Cmd {
									cmd := tea.Cmd(func() tea.Msg {
										return OutputMsg(dir.Path)
									})

									return &cmd
								},
							},
							{
								Name:      "history",
								Path:      "/usr/bin/history",
								Directory: false,
								Owner:     "root",
								Group:     "root",
								Mode:      0711,
								HelpText:  "Usage: history\n Display the command history.",
								Exec: func(dir *Node, params []string) *tea.Cmd {
									cmd := tea.Cmd(func() tea.Msg {
										return HistoryListMsg("")
									})

									return &cmd
								},
							},
							{
								Name:      "cat",
								Path:      "/usr/bin/cat",
								Directory: false,
								Owner:     "root",
								Group:     "root",
								Mode:      0711,
								Exec:      catExec,
								HelpText:  catHelp,
							},
							{
								Name:      "less",
								Path:      "/usr/bin/less",
								Directory: false,
								Owner:     "root",
								Group:     "root",
								Mode:      0711,
								Exec:      catExec,
								HelpText:  catHelp,
							},
							{
								Name:      "more",
								Path:      "/usr/bin/more",
								Directory: false,
								Owner:     "root",
								Group:     "root",
								Mode:      0711,
								Exec:      catExec,
								HelpText:  catHelp,
							},
							{
								Name:      "celebrate",
								Path:      "/usr/bin/celebrate",
								Directory: false,
								Owner:     "root",
								Group:     "root",
								Mode:      0711,
								Exec: func(dir *Node, params []string) *tea.Cmd {
									cmds := []tea.Cmd{}
									cmds = append(cmds, tea.Cmd(func() tea.Msg {
										return SetRunningCmd("confetti")
									}))

									// Start the confetti animation after a short delay for the previous command to finish.
									cmds = append(cmds, tea.Cmd(func() tea.Msg {
										time.Sleep(time.Millisecond * 100)
										return confetti.Burst()
									}))

									// Reset the running command after the confetti has finished.
									cmds = append(cmds, tea.Cmd(func() tea.Msg {
										time.Sleep(time.Second * 4)
										return SetRunningCmd("")
									}))

									batch := tea.Batch(cmds...)
									return &batch
								},
							},
							{
								Name:      "matrix",
								Path:      "/usr/bin/matrix",
								Directory: false,
								Owner:     "root",
								Group:     "root",
								Mode:      0711,
								Exec: func(dir *Node, params []string) *tea.Cmd {
									cmds := []tea.Cmd{}
									cmds = append(cmds, tea.Cmd(func() tea.Msg {
										return SetRunningCmd("matrix")
									}))

									cmds = append(cmds, tea.Cmd(func() tea.Msg {
										time.Sleep(time.Millisecond * 100)
										return matrix.Start()
									}))

									batch := tea.Batch(cmds...)
									return &batch
								},
							},
							{
								Name:      "cd",
								Path:      "/usr/bin/cd",
								Directory: false,
								Owner:     "root",
								Group:     "root",
								Mode:      0711,
								HelpText:  "Usage: cd [DIRECTORY]\n Change the shell working directory.",
								Exec: func(dir *Node, params []string) *tea.Cmd {
									var cmd tea.Cmd
									cmd = func() tea.Msg {
										if len(params) == 0 {
											params = []string{"/home/you"}
										}

										newPath := params[0]
										newNode, err := GetNodeByPath(dir, newPath)
										if err != nil {
											return OutputMsg(err.Error())
										}

										return ChangeDirMsg{
											Path: newPath,
											Node: newNode,
										}
									}

									return &cmd
								},
							},
							{
								Name:      "uname",
								Path:      "/usr/bin/uname",
								Directory: false,
								Owner:     "root",
								Group:     "root",
								Mode:      0711,
								HelpText:  "Usage: uname [OPTION]...\n Print system information.",
								Exec: func(dir *Node, params []string) *tea.Cmd {
									s := "Linux"
									n := "Hardhat"
									r := "6.22.0-81-generic"
									v := "#148-HardHat SMP Fri Mar 14 19:05:48 UTC 2025"
									m := "x86_64"
									o := "Hardhat Linux"
									output := []string{}

									if len(params) == 0 {
										output = append(output, o)
									} else {
										for _, param := range params {
											switch param {
											case "-a":
												output = append(output, fmt.Sprintf("%s %s %s %s %s %s", s, n, r, v, m, o))
											case "-s":
												output = append(output, s)
											case "-n":
												output = append(output, n)
											case "-r":
												output = append(output, r)
											case "-v":
												output = append(output, v)
											case "-m", "-p", "-i":
												output = append(output, m)
											case "-o":
												output = append(output, o)
											default:
												output = []string{fmt.Sprintf("uname: invalid option -- '%s'", param)}
											}
										}
									}

									cmd := tea.Cmd(func() tea.Msg {
										return OutputMsg(strings.Join(output, " "))
									})

									return &cmd
								},
							},
							{
								Name:      "lsb_release",
								Path:      "/usr/bin/lsb_release",
								Directory: false,
								Owner:     "root",
								Group:     "root",
								Mode:      0711,
								HelpText:  "w - Show who is logged on and what they are doing.",
								Exec: func(dir *Node, params []string) *tea.Cmd {
									cmd := tea.Cmd(func() tea.Msg {
										if len(params) == 0 {
											return OutputMsg("No LSB modules are available.")
										}

										for _, param := range params {
											switch param {
											case "-a":
												return OutputMsg("Distributor ID: Hardhat\nDescription: Hardhat Linux 1.0\nRelease: 1.0\nCodename: hardhat")
											case "-d":
												return OutputMsg("Description: Hardhat Linux 1.0")
											case "-r":
												return OutputMsg("Release: 1.0")
											case "-c":
												return OutputMsg("Codename: hardhat")
											case "-i":
												return OutputMsg("Distributor ID: Hardhat")
											case "-s":
												return OutputMsg("Hardhat")
											case "-v":
												return OutputMsg("Hardhat Linux 1.0")
											default:
												return OutputMsg("lsb_release: invalid option -- '" + param + "'")
											}
										}

										return OutputMsg("lsb_release: no options provided")
									})

									return &cmd
								},
							},
						},
						Owner: "root",
						Group: "root",
					},
					{
						Name:      "local",
						Path:      "/usr/local",
						Directory: true,
						Children:  []*Node{},
						Owner:     "root",
						Group:     "root",
					},
				},
				Owner: "root",
				Group: "root",
			},
			{
				Name:      "home",
				Path:      "/home",
				Directory: true,
				Children: []*Node{
					HomeDir,
				},
				Owner: "root",
				Group: "root",
			},
		},
		Owner: "root",
		Group: "root",
	}

	applyAdditionalNodes()
}
