package main

// https://github.com/charmbracelet/wish

import (
	"net"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/mikeflynn/honeybearhoneypot/internal/config"
	"github.com/mikeflynn/honeybearhoneypot/internal/db"
	"github.com/mikeflynn/honeybearhoneypot/internal/entity"
	"github.com/mikeflynn/honeybearhoneypot/internal/gui"
	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot"
	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot/filesystem"
)

const (
	appName = "HoneyBearHoneyPot"
)

func main() {
	cfg, _, err := config.Parse()
	if err != nil {
		log.Fatal("Failed to parse configuration", "error", err)
	}

	filesystem.SetAdditionalNodes(cfg.Filesystem)

	log.SetLevel(translateLogLevel(cfg.LogLevel))
	log.Info("Starting Honey Bear Honey Pot...")

	appConfigDir := setup()
	defer cleanup()

	var primaryPort string
	var additionalListeners []*net.Listener
	ports := cfg.SSHPorts
	for x, port := range ports {
		log.Debug("Adding listener", "port", port)
		if x == 0 {
			primaryPort = port
			continue
		}

		if listener, err := net.Listen("tcp", ":"+port); err == nil {
			additionalListeners = append(additionalListeners, &listener)
		} else {
			log.Error("Failed to add listener", "port", port, "err", err)
		}
	}

	honeypot.SetPort(primaryPort)
	host := cfg.Tunnel
	key := cfg.TunnelKey
	honeypot.SetTunnel(&host, &key)
	honeypot.AddListeners(additionalListeners...)

	if !cfg.NoGUI {
		go func() {
			honeypot.StartHoneyPot(appConfigDir)
		}()

		if cfg.PinReset != "" {
			entity.OptionSet("gui_pin", cfg.PinReset)
		}

		gui.StartGUI(cfg.FullScreen, float32(cfg.Width), float32(cfg.Height))
	} else {
		honeypot.StartHoneyPot(appConfigDir)
	}
}

func setup() string {
	// Ensure the app data directory exists
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatal(err)
	}

	appConfigDir := filepath.Join(userConfigDir, appName)
	dirCheck, err := os.Stat(appConfigDir)
	if os.IsNotExist(err) || !dirCheck.IsDir() {
		// Create the directory
		err = os.Mkdir(appConfigDir, 0755)
		if err != nil {
			log.Fatal(err)
		}
	}

	log.Debug("App Data Directory", "path", appConfigDir)

	// Initialize the database
	db.Initialize(
		appConfigDir,
		entity.EventInitialization(),
		entity.OptionInitialization(),
	)

	return appConfigDir
}

func cleanup() {
	// Close the database connection
	db.Close()
}

func translateLogLevel(logLevel string) log.Level {
	switch logLevel {
	case "debug":
		return log.DebugLevel
	case "info":
		return log.InfoLevel
	case "warn":
		return log.WarnLevel
	case "error":
		return log.ErrorLevel
	case "fatal":
		return log.FatalLevel
	default:
		return log.InfoLevel
	}
}
