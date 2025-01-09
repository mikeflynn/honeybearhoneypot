package db

import (
	"database/sql"
	"log"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

const (
	dbFilename        = "database.db"
	EventSourceSystem = "system"
	EventSourceUser   = "user"
)

var client *sql.DB

func Initialize(appConfigDir string) {
	// Initialize the database
	client, err := sql.Open("sqlite3", filepath.Join(appConfigDir, dbFilename))
	if err != nil {
		log.Fatal(err)
	}

	// Create the tables
	EventInitialization(client)
	OptionInitialization(client)
}

func makeQuery(query string, values ...string) (*sql.Rows, error) {
	rows, err := client.Query(query, values)
	if err != nil {
		return nil, err
	}

	//defer rows.Close()
	return rows, nil
}

func makeWrite(query string, values ...string) error {
	_, err := client.Exec(query, values)
	if err != nil {
		return err
	}

	return nil
}

func Close() {
	if client == nil {
		return
	}

	err := client.Close()
	if err != nil {
		log.Fatal(err)
	}
}
