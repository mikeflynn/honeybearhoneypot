package db

import (
	"database/sql"
	"time"
)

func EventInitialization(client *sql.DB) error {
	initializeStmt := `
		CREATE TABLE IF NOT EXISTS events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
	    user TEXT NOT NULL,
	    host TEXT NOT NULL,
	    source TEXT NOT NULL,
	    type TEXT NOT NULL,
	    command TEXT NOT NULL,
	    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
		);`
	return makeWrite(initializeStmt)
}

type Event struct {
	ID        int       `json:"id"`
	User      string    `json:"user"`
	Host      string    `json:"host"`
	Source    string    `json:"source"` // EventSource*
	Type      string    `json:"type"`
	Command   string    `json:"command"`
	Timestamp time.Time `json:"timestamp"`
}

func (e *Event) Save() error {
	insertStmt := `INSERT INTO events (user, host, source, type, command) VALUES (?, ?, ?, ?, ?);`
	return makeWrite(insertStmt, e.User, e.Host, e.Source, e.Type, e.Command)
}

func EventQuery(query string, values ...string) ([]*Event, error) {
	rows, err := makeQuery(query, values...)
	if err != nil {
		return nil, err
	}

	ret := []*Event{}

	defer rows.Close()
	if rows.Next() {
		e := &Event{}
		err = rows.Scan(&e.ID, &e.User, &e.Host, &e.Source, &e.Type, &e.Command, &e.Timestamp)
		if err != nil {
			return nil, err
		}
		ret = append(ret, e)
	}

	return ret, nil
}
