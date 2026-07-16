package ctf

import (
	"strings"
	"testing"
)

func TestPartitionTasks(t *testing.T) {
	tasks := []Task{
		{Name: "a", Points: 1, Archived: false, Completed: false},
		{Name: "b", Points: 2, Archived: false, Completed: true},
		{Name: "c", Points: 3, Archived: true, Completed: true},
		{Name: "d", Points: 4, Archived: true, Completed: false},
	}

	active, archivedDone := partitionTasks(tasks)

	if len(active) != 2 || active[0].Name != "a" || active[1].Name != "b" {
		t.Fatalf("active mismatch: %+v", active)
	}
	if len(archivedDone) != 1 || archivedDone[0].Name != "c" {
		t.Fatalf("archivedDone mismatch: %+v", archivedDone)
	}
}

func TestPartitionTasksEmpty(t *testing.T) {
	active, archived := partitionTasks(nil)
	if len(active) != 0 || len(archived) != 0 {
		t.Fatalf("expected empty slices, got %d active, %d archived", len(active), len(archived))
	}
}

func TestSuccessBanner(t *testing.T) {
	if got := successBanner(""); got != "Challenge Complete!" {
		t.Fatalf("empty: got %q", got)
	}
	if got := successBanner("   "); got != "Challenge Complete!" {
		t.Fatalf("whitespace: got %q", got)
	}
	if got := successBanner("  The vault is open.  "); got != "The vault is open." {
		t.Fatalf("custom: got %q", got)
	}
}

func TestTallyValue(t *testing.T) {
	if got := tallyValue(0, 50, 0, 10); got != 0 {
		t.Fatalf("frame 0: got %d", got)
	}
	if got := tallyValue(0, 50, -3, 10); got != 0 {
		t.Fatalf("negative frame: got %d", got)
	}
	if got := tallyValue(0, 50, 5, 10); got != 25 {
		t.Fatalf("midpoint: got %d", got)
	}
	if got := tallyValue(0, 50, 10, 10); got != 50 {
		t.Fatalf("end frame: got %d", got)
	}
	if got := tallyValue(0, 50, 99, 10); got != 50 {
		t.Fatalf("past end: got %d", got)
	}
	if got := tallyValue(0, 50, 5, 0); got != 50 {
		t.Fatalf("zero duration: got %d", got)
	}
	if got := tallyValue(100, 150, 5, 10); got != 125 {
		t.Fatalf("nonzero start midpoint: got %d", got)
	}
}

func TestDottedLeader(t *testing.T) {
	got := dottedLeader("BONUS", "+50", 32)
	if len([]rune(got)) != 32 {
		t.Fatalf("width: got len %d (%q)", len([]rune(got)), got)
	}
	if !strings.HasPrefix(got, "BONUS ") || !strings.HasSuffix(got, " +50") {
		t.Fatalf("format: got %q", got)
	}
	if !strings.Contains(got, "..") {
		t.Fatalf("expected dots: got %q", got)
	}
	cramped := dottedLeader("BONUS", "+50", 3)
	if !strings.Contains(cramped, ".") {
		t.Fatalf("cramped: expected a dot, got %q", cramped)
	}
}
