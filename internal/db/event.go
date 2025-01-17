package db

import (
	"database/sql"
	"time"
)

const (
	EventSourceSystem = "system"
	EventSourceUser   = "user"
)

func EventInitialization(client *sql.DB) error {
	initializeStmt := `
		PRAGMA user_version = 1;
		CREATE TABLE IF NOT EXISTS events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
	    user TEXT NOT NULL,
	    host TEXT NOT NULL,
			app TEXT NOT NULL,
	    source TEXT NOT NULL,
	    type TEXT NOT NULL,
	    action TEXT NOT NULL,
	    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		`
	return makeWrite(initializeStmt)
}

type Event struct {
	ID        int       `json:"id"`
	User      string    `json:"user"`
	Host      string    `json:"host"`
	App       string    `json:"app"`
	Source    string    `json:"source"` // EventSource*
	Type      string    `json:"type"`
	Action    string    `json:"action"`
	Timestamp time.Time `json:"timestamp"`
}

func (e *Event) Save() error {
	insertStmt := `INSERT INTO events (user, host, app, source, type, action) VALUES (?, ?, ?, ?, ?, ?);`
	return makeWrite(insertStmt, e.User, e.Host, e.App, e.Source, e.Type, e.Action)
}

func EventQuery(query string, values ...any) ([]*Event, error) {
	rows, err := makeQuery(query, values...)
	if err != nil {
		return nil, err
	}

	ret := []*Event{}

	defer rows.Close()
	if rows.Next() {
		e := &Event{}
		err = rows.Scan(&e.ID, &e.User, &e.Host, &e.App, &e.Source, &e.Type, &e.Action, &e.Timestamp)
		if err != nil {
			return nil, err
		}
		ret = append(ret, e)
	}

	return ret, nil
}
