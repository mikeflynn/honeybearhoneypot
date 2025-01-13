package main

// https://github.com/charmbracelet/wish

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/mikeflynn/hardhat-honeybear/internal/db"
	"github.com/mikeflynn/hardhat-honeybear/internal/gui"
	"github.com/mikeflynn/hardhat-honeybear/internal/honeypot"
)

const (
	appName = "HoneyBearHoneyPot"
)

func main() {
	noGui := flag.Bool("no-gui", false, "Run the honey pot without the GUI")
	flag.Parse()

	setup()

	if *noGui == false {
		go func() {
			honeypot.StartHoneyPot()
		}()

		gui.StartGUI()
	} else {
		honeypot.StartHoneyPot()
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

	log.Printf("App data directory: %s\n", appConfigDir)

	// Initialize the database
	db.Initialize(appConfigDir)
}

func cleanup() {
	// Close the database connection
	db.Close()
}
