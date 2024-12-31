# Honey Bear Honey Pot

A SSH honeypot application that has a wimsical GUI application.

## Development / Running Locally

To run the application locally, you will need to have Go installed on your machine.

```bash
$ go run main.go
```

On first run, it will be a bit slower, but you will see the GUI application pop up.

## Deployment / Exporting

You will need the fyne-cross application and Docker to cross compile the application for different platforms.

```bash
$ go install github.com/fyne-io/fyne-cross@latest
$ fyne-cross linux
```

The app will be exported to the `fyne-cross/dist` directory as tar file with a Makefile and the application binary.
