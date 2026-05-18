package entity

import (
	"fmt"
	"strconv"
	"time"

	"charm.land/log/v2"
	"github.com/mikeflynn/honeybearhoneypot/internal/db"
)

const (
	KeyAdminPIN        = "gui_pin"
	KeyPotSSHPort      = "pot_ssh_port"
	KeyPotMaxUsers     = "pot_max_users"
	KeyRateLimitWindow = "pot_rate_limit_window"
	KeyRateLimitMax    = "pot_rate_limit_max"
	KeyRateLimitBan    = "pot_rate_limit_ban"
)

func OptionInitialization() string {
	return `
		CREATE TABLE IF NOT EXISTS options (
			name TEXT PRIMARY KEY,
			value TEXT,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
		);`
}

func OptionGet(name string) string {
	o := &Option{Name: name}
	err := o.Load()
	if err != nil {
		log.Error("OptionGet Error", "name", name, "error", err)
		return ""
	}

	return o.Value
}

func OptionGetInt(name string) int {
	val := OptionGet(name)
	if val == "" {
		return 0
	}

	intval, err := strconv.Atoi(val)
	if err != nil {
		log.Error("OptionGetInt Error", "name", name, "error", err)
		return 0
	}

	return intval
}

func OptionSet(name, value string) {
	o := &Option{Name: name, Value: value}
	err := o.Save()
	if err != nil {
		log.Error("OptionSet Error", "name", name, "val", value, "error", err)
	}
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
	rows, err := db.MakeQuery(query, o.Name)
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

func OptionsAll() ([]*Option, error) {
	query := `SELECT name, value, timestamp FROM options;`
	rows, err := db.MakeQuery(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var options []*Option
	for rows.Next() {
		o := &Option{}
		err = rows.Scan(&o.Name, &o.Value, &o.Timestamp)
		if err != nil {
			return nil, err
		}
		options = append(options, o)
	}
	return options, nil
}
