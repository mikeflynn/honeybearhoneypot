package ctf

import "testing"

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

func TestSuccessMessage(t *testing.T) {
	t.Run("empty uses default", func(t *testing.T) {
		got := successMessage("", 50)
		want := "🎉 Correct! +50 points! 🎉"
		if got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})

	t.Run("whitespace-only uses default", func(t *testing.T) {
		got := successMessage("   ", 50)
		want := "🎉 Correct! +50 points! 🎉"
		if got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})

	t.Run("custom replaces with points suffix", func(t *testing.T) {
		got := successMessage("The vault is open.", 50)
		want := "The vault is open. +50 points!"
		if got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})
}
