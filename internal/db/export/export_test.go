package export

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mikeflynn/honeybearhoneypot/internal/db"
	"github.com/mikeflynn/honeybearhoneypot/internal/entity"
)

func TestExportDatabase(t *testing.T) {
	// Setup temporary directory for DB
	dbDir, err := os.MkdirTemp("", "honeybear_db_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dbDir)

	// Setup temporary directory for Export
	exportDir, err := os.MkdirTemp("", "honeybear_export_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(exportDir)

	// Initialize DB
	db.Initialize(dbDir, entity.EventInitialization(), entity.OptionInitialization(), entity.CTFUserInit)
	defer db.Close()

	// Insert dummy data
	evt := &entity.Event{
		User:   "testuser",
		Host:   "127.0.0.1",
		App:    "ssh",
		Source: entity.EventSourceUser,
		Type:   "login",
		Action: "success",
	}
	if err := evt.Save(); err != nil {
		t.Fatal(err)
	}

	entity.OptionSet("test_opt", "value123")

	ctfUser := &entity.CTFUser{
		Username: "hacker1",
		Points:   100,
	}
	if err := ctfUser.Save(); err != nil {
		t.Fatal(err)
	}

	// Test JSON Export
	err = ExportDatabase([]DataType{}, JSON, exportDir)
	if err != nil {
		t.Errorf("JSON Export failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(exportDir, "events.json")); os.IsNotExist(err) {
		t.Error("events.json not created")
	}
	if _, err := os.Stat(filepath.Join(exportDir, "options.json")); os.IsNotExist(err) {
		t.Error("options.json not created")
	}
	if _, err := os.Stat(filepath.Join(exportDir, "ctf.json")); os.IsNotExist(err) {
		t.Error("ctf.json not created")
	}

	// Test CSV Export
	err = ExportDatabase([]DataType{}, CSV, exportDir)
	if err != nil {
		t.Errorf("CSV Export failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(exportDir, "events.csv")); os.IsNotExist(err) {
		t.Error("events.csv not created")
	}
	if _, err := os.Stat(filepath.Join(exportDir, "options.csv")); os.IsNotExist(err) {
		t.Error("options.csv not created")
	}
	if _, err := os.Stat(filepath.Join(exportDir, "ctf.csv")); os.IsNotExist(err) {
		t.Error("ctf.csv not created")
	}

	// Test RAW Export
	err = ExportDatabase(nil, RAW, exportDir)
	if err != nil {
		t.Errorf("RAW Export failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(exportDir, "database.db")); os.IsNotExist(err) {
		t.Error("database.db not created (exported)")
	}
}
