package db

import (
	"database/sql"
	"log"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

const (
	dbFilename = "database.db"
)

var client *sql.DB

func Initialize(appConfigDir string, initQueries ...string) {
	// Initialize the database
	var err error
	client, err = sql.Open("sqlite3", filepath.Join(appConfigDir, dbFilename))
	if err != nil {
		log.Fatal(err)
	}

	// Create the tables
	for _, query := range initQueries {
		MakeWrite(query)
	}
}

func MakeQuery(query string, values ...any) (*sql.Rows, error) {
	rows, err := client.Query(query, values...)
	if err != nil {
		return nil, err
	}

	//defer rows.Close()
	return rows, nil
}

func MakeWrite(query string, values ...any) error {
	_, err := client.Exec(query, values...)
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
