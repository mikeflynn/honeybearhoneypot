package filesystem

import (
	"testing"
)

func TestAllDirectories(t *testing.T) {
	Initialize()

	dirs := AllDirectories()
	if len(dirs) == 0 {
		t.Fatal("AllDirectories returned empty slice")
	}

	// Every returned node must be a directory.
	for _, n := range dirs {
		if !n.IsDirectory() {
			t.Errorf("AllDirectories returned non-directory: %s", n.Path)
		}
		if n.IsCloaked() {
			t.Errorf("AllDirectories returned cloaked node: %s", n.Path)
		}
	}

	// Root must be included.
	root := GetRoot()
	found := false
	for _, n := range dirs {
		if n == root {
			found = true
			break
		}
	}
	if !found {
		t.Error("AllDirectories did not include root")
	}

	// Some well-known dirs from the seeded filesystem must be reachable.
	wantPaths := []string{"/home/you", "/etc", "/var"}
	for _, want := range wantPaths {
		seen := false
		for _, n := range dirs {
			if n.Path == want {
				seen = true
				break
			}
		}
		if !seen {
			t.Errorf("AllDirectories missing expected dir %q", want)
		}
	}
}

func TestChangeDirMsgHasSilentField(t *testing.T) {
	// Compile-time check that the Silent field exists.
	_ = ChangeDirMsg{Path: "/tmp", Silent: true}
}
