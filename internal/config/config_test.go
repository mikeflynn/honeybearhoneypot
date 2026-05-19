package config

import (
	"encoding/json"
	"testing"
)

func TestTaskArchivedJSON(t *testing.T) {
	t.Run("archived true round-trips", func(t *testing.T) {
		in := Task{Name: "x", Flag: "f", Points: 1, Archived: true}
		b, err := json.Marshal(in)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		var out Task
		if err := json.Unmarshal(b, &out); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if !out.Archived {
			t.Fatalf("expected Archived=true, got false; json=%s", string(b))
		}
	})

	t.Run("archived omitted defaults false", func(t *testing.T) {
		var out Task
		if err := json.Unmarshal([]byte(`{"name":"x","flag":"f","points":1}`), &out); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if out.Archived {
			t.Fatalf("expected Archived=false when omitted, got true")
		}
	})

	t.Run("archived false omitted from output", func(t *testing.T) {
		b, err := json.Marshal(Task{Name: "x", Flag: "f", Points: 1})
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if contains(string(b), "archived") {
			t.Fatalf("expected omitempty to drop archived field, got %s", string(b))
		}
	})
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
