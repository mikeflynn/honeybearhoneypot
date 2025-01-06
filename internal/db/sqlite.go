package db

import (
	"database/sql"
	"log"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

const (
	dbFilename        = "database.db"
	EventSourceSystem = "system"
	EventSourceUser   = "user"
)

type EventLog struct {
	User      string    `json:"user"`
	IP        string    `json:"ip"`
	Source    string    `json:"source"` // EventSource*
	Type      string    `json:"type"`
	Command   string    `json:"command"`
	Timestamp time.Time `json:"timestamp"`
}

type Option struct {
	Name      string    `json:"name"`
	Value     string    `json:"value"`
	Timestamp time.Time `json:"timestamp"`
}

var client *sql.DB

func Initialize(appConfigDir string) {
	// Initialize the database
	client, err := sql.Open("sqlite3", filepath.Join(appConfigDir, dbFilename))
	if err != nil {
		log.Fatal(err)
	}

	// Create the tables
	initializeStmt := ``
	_, err = client.Exec(initializeStmt)
	if err != nil {
		log.Fatal(err)
	}
}

func Query(query string) (*sql.Rows, error) {
	rows, err := client.Query(query)
	if err != nil {
		return nil, err
	}

	//defer rows.Close()
	return rows, nil
}

func Insert() {

}

func Update() {

}

func Delete() {

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
