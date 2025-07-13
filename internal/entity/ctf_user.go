package entity

import (
	"fmt"
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
	ID        int
	Username  string
	Password  string
	Points    int
	CreatedAt time.Time
}

type CTFUserTask struct {
	ID        int
	Username  string
	Task      string
	Points    int
	CreatedAt time.Time
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

// Leaderboard returns the top users ordered by points.
func Leaderboard(limit int) ([]CTFUser, error) {
	rows, err := db.MakeQuery("SELECT username, points FROM ctf_users ORDER BY points DESC LIMIT ?", limit)
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

// TaskNameUsed checks if any user has previously completed a task with the given name.
func TaskNameUsed(name string) (bool, error) {
	rows, err := db.MakeQuery("SELECT 1 FROM ctf_user_tasks WHERE task=? LIMIT 1", name)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	return rows.Next(), nil
}
