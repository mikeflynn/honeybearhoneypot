package db

import (
	"database/sql"
	"fmt"
	"time"
)

func OptionInitialization(client *sql.DB) error {
	initializeStmt := `
		CREATE TABLE IF NOT EXISTS options (
			name TEXT PRIMARY KEY,
			value TEXT,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
		);`
	return makeWrite(initializeStmt)
}

type Option struct {
	Name      string    `json:"name"`
	Value     string    `json:"value"`
	Timestamp time.Time `json:"timestamp"`
}

func (o *Option) Save() error {
	query := `
		INSERT INTO options (name, value)
		VALUES (?, ?)
		ON CONFLICT(name)
		DO UPDATE SET value = excluded.value, timestamp = CURRENT_TIMESTAMP;
	`
	return makeWrite(query, o.Name, o.Value)
}

func (o *Option) Load() error {
	if o.Name == "" {
		return fmt.Errorf("name is required")
	}

	query := `SELECT name, value, timestamp FROM options WHERE name = ?;`
	rows, err := makeQuery(query)
	if err != nil {
		return err
	}

	defer rows.Close()
	if rows.Next() {
		err = rows.Scan(&o.Name, &o.Value, &o.Timestamp)
		if err != nil {
			return err
		}
	}

	return nil
}
