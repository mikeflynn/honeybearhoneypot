package filesystem

import (
	"sort"
	"testing"
)

func TestAutoComplete(t *testing.T) {
	Initialize()

	root := GetRoot()
	home, _ := GetNodeByPath(root, "/home/you")

	// Test 1: Command completion (ls from /usr/bin)
	// "l" -> "ls "
	// Note: there might be other commands starting with l (less, lsb_release, leaderboard).
	// ls, less, lsb_release, leaderboard. Common prefix is "l".
	// Wait, "leaderboard", "less", "ls", "lsb_release".
	// Common prefix is "l".
	// So "l" -> "l" with multiple matches.

	res, matches := AutoComplete(home, "l")
	if res != "l" {
		t.Errorf("Expected 'l', got '%s'", res)
	}
	if len(matches) < 2 {
		t.Errorf("Expected multiple matches for 'l', got %d", len(matches))
	}

	// Test specific command: "cle" -> "clear "
	res, matches = AutoComplete(home, "cle")
	if res != "clear " {
		t.Errorf("Expected 'clear ', got '%s'", res)
	}

	// Test 2: Argument completion (local file)
	// "cat pat" -> "cat patch.md "
	res, matches = AutoComplete(home, "cat pat")
	if res != "cat patch.md " {
		t.Errorf("Expected 'cat patch.md ', got '%s'", res)
	}

	// Test 3: Directory completion
	// "cd .s" -> "cd .ssh/"
	res, matches = AutoComplete(home, "cd .s")
	if res != "cd .ssh/" {
		t.Errorf("Expected 'cd .ssh/', got '%s'", res)
	}

	// Test 4: Absolute path
	// "ls /ho" -> "ls /home/"
	res, matches = AutoComplete(home, "ls /ho")
	if res != "ls /home/" {
		t.Errorf("Expected 'ls /home/', got '%s'", res)
	}

	// Test 5: Ambiguous with common prefix
	// Manually add files to home
	home.Children = append(home.Children, newFile("/home/you/foo1", []byte{}, 0644))
	home.Children = append(home.Children, newFile("/home/you/foo2", []byte{}, 0644))

	// "cat fo" -> "cat foo" (common prefix)
	res, matches = AutoComplete(home, "cat fo")
	if res != "cat foo" {
		t.Errorf("Expected 'cat foo', got '%s'", res)
	}
	if len(matches) != 2 {
		t.Errorf("Expected 2 matches, got %d", len(matches))
	}

	// Verify matches contains foo1 and foo2
	sort.Strings(matches)
	if matches[0] != "foo1 " || matches[1] != "foo2 " {
		t.Errorf("Expected foo1 and foo2, got %v", matches)
	}
}
