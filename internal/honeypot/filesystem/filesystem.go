package filesystem

import (
	tea "github.com/charmbracelet/bubbletea"
)

var (
	SystemRoot *Node
	SystemPath []string
	HomeDir    *Node
)

type (
	FileContentsMsg string
	OutputMsg       string
	ClearOutputMsg  string
	ChangeDirMsg    struct {
		Path string
		Node *Node
	}
)

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
				Name:      "notes.txt",
				Path:      "/home/you/notes.txt",
				Directory: false,
				Owner:     "you",
				Group:     "default",
				Mode:      0644,
				Content: func() string {
					return "This is a note."
				},
			},
		},
		Owner: "you",
		Group: "default",
	}

	SystemRoot = &Node{
		Name:      "",
		Path:      "/",
		Directory: true,
		Children: []*Node{
			{
				Name:      "opt",
				Path:      "/opt",
				Directory: true,
				Children:  []*Node{},
				Owner:     "root",
				Group:     "root",
			},
			{
				Name:      "root",
				Path:      "/root",
				Directory: true,
				Children:  []*Node{},
				Owner:     "root",
				Group:     "root",
			},
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
								Mode:      0755,
								Exec: func(dir *Node, params []string) *tea.Cmd {
									cmd := tea.Cmd(func() tea.Msg {
										output := ""
										separater := "\t"
										targetPath := "."

										if len(params) > 0 {
											for _, param := range params {
												if param == "-l" {
													separater = "\n"
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
												output += child.Name + separater
											}
										}

										return OutputMsg(output)
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
								Mode:      0755,
								Exec: func(dir *Node, params []string) *tea.Cmd {
									var cmd tea.Cmd
									cmd = func() tea.Msg {
										return ClearOutputMsg("")
									}

									return &cmd
								},
							},
							{
								Name:      "pwd",
								Path:      "/usr/bin/pwd",
								Directory: false,
								Owner:     "root",
								Group:     "root",
								Mode:      0755,
								Exec: func(dir *Node, params []string) *tea.Cmd {
									cmd := tea.Cmd(func() tea.Msg {
										return OutputMsg(dir.Path)
									})

									return &cmd
								},
							},
							{
								Name:      "cd",
								Path:      "/usr/bin/cd",
								Directory: false,
								Owner:     "root",
								Group:     "root",
								Mode:      0755,
								Exec: func(dir *Node, params []string) *tea.Cmd {
									var cmd tea.Cmd
									cmd = func() tea.Msg {
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
				Name:      "var",
				Path:      "/var",
				Directory: true,
				Children:  []*Node{},
				Owner:     "root",
				Group:     "root",
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
			{
				Name:      "tmp",
				Path:      "/tmp",
				Directory: true,
				Children:  []*Node{},
				Owner:     "root",
				Group:     "root",
			},
		},
		Owner: "root",
		Group: "root",
	}
}
