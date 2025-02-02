package main

// https://github.com/charmbracelet/wish

import (
	"flag"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/mikeflynn/hardhat-honeybear/internal/db"
	"github.com/mikeflynn/hardhat-honeybear/internal/gui"
	"github.com/mikeflynn/hardhat-honeybear/internal/honeypot"
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
	flag.Parse()

	log.SetLevel(translateLogLevel(*logLevel))
	log.Info("Starting Honey Bear Honey Pot...")

	setup()

	if *noGui == false {
		go func() {
			honeypot.StartHoneyPot(*sshPort)
		}()

		gui.StartGUI(!*noFS, float32(*widthFlag), float32(*heightFlag))
	} else {
		honeypot.StartHoneyPot(*sshPort)
	}

	cleanup()
}

func setup() {
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

	log.Debug("App data directory: %s\n", appConfigDir)

	// Initialize the database
	db.Initialize(appConfigDir)
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
