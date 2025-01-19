package filesystem

import (
	"errors"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func GetNodeByPath(currentNode *Node, path string, flags ...int) (*Node, error) {
	depth := 0
	if len(flags) > 0 {
		depth = flags[0]
	}

	if currentNode == nil {
		return nil, errors.New("000x00")
	}

	// If the path starts with a /, look from the root
	if path == "" || path == "." {
		return currentNode, nil
	} else if strings.HasPrefix(path, "..") {
		// If the path starts with .., look from the parent directory
		parent := currentNode.Parent()
		if parent == nil {
			parent = currentNode
		}

		if len(path) > 2 {
			return GetNodeByPath(parent, path[3:], depth+1)
		}

		return parent, nil
	} else if strings.HasPrefix(path, "/") {
		// If the path starts with a /, look from the root
		return GetNodeByPath(SystemRoot, path[1:], depth+1)
	} else if strings.HasPrefix(path, "./") {
		// If the path starts with ./, look from the current directory
		return GetNodeByPath(currentNode, path[2:], depth+1)
	} else if !strings.Contains(path, "/") && depth == 0 {
		// If the path doesn't contain a /, look from the current directory and the exec path
		found := currentNode.Child(path)
		if found != nil {
			return found, nil
		}

		// If not found in current directory, look in the exec path
		for _, p := range SystemPath {
			found, _ := GetNodeByPath(currentNode, p+path, depth+1)
			if found != nil {
				return found, nil
			}
		}

		return nil, errors.New("not found")
	} else {
		// Otherwise, look from the current directory recursively
		parts := strings.Split(path, "/")
		if len(parts) == 0 {
			return currentNode, nil
		}

		found := currentNode.Child(parts[0])
		if found != nil && len(parts[1:]) > 0 {
			return GetNodeByPath(found, strings.Join(parts[1:], "/"), depth+1)
		}

		if found != nil {
			return found, nil
		}

		return nil, errors.New("not found")
	}
}

func GetRoot() *Node {
	return SystemRoot
}

func GetContent(currentNode *Node, path string, user string, group string) (string, error) {
	if node, err := GetNodeByPath(currentNode, path); err == nil {
		if !node.IsReadable(user, group) {
			return "", errors.New("not readable")
		}

		return node.Content(), nil
	}

	return "", errors.New("not found")
}

func RunNode(currentNode *Node, path string, params []string, user string, group string) (*tea.Cmd, error) {
	if node, err := GetNodeByPath(currentNode, path); err == nil {
		if !node.IsExecutable(user, group) {
			return nil, errors.New("not executable")
		}

		return node.Run(params)
	}

	return nil, errors.New("not found")
}

type Node struct {
	Name      string
	Path      string
	Directory bool
	Children  []*Node
	AssetName string
	Content   func() string
	Exec      func(*Node, []string) *tea.Cmd
	Owner     string
	Group     string
	Mode      int
}

func (n *Node) IsDirectory() bool {
	return n.Directory
}

func (n *Node) IsFile() bool {
	return n.Directory == false
}

func (n *Node) IsExecutable(user string, group string) bool {
	// Check if it's a file and has an executable function
	if !n.IsFile() || n.Exec != nil {
		return false
	}

	// Check if the user is the owner and has execute permissions
	if user == n.Owner && n.Mode&0100 != 0 {
		return true
	}

	// Check if the user is in the group and has execute permissions
	if group == n.Group && n.Mode&0010 != 0 {
		return true
	}

	// Check if the user is not the owner or in the group and has execute permissions
	if n.Mode&0001 != 0 {
		return true
	}

	return false
}

func (n *Node) IsReadable(user string, group string) bool {
	// Check if the user is the owner and has read permissions
	if user == n.Owner && n.Mode&0400 != 0 {
		return true
	}

	// Check if the user is in the group and has read permissions
	if group == n.Group && n.Mode&0040 != 0 {
		return true
	}

	// Check if the user is not the owner or in the group and has read permissions
	if n.Mode&0004 != 0 {
		return true
	}

	return false
}

func (n *Node) Parent() *Node {
	path := strings.Split(n.Path, "/")
	if len(path) < 2 {
		return nil
	} else if len(path) == 2 {
		return SystemRoot
	}

	res, err := GetNodeByPath(n, strings.Join(path[:len(path)-1], "/"))
	if err != nil {
		return nil
	}

	return res
}

func (n *Node) Child(name string) *Node {
	if n.IsFile() || n.Children == nil || len(n.Children) == 0 {
		return nil
	}

	for _, child := range n.Children {
		if child.Name == name {
			return child
		}
	}

	return nil
}

func (n *Node) Open() (string, error) {
	if n.IsFile() && n.Content != nil {
		return n.Content(), nil
	}

	return "", errors.New("not a file")
}

func (n *Node) Run(params []string) (*tea.Cmd, error) {
	if n.Exec != nil {
		return n.Exec(n, params), nil
	}

	return nil, errors.New("not executable")
}
