package main

// https://github.com/charmbracelet/wish

import (
	_ "embed"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"charm.land/log/v2"
	"github.com/mikeflynn/honeybearhoneypot/internal/config"
	"github.com/mikeflynn/honeybearhoneypot/internal/db"
	"github.com/mikeflynn/honeybearhoneypot/internal/db/export"
	"github.com/mikeflynn/honeybearhoneypot/internal/entity"
	"github.com/mikeflynn/honeybearhoneypot/internal/geo"
	"github.com/mikeflynn/honeybearhoneypot/internal/gui"
	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot"
	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot/filesystem"
)

const (
	appName = "HoneyBearHoneyPot"
)

//go:embed VERSION
var versionFile string

func init() {
	if v := strings.TrimSpace(versionFile); v != "" {
		gui.Version = v
	}
}

func main() {
	for _, arg := range os.Args[1:] {
		if arg == "--version" || arg == "-version" {
			fmt.Println(strings.TrimSpace(versionFile))
			return
		}
	}

	cfg, _, err := config.Parse()
	if err != nil {
		log.Fatal("Failed to parse configuration", "error", err)
	}

	filesystem.SetAdditionalNodes(cfg.Filesystem)
	filesystem.SetCurlResponses(cfg.CurlResponses)
	filesystem.SetNmapHosts(cfg.NmapHosts)

	log.SetLevel(translateLogLevel(cfg.LogLevel))

	appConfigDir := setup()
	defer cleanup()

	// Handle Export
	if cfg.ExportFormat != "" {
		if cfg.ExportPath == "" {
			log.Fatal("Export path is required when export format is specified")
		}

		log.Info("Starting Export...", "format", cfg.ExportFormat, "path", cfg.ExportPath)

		var types []export.DataType
		for _, t := range cfg.ExportTypes {
			types = append(types, export.DataType(t))
		}

		err := export.ExportDatabase(types, export.ExportFormat(cfg.ExportFormat), cfg.ExportPath)
		if err != nil {
			log.Fatal("Export failed", "error", err)
		}

		log.Info("Export completed successfully")
		return
	}

	log.Info("Starting Honey Bear Honey Pot...")

	filesystem.SetNoFun(config.Active.NoFun)

	// Sync config to DB
	if cfg.RateLimitMax != 0 {
		entity.OptionSet(entity.KeyRateLimitMax, strconv.Itoa(cfg.RateLimitMax))
	}
	if cfg.RateLimitWindow != 0 {
		entity.OptionSet(entity.KeyRateLimitWindow, strconv.Itoa(cfg.RateLimitWindow))
	}
	if cfg.RateLimitBan != 0 {
		entity.OptionSet(entity.KeyRateLimitBan, strconv.Itoa(cfg.RateLimitBan))
	}

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
	bind := cfg.TunnelBind
	remotePort := cfg.TunnelRemotePort
	honeypot.SetTunnel(&host, &key, &bind, &remotePort)
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
		entity.CTFUserInit,
		entity.CTFUserTaskInit,
		entity.MigrationFixExecLoginSource,
		entity.GeoCacheInitialization(),
	)

	if err := geo.Init(); err != nil {
		log.Error("geo init failed", "err", err)
	}

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
