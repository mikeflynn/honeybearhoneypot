package main

// https://github.com/charmbracelet/wish

import (
	"flag"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/mikeflynn/honeybearhoneypot/internal/db"
	"github.com/mikeflynn/honeybearhoneypot/internal/entity"
	"github.com/mikeflynn/honeybearhoneypot/internal/gui"
	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot"
)

const (
	appName = "HoneyBearHoneyPot"
)

func main() {
	noGui := flag.Bool("no-gui", false, "Run the honey pot without the GUI")
	noFS := flag.Bool("no-fs", false, "Don't start the gui in full screen mode")
	sshPort := flag.String("ssh-port", "1337", "The port to listen on for honey pot SSH connections. Comma separated list for multiple ports.")
	widthFlag := flag.Int("width", 0, "The width of the GUI window")
	heightFlag := flag.Int("height", 0, "The height of the GUI window")
	logLevel := flag.String("log-level", "info", "Log level (debug, info, warn, error, fatal)")
	pinOverride := flag.String("pin-reset", "", "Reset the admin PIN to a specific value")
	tunnelHost := flag.String("tunnel", "", "The user and host to connect to via SSH. Ex: user@server.com:22")
	tunnelKey := flag.String("tunnel-key", "", "The SSH key to use to connect to the specified remote host.")
	flag.Parse()

	log.SetLevel(translateLogLevel(*logLevel))
	log.Info("Starting Honey Bear Honey Pot...")

	setup()

	var primaryPort string
	var additionalListeners []*net.Listener
	ports := strings.Split(*sshPort, ",")
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
	honeypot.SetTunnel(tunnelHost, tunnelKey)
	honeypot.AddListeners(additionalListeners...)

	if *noGui == false {
		go func() {
			honeypot.StartHoneyPot()
		}()

		if *pinOverride != "" {
			entity.OptionSet("gui_pin", *pinOverride)
		}

		gui.StartGUI(!*noFS, float32(*widthFlag), float32(*heightFlag))
	} else {
		honeypot.StartHoneyPot()
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
