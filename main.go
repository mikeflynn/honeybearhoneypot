package main

// https://github.com/charmbracelet/wish

import (
	"flag"
	"net"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/mikeflynn/honeybearhoneypot/internal/db"
	"github.com/mikeflynn/honeybearhoneypot/internal/entity"
	"github.com/mikeflynn/honeybearhoneypot/internal/gui"
	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot"
	"tailscale.com/tsnet"
)

const (
	appName = "HoneyBearHoneyPot"
)

func main() {
	noGui := flag.Bool("no-gui", false, "Run the honey pot without the GUI")
	noFS := flag.Bool("no-fs", false, "Don't start the gui in full screen mode")
	sshPort := flag.String("ssh-port", "1337", "The port to listen on for honey pot SSH connections")
	widthFlag := flag.Int("width", 0, "The width of the GUI window")
	heightFlag := flag.Int("height", 0, "The height of the GUI window")
	logLevel := flag.String("log-level", "info", "Log level (debug, info, warn, error, fatal)")
	pinOverride := flag.String("pin-reset", "", "Reset the admin PIN to a specific value")
	tailscaleAuthKey := flag.String("tailscale-key", "", "A Tailscale Auth Key to use for public internet access.")
	flag.Parse()

	log.SetLevel(translateLogLevel(*logLevel))
	log.Info("Starting Honey Bear Honey Pot...")

	appConfigDir := setup()

	var additionalListener *net.Listener
	if *tailscaleAuthKey != "" {
		tsSrv := new(tsnet.Server)
		tsSrv.AuthKey = *tailscaleAuthKey
		tsSrv.Hostname = "honeybearhoneypot"
		tsSrv.Dir = appConfigDir
		tsSrv.Ephemeral = true
		tsl, err := tsSrv.ListenFunnel("tcp", ":10000")
		if err != nil {
			log.Fatal("Tailscale failed to connect.", "Err", err)
		}

		additionalListener = &tsl
		defer tsSrv.Close()
	}

	if *noGui == false {
		go func() {
			honeypot.StartHoneyPot(*sshPort, additionalListener)
		}()

		if *pinOverride != "" {
			entity.OptionSet("gui_pin", *pinOverride)
		}

		gui.StartGUI(!*noFS, float32(*widthFlag), float32(*heightFlag))
	} else {
		honeypot.StartHoneyPot(*sshPort, additionalListener)
	}

	cleanup()
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
