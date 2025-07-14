package config

import (
	"encoding/json"
	"flag"
	"os"
	"strings"

	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot/filesystem"
)

type Task struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Flag        string `json:"flag"`
	Points      int    `json:"points"`
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
	PinReset   string            `json:"-"`
}

var (
	configPath    = flag.String("config", "", "Path to optional JSON config file")
	noGuiFlag     = flag.Bool("no-gui", false, "Run the honey pot without the GUI")
	fullScreen    = flag.Bool("fs", false, "Start the gui in full screen mode")
	sshPort       = flag.String("ssh-port", "", "The port to listen on for honey pot SSH connections. Comma separated list for multiple ports.")
	widthFlag     = flag.Int("width", 0, "The width of the GUI window")
	heightFlag    = flag.Int("height", 0, "The height of the GUI window")
	logLevelFlag  = flag.String("log-level", "", "Log level (debug, info, warn, error, fatal)")
	pinResetFlag  = flag.String("pin-reset", "", "Reset the admin PIN to a specific value")
	tunnelHost    = flag.String("tunnel", "", "The user and host to connect to via SSH. Ex: user@server.com:22")
	tunnelKeyFlag = flag.String("tunnel-key", "", "The SSH key to use to connect to the specified remote host.")
)

// Active holds the configuration loaded via Parse so it can be referenced by
// other packages at runtime.
var Active *Config

// Default contains the base configuration values used when no CLI flags or config file options are provided.
var Default = Config{
	SSHPorts: []string{"1337"},
	LogLevel: "info",
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

// Parse reads CLI flags and an optional JSON configuration file, returning the
// merged settings along with the value of the pin reset flag.
func Parse() (*Config, string, error) {
	flag.Parse()

	cfg := Default

	if *configPath != "" {
		loaded, err := Load(*configPath)
		if err != nil {
			return nil, "", err
		}
		merge(&cfg, loaded)
	}

	if *sshPort != "" {
		cfg.SSHPorts = strings.Split(*sshPort, ",")
	}
	if *tunnelHost != "" {
		cfg.Tunnel = *tunnelHost
	}
	if *tunnelKeyFlag != "" {
		cfg.TunnelKey = *tunnelKeyFlag
	}
	if *noGuiFlag {
		cfg.NoGUI = true
	}
	if *fullScreen {
		cfg.FullScreen = true
	}
	if *widthFlag != 0 {
		cfg.Width = *widthFlag
	}
	if *heightFlag != 0 {
		cfg.Height = *heightFlag
	}
	if *logLevelFlag != "" {
		cfg.LogLevel = *logLevelFlag
	}

	cfg.PinReset = *pinResetFlag

	Active = &cfg
	return Active, cfg.PinReset, nil
}

func merge(dst *Config, src *Config) {
	if len(src.SSHPorts) > 0 {
		dst.SSHPorts = src.SSHPorts
	}
	if src.Tunnel != "" {
		dst.Tunnel = src.Tunnel
	}
	if src.TunnelKey != "" {
		dst.TunnelKey = src.TunnelKey
	}
	if src.NoGUI {
		dst.NoGUI = true
	}
	if src.FullScreen {
		dst.FullScreen = true
	}
	if src.Width != 0 {
		dst.Width = src.Width
	}
	if src.Height != 0 {
		dst.Height = src.Height
	}
	if src.LogLevel != "" {
		dst.LogLevel = src.LogLevel
	}
	if src.Filesystem != nil {
		dst.Filesystem = src.Filesystem
	}
	if src.Tasks != nil {
		dst.Tasks = src.Tasks
	}
}
