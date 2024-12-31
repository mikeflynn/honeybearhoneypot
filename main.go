package main

// https://github.com/charmbracelet/wish

import (
	"github.com/mikeflynn/hardhat-honeybear/internal/gui"
	"github.com/mikeflynn/hardhat-honeybear/internal/honeypot"
)

func main() {
	gui.StartGUI()

	honeypot.StartHoneyPot()
}
