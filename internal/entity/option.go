package entity

import (
	"fmt"
	"time"

	"github.com/mikeflynn/hardhat-honeybear/internal/db"
)

func OptionInitialization() string {
	return `
		CREATE TABLE IF NOT EXISTS options (
			name TEXT PRIMARY KEY,
			value TEXT,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
		);`
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
	return db.MakeWrite(query, o.Name, o.Value)
}

func (o *Option) Load() error {
	if o.Name == "" {
		return fmt.Errorf("name is required")
	}

	query := `SELECT name, value, timestamp FROM options WHERE name = ?;`
	rows, err := db.MakeQuery(query)
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
