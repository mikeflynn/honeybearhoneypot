# Honey Bear Honey Pot

## Project Overview

**Honey Bear Honey Pot** is a whimsical yet functional SSH honeypot written in Go. It features a unique graphical user interface (GUI) built with the [Fyne](https://fyne.io/) toolkit, displaying an animated bear that reacts to attacker activity. The core honeypot functionality simulates a Linux environment, capturing SSH sessions, commands, and login attempts.

### Key Features
- **Dual Interface:** Runs as both a GUI application (monitoring dashboard) and an SSH server (the honeypot).
- **Interactive GUI:** Features an animated bear, real-time connection stats, and an admin panel.
- **Simulated Environment:** Provides a fake file system and common Linux commands to deceive attackers.
- **CTF Support:** Includes functionality for Capture The Flag challenges (configurable flags and tasks).
- **Logging:** Persists activity to a SQLite database.
- **Reverse Tunnel:** Optional support for SSH reverse tunneling.

## Architecture & Technology Stack

- **Language:** Go (Golang)
- **GUI Framework:** [Fyne](https://fyne.io/)
- **SSH Server:** [Charmbracelet Wish](https://github.com/charmbracelet/wish)
- **TUI/Terminal:** [Charmbracelet Bubble Tea](https://github.com/charmbracelet/bubbletea) & [Lipgloss](https://github.com/charmbracelet/lipgloss)
- **Database:** SQLite (`github.com/mattn/go-sqlite3`)

### Directory Structure

- **`main.go`**: Application entry point. Orchestrates the startup of the SSH server and the GUI.
- **`internal/config/`**: Handles configuration via command-line flags and JSON files.
- **`internal/honeypot/`**: Core honeypot logic, including the simulated filesystem, command execution, and session handling.
- **`internal/gui/`**: Fyne-based GUI implementation (window management, assets, themes).
- **`internal/db/`**: Database initialization and connection management.
- **`internal/entity/`**: Data models (e.g., `ctf_user`, `event`, `option`).
- **`misc/`**: Contains design assets and sample configuration.

## Building and Running

### Prerequisites
- Go 1.24+
- CGO enabled (required for Fyne and SQLite)
- Fyne command-line tools (optional, for packaging)

### Local Development
To run the application directly from source:
```bash
go run main.go
```

**Common Flags:**
- `-no-gui`: Run only the SSH honeypot server (headless mode).
- `-ssh-port <port>`: Specify the listening port (default: 1337).
- `-log-level <level>`: Set logging verbosity (debug, info, warn, error).
- `-fs`: Start GUI in full-screen mode.

Example:
```bash
go run main.go -ssh-port 2222 -log-level debug
```

### Building
To build a binary for the current platform:
```bash
go build -o honeybear main.go
```

To build a packaged Fyne application:
```bash
go install fyne.io/fyne/v2/cmd/fyne@latest
fyne build
```

### Cross-Compilation
The project supports cross-compilation using `fyne-cross` and Docker:
```bash
go install github.com/fyne-io/fyne-cross/cmd/fyne-cross@latest
fyne-cross linux
```
Artifacts will be placed in `fyne-cross/dist`.

## Configuration

Configuration can be provided via CLI flags (as seen above) or a JSON file passed with `-config`.

**Sample JSON Config (`misc/config.sample.json`):**
```json
{
  "ssh_ports": ["1337"],
  "log_level": "info",
  "filesystem": [...], // Custom files/directories
  "tasks": [...]       // CTF tasks
}
```

## Development Conventions

- **Style:** Standard Go formatting (`gofmt`).
- **Architecture:** The project separates concerns into `internal` packages. `gui` handles presentation, `honeypot` handles the business logic of the simulation, and `entity` defines the data structures.
- **Assets:** Binary assets (images) are likely bundled or loaded from the `internal/gui/assets` directory.
