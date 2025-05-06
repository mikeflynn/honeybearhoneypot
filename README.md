# Honey Bear Honey Pot

A SSH honeypot application that has a wimsical GUI application...and, if you prefer, can run on a hart hat!

See more information about this project: [honeybear.hydrox.fun](https://honeybear.hydrox.fun)

## Configuration

[ TO DO ]

## Usage

### The GUI

[ TO DO ]

### The SSH Honey Pot

[ TO DO ]

## Development / Running Locally

To run the application locally, you will need to have Go installed on your machine. Checkout the repo and run

```bash
$ go run main.go -h

Usage of /***/main
  -fs
        Start the gui in full screen mode
  -height int
        The height of the GUI window
  -log-level string
        Log level (debug, info, warn, error, fatal) (default "info")
  -no-gui
        Run the honey pot without the GUI
  -pin-reset string
        Reset the admin PIN to a specific value
  -ssh-port string
        The port to listen on for honey pot SSH connections. Comma separated list for multiple ports. (default "1337")
  -tunnel string
        The user and host to connect to via SSH. Ex: user@server.com:22
  -tunnel-key string
        The SSH key to use to connect to the specified remote host.
  -width int
        The width of the GUI window
```

On first run, it will be a bit slower, but you will see the GUI application pop up.

## Deployment / Exporting

If you would like to build a binary, you will need the [fyne-cross](https://github.com/fyne-io/fyne-cross) application and Docker to cross compile the application for different platforms.

```bash
$ go install github.com/fyne-io/fyne-cross@latest
$ fyne-cross linux
```

The app will be exported to the `fyne-cross/dist` directory as tar file with a Makefile and the application binary.

## Contributing

Please feel free to submit issues, and especially pull requets! 
