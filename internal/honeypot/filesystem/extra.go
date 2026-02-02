package filesystem

import (
	"errors"
	"path"
	"strings"

	"github.com/charmbracelet/log"
)

var additionalNodes []Node
var noFun bool = false

// SetAdditionalNodes stores nodes that will be merged into the filesystem
// during initialization.
func SetAdditionalNodes(nodes []Node) {
	additionalNodes = nodes
}

func SetNoFun(enabled bool) {
	log.Infof("Setting noFun to %v", enabled)
	noFun = enabled
}

// addNode inserts a node into the filesystem tree under its parent path.
// Parent directories must already exist.
func addNode(n Node) error {
	if SystemRoot == nil {
		return errors.New("filesystem not initialized")
	}
	if n.Path == "" {
		return errors.New("node path required")
	}

	if n.Name == "" {
		n.Name = path.Base(n.Path)
	}
	parentPath := path.Dir(n.Path)
	if parentPath == "." {
		parentPath = "/"
	}
	parent, err := GetNodeByPath(SystemRoot, strings.TrimPrefix(parentPath, "/"))
	if err != nil {
		return err
	}

	if n.Owner == "" {
		n.Owner = "root"
	}
	if n.Group == "" {
		n.Group = "root"
	}
	if n.Mode == 0 {
		if n.Directory {
			n.Mode = 0755
		} else {
			n.Mode = 0644
		}
	}

	parent.Children = append(parent.Children, &n)
	return nil
}

func applyAdditionalNodes() {
	for _, n := range additionalNodes {
		_ = addNode(n)
	}
}
