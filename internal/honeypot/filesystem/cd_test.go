package filesystem

import (
	"strings"
	"testing"
)

func TestCdExecRejectsFile(t *testing.T) {
	Initialize()
	home, err := GetNodeByPath(SystemRoot, "home/you")
	if err != nil || home == nil {
		t.Fatalf("failed to resolve /home/you: %v", err)
	}

	cdNode, err := GetNodeByPath(home, "/usr/bin/cd")
	if err != nil || cdNode == nil {
		t.Fatalf("failed to resolve /usr/bin/cd: %v", err)
	}

	cmd, err := cdNode.Run(home, []string{"/home/you/patch.md"}, "you", "you", map[string]string{})
	if err != nil {
		t.Fatalf("cd.Run returned error: %v", err)
	}
	if cmd == nil {
		t.Fatal("expected a command, got nil")
	}

	msg := (*cmd)()
	out, ok := msg.(OutputMsg)
	if !ok {
		t.Fatalf("expected OutputMsg for cd into a file, got %T (%v)", msg, msg)
	}
	if !strings.Contains(string(out), "Not a directory") {
		t.Errorf("expected 'Not a directory' error, got %q", string(out))
	}
}

func TestCdExecAllowsDirectory(t *testing.T) {
	Initialize()
	home, err := GetNodeByPath(SystemRoot, "home/you")
	if err != nil || home == nil {
		t.Fatalf("failed to resolve /home/you: %v", err)
	}

	cdNode, err := GetNodeByPath(home, "/usr/bin/cd")
	if err != nil || cdNode == nil {
		t.Fatalf("failed to resolve /usr/bin/cd: %v", err)
	}

	cmd, err := cdNode.Run(home, []string{"/etc"}, "you", "you", map[string]string{})
	if err != nil {
		t.Fatalf("cd.Run returned error: %v", err)
	}
	if cmd == nil {
		t.Fatal("expected a command, got nil")
	}

	msg := (*cmd)()
	changeMsg, ok := msg.(ChangeDirMsg)
	if !ok {
		t.Fatalf("expected ChangeDirMsg for cd into a directory, got %T (%v)", msg, msg)
	}
	if changeMsg.Node == nil || !changeMsg.Node.Directory {
		t.Errorf("expected resulting node to be a directory")
	}
}
