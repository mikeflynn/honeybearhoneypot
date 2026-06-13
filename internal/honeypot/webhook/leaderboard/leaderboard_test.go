package leaderboard

import (
	"encoding/json"
	"testing"

	"github.com/mikeflynn/honeybearhoneypot/internal/entity"
)

func TestBuildPayload_ProjectsRanks(t *testing.T) {
	src := func(limit int) ([]entity.CTFUser, error) {
		return []entity.CTFUser{
			{Username: "alice", Points: 150},
			{Username: "bob", Points: 90},
		}, nil
	}
	p := newPayloadFunc(src, 10)
	got, err := p("solve")
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	var parsed struct {
		Event       string `json:"event"`
		Source      string `json:"source"`
		Timestamp   string `json:"timestamp"`
		Leaderboard []struct {
			Rank     int    `json:"rank"`
			Username string `json:"username"`
			Points   int    `json:"points"`
		} `json:"leaderboard"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("payload not json: %v", err)
	}
	if parsed.Event != "solve" {
		t.Errorf("event = %q", parsed.Event)
	}
	if parsed.Source != "honeybear" {
		t.Errorf("source = %q", parsed.Source)
	}
	if parsed.Timestamp == "" {
		t.Error("timestamp empty")
	}
	if len(parsed.Leaderboard) != 2 {
		t.Fatalf("len = %d, want 2", len(parsed.Leaderboard))
	}
	if parsed.Leaderboard[0].Rank != 1 || parsed.Leaderboard[0].Username != "alice" || parsed.Leaderboard[0].Points != 150 {
		t.Errorf("row 0 = %+v", parsed.Leaderboard[0])
	}
	if parsed.Leaderboard[1].Rank != 2 || parsed.Leaderboard[1].Username != "bob" {
		t.Errorf("row 1 = %+v", parsed.Leaderboard[1])
	}
}

func TestBuildPayload_EmptyLeaderboardEmitsEmptyArray(t *testing.T) {
	src := func(limit int) ([]entity.CTFUser, error) { return nil, nil }
	p := newPayloadFunc(src, 10)
	got, err := p("heartbeat")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(got)
	var parsed map[string]json.RawMessage
	_ = json.Unmarshal(body, &parsed)
	if string(parsed["leaderboard"]) != "[]" {
		t.Errorf("leaderboard = %s, want []", string(parsed["leaderboard"]))
	}
}

func TestBuildPayload_LimitDefaultsTo10WhenZero(t *testing.T) {
	var gotLimit int
	src := func(limit int) ([]entity.CTFUser, error) {
		gotLimit = limit
		return nil, nil
	}
	p := newPayloadFunc(src, 0)
	if _, err := p("solve"); err != nil {
		t.Fatal(err)
	}
	if gotLimit != 10 {
		t.Errorf("limit passed = %d, want 10", gotLimit)
	}
}
