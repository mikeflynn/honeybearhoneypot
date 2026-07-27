package entity

import (
	"testing"

	"github.com/mikeflynn/honeybearhoneypot/internal/db"
)

// seedCTFUsers initializes a throwaway SQLite DB and inserts the given
// username→points pairs.
func seedCTFUsers(t *testing.T, users map[string]int) {
	t.Helper()
	db.Initialize(t.TempDir(), CTFUserInit)
	t.Cleanup(db.Close)
	for name, pts := range users {
		if err := db.MakeWrite("INSERT INTO ctf_users (username, points) VALUES (?, ?)", name, pts); err != nil {
			t.Fatalf("seed %s: %v", name, err)
		}
	}
}

func TestRankFor_TopUserIsRankOne(t *testing.T) {
	seedCTFUsers(t, map[string]int{"alice": 150, "bob": 100, "dave": 40})

	rank, points, found, err := RankFor("alice")
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("alice not found")
	}
	if rank != 1 || points != 150 {
		t.Errorf("alice rank=%d points=%d, want rank=1 points=150", rank, points)
	}
}

func TestRankFor_TiesShareRank(t *testing.T) {
	// bob and carol tie at 100; both sit strictly below alice, so both rank 2.
	// dave, below both, is rank 4 (competition ranking).
	seedCTFUsers(t, map[string]int{"alice": 150, "bob": 100, "carol": 100, "dave": 40})

	carolRank, _, _, err := RankFor("carol")
	if err != nil {
		t.Fatal(err)
	}
	if carolRank != 2 {
		t.Errorf("carol rank=%d, want 2", carolRank)
	}

	daveRank, _, _, err := RankFor("dave")
	if err != nil {
		t.Fatal(err)
	}
	if daveRank != 4 {
		t.Errorf("dave rank=%d, want 4", daveRank)
	}
}

func TestRankFor_UnknownUserNotFound(t *testing.T) {
	seedCTFUsers(t, map[string]int{"alice": 150})

	_, _, found, err := RankFor("ghost")
	if err != nil {
		t.Fatal(err)
	}
	if found {
		t.Error("ghost should not be found")
	}
}

func TestRankFor_UnscoredUserNotFound(t *testing.T) {
	// A registered user with 0 points is absent from the board (Leaderboard
	// filters points > 0), so RankFor must report them as unranked too.
	seedCTFUsers(t, map[string]int{"alice": 150, "newbie": 0})

	_, _, found, err := RankFor("newbie")
	if err != nil {
		t.Fatal(err)
	}
	if found {
		t.Error("unscored user should not be ranked")
	}
}
