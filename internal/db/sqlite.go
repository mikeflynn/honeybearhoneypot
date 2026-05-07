package db

import (
	"database/sql"
	"errors"
	"log"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

var ErrNotInitialized = errors.New("db: not initialized")

const (
	dbFilename = "database.db"
)

var (
	client *sql.DB
	dbPath string
)

func Initialize(appConfigDir string, initQueries ...string) {
	// Initialize the database
	var err error
	dbPath = filepath.Join(appConfigDir, dbFilename)
	client, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal(err)
	}

	// Create the tables
	for _, query := range initQueries {
		MakeWrite(query)
	}
}

func GetDBPath() string {
	return dbPath
}

func MakeQuery(query string, values ...any) (*sql.Rows, error) {
	if client == nil {
		return nil, ErrNotInitialized
	}
	rows, err := client.Query(query, values...)
	if err != nil {
		return nil, err
	}

	//defer rows.Close()
	return rows, nil
}

func MakeWrite(query string, values ...any) error {
	if client == nil {
		return ErrNotInitialized
	}
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
