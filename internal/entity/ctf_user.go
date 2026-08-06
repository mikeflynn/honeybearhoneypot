package entity

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/mikeflynn/honeybearhoneypot/internal/db"
)

const CTFUserInit = `
CREATE TABLE IF NOT EXISTS ctf_users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT UNIQUE,
    password TEXT,
    points INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
`

const CTFUserTaskInit = `
CREATE TABLE IF NOT EXISTS ctf_user_tasks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL,
    task TEXT NOT NULL,
    points INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(username, task)
);
`

type CTFUser struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Password  string    `json:"-"`
	Points    int       `json:"points"`
	CreatedAt time.Time `json:"created_at"`
}

type CTFUserTask struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Task      string    `json:"task"`
	Points    int       `json:"points"`
	CreatedAt time.Time `json:"created_at"`
}

func (u *CTFUser) Load() error {
	if u.Username == "" {
		return fmt.Errorf("username required")
	}
	row, err := db.MakeQuery("SELECT id, username, password, points, created_at FROM ctf_users WHERE username = ?", u.Username)
	if err != nil {
		return err
	}
	defer row.Close()
	if row.Next() {
		return row.Scan(&u.ID, &u.Username, &u.Password, &u.Points, &u.CreatedAt)
	}
	return fmt.Errorf("user not found")
}

func (u *CTFUser) Save() error {
	query := `INSERT INTO ctf_users (username, password, points) VALUES (?, ?, ?) ON CONFLICT(username) DO UPDATE SET password=excluded.password, points=excluded.points`
	return db.MakeWrite(query, u.Username, u.Password, u.Points)
}

func (u *CTFUser) AddPoints(p int) error {
	u.Points += p
	return db.MakeWrite("UPDATE ctf_users SET points=? WHERE username=?", u.Points, u.Username)
}

func (u *CTFUser) CompleteTask(task string, points int) error {
	if u.Username == "" {
		return fmt.Errorf("username required")
	}

	rows, err := db.MakeQuery("SELECT 1 FROM ctf_user_tasks WHERE username=? AND task=?", u.Username, task)
	if err != nil {
		return err
	}
	defer rows.Close()
	if rows.Next() {
		return fmt.Errorf("task already completed")
	}

	if err := db.MakeWrite("INSERT INTO ctf_user_tasks (username, task, points) VALUES (?, ?, ?)", u.Username, task, points); err != nil {
		return err
	}

	return u.AddPoints(points)
}

// CompletedTasks returns a list of task names the user has finished.
func (u *CTFUser) CompletedTasks() ([]string, error) {
	rows, err := db.MakeQuery("SELECT task FROM ctf_user_tasks WHERE username=?", u.Username)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tasks = append(tasks, name)
	}
	return tasks, nil
}

var (
	leaderboardExcludedUsersMu sync.RWMutex
	leaderboardExcludedUsers   []string
)

// SetLeaderboardExcludedUsers configures usernames that should be hidden from
// the CTF leaderboard and rank calculations (e.g. house accounts, testers).
func SetLeaderboardExcludedUsers(users []string) {
	leaderboardExcludedUsersMu.Lock()
	defer leaderboardExcludedUsersMu.Unlock()
	leaderboardExcludedUsers = append([]string(nil), users...)
}

func excludedUsersClause(args []any) (string, []any) {
	leaderboardExcludedUsersMu.RLock()
	excluded := leaderboardExcludedUsers
	leaderboardExcludedUsersMu.RUnlock()

	if len(excluded) == 0 {
		return "", args
	}

	placeholders := make([]string, len(excluded))
	for i, u := range excluded {
		placeholders[i] = "?"
		args = append(args, u)
	}
	return " AND username NOT IN (" + strings.Join(placeholders, ",") + ")", args
}

// Leaderboard returns the top users ordered by points, excluding any
// usernames configured via SetLeaderboardExcludedUsers.
func Leaderboard(limit int) ([]CTFUser, error) {
	clause, args := excludedUsersClause(nil)
	args = append(args, limit)

	rows, err := db.MakeQuery("SELECT username, points FROM ctf_users WHERE points > 0"+clause+" ORDER BY points DESC LIMIT ?", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []CTFUser
	for rows.Next() {
		var u CTFUser
		if err := rows.Scan(&u.Username, &u.Points); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, nil
}

// RankFor returns the competition rank (1-based; ties share a rank) and point
// total for a single user, so a player outside the top-N board can still find
// their standing. found is false when no such user exists or the user has no
// ranked score (points <= 0), matching Leaderboard's "points > 0" filter.
func RankFor(username string) (rank, points int, found bool, err error) {
	clause, clauseArgs := excludedUsersClause(nil)
	args := append([]any{}, clauseArgs...)
	args = append(args, username)

	rows, err := db.MakeQuery(
		"SELECT points, (SELECT COUNT(*) FROM ctf_users c2 WHERE c2.points > c1.points"+clause+") + 1 AS rank "+
			"FROM ctf_users c1 WHERE username = ? AND points > 0", args...)
	if err != nil {
		return 0, 0, false, err
	}
	defer rows.Close()

	if !rows.Next() {
		return 0, 0, false, nil
	}
	if err := rows.Scan(&points, &rank); err != nil {
		return 0, 0, false, err
	}
	return rank, points, true, nil
}

func CTFUsersAll() ([]*CTFUser, error) {
	rows, err := db.MakeQuery("SELECT id, username, password, points, created_at FROM ctf_users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*CTFUser
	for rows.Next() {
		u := &CTFUser{}
		if err := rows.Scan(&u.ID, &u.Username, &u.Password, &u.Points, &u.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, nil
}
