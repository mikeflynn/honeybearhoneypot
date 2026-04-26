# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Honey Bear Honey Pot is an SSH honeypot written in Go with a Fyne-based GUI that displays an animated bear reacting to attacker activity. It simulates a Linux environment ("Hardhat Linux") using Charmbracelet Wish/Bubble Tea for the SSH server and terminal UI.

## Build & Run Commands

```bash
# Run from source (launches GUI + SSH server on port 1337)
go run main.go

# Run headless (no GUI)
go run main.go -no-gui -ssh-port 2222 -log-level debug

# Run with config file
go run main.go -config config.json

# Run tests
go test ./...

# Run a single package's tests
go test ./internal/honeypot/filesystem/

# Format code
go fmt ./...

# Build binary (requires CGO for SQLite + Fyne)
go build -o honeybear main.go

# Build packaged Fyne app
fyne build

# Cross-compile (requires fyne-cross + Docker)
fyne-cross linux
```

## Architecture

The app has two concurrent subsystems started from `main.go`: the SSH honeypot server and the Fyne GUI (GUI runs on main goroutine, honeypot in a goroutine; reversed with `-no-gui`).

**Key packages:**

- **`internal/config`** — CLI flags + JSON config file parsing. Merged config stored in `config.Active` global for runtime access. Defaults defined in `config.Default`.
- **`internal/db`** — SQLite connection management. Initialized with migration functions passed from `entity` package. DB stored in user config dir (`~/Library/Application Support/HoneyBearHoneyPot/` on macOS).
- **`internal/entity`** — Data models (`Event`, `Option`, CTF users/tasks) with direct SQL operations. Acts as a simple ORM over the db layer.
- **`internal/gui`** — Fyne GUI: animated bear face, notifications, user count, admin panel with PIN-protected menu. Assets bundled in `gui/assets/`.
- **`internal/honeypot`** — SSH server (Wish), session handling, command execution (`exec.go`), rate limiting, stats, reverse tunneling.
- **`internal/honeypot/filesystem`** — Fake filesystem with configurable additional nodes from config. Supports autocomplete.
- **`internal/honeypot/simulation`** — Simulated command output (ps, netstat, etc.).
- **`internal/honeypot/ctf`** — Capture The Flag system with user registration and flag submission.
- **`internal/honeypot/embedded`** — Embedded text files (banner, help, fake patch notes) loaded via `go:embed`.

## Key Technical Details

- CGO is required (SQLite via `mattn/go-sqlite3` and Fyne GUI toolkit).
- Go 1.24+ required.
- The honeypot accepts any username/password — all auth attempts are logged.
- Config merges JSON file values first, then CLI flags override. See `config.sample.json` for schema.
- `config.hardhat.json` contains the production deployment config with CTF tasks and flags — do not commit secrets or real tunnel credentials.
- Environment variables in config values are supported via `${VAR_NAME}` syntax.

## Coding Conventions

- Standard Go formatting (`go fmt`).
- Follow existing code style and naming conventions.
- `misc/design/` contains design assets only — ignore for code changes.
- Never commit to git. The user will handle all version control and deployment manually. Focus on local development and testing.
