package config

import (
	"encoding/json"
	"os"

	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot/filesystem"
)

type Task struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Flag        string `json:"flag"`
}

type Config struct {
	SSHPorts   []string          `json:"ssh_ports,omitempty"`
	Tunnel     string            `json:"tunnel,omitempty"`
	TunnelKey  string            `json:"tunnel_key,omitempty"`
	NoGUI      bool              `json:"no_gui,omitempty"`
	FullScreen bool              `json:"full_screen,omitempty"`
	Width      int               `json:"width,omitempty"`
	Height     int               `json:"height,omitempty"`
	LogLevel   string            `json:"log_level,omitempty"`
	Filesystem []filesystem.Node `json:"filesystem,omitempty"`
	Tasks      []Task            `json:"tasks,omitempty"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}
