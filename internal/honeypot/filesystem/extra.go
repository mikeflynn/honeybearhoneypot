package filesystem

import (
	"errors"
	"path"
	"strings"

	"charm.land/log/v2"
)

var additionalNodes []Node
var noFun bool = false

// CurlResponse is a single fake URL-to-body mapping consumed by curlExec.
type CurlResponse struct {
	URL  string `json:"url"`
	Body string `json:"body"`
}

// NmapPort is a single open-port entry for a fake host.
type NmapPort struct {
	Port    int    `json:"port"`
	Service string `json:"service"`
	Version string `json:"version,omitempty"`
}

// NmapHost is a single fake host with its open ports.
type NmapHost struct {
	IP    string     `json:"ip"`
	Ports []NmapPort `json:"ports"`
}

var (
	curlResponseBodies map[string]string
	nmapHostsByIP      map[string]NmapHost
)

// SetCurlResponses installs the curl response table used by curlExec.
// URLs are pre-normalized for O(1) lookup.
func SetCurlResponses(responses []CurlResponse) {
	m := make(map[string]string, len(responses))
	for _, r := range responses {
		key, _ := normalizeURL(r.URL)
		m[key] = r.Body
	}
	curlResponseBodies = m
}

// SetNmapHosts installs the nmap host table used by nmapExec, keyed by IP
// for O(1) lookup when expanding range scans.
func SetNmapHosts(hosts []NmapHost) {
	m := make(map[string]NmapHost, len(hosts))
	for _, h := range hosts {
		m[h.IP] = h
	}
	nmapHostsByIP = m
}

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

	n.Path = path.Clean(n.Path)
	n.Name = path.Base(n.Path)
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

	for i, child := range parent.Children {
		if child.Name == n.Name {
			parent.Children[i] = &n
			return nil
		}
	}
	parent.Children = append(parent.Children, &n)
	return nil
}

func applyAdditionalNodes() {
	for _, n := range additionalNodes {
		if err := addNode(n); err != nil {
			log.Errorf("filesystem: failed to add node %q: %v", n.Path, err)
		}
	}
}
