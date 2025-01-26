package gui

// https://github.com/fyne-io/fyne

import (
	"bytes"
	"fmt"
	"math/rand"
	"net/url"
	"time"

	"github.com/mikeflynn/hardhat-honeybear/internal/gui/assets"
	"github.com/mikeflynn/hardhat-honeybear/internal/honeypot"

	fyne "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	version       = "v0.1.0"
	defaultWidth  = 800
	defaultHeight = 480
)

var (
	currentBear string
	w           fyne.Window
	width       float32 = defaultWidth
	height      float32 = defaultHeight
)

func StartGUI(fullscreen bool, overrideWidth, overrideHeight float32) {
	if overrideWidth != 0 {
		width = overrideWidth
	}

	if overrideHeight != 0 {
		height = overrideHeight
	}

	a := app.New()
	a.Settings().SetTheme(&touchTheme{})

	w = a.NewWindow("Honey Bear Honey Pot")

	currentBear = "sleeping"
	background := container.NewStack(
		getBear(currentBear),
	)
	//background.Resize(fyne.NewSize(1280, 720))

	functionToolbar := container.NewPadded(
		container.NewHBox(
			layout.NewSpacer(),
			container.NewVBox(
				layout.NewSpacer(),
				aboutButton(),
				getAdminButton(),
			),
		),
	)

	statCurrentUsers := canvas.NewText(fmt.Sprintf("%d / %d", 0, honeypot.StatMaxUsers()), theme.Color(theme.ColorNameForeground))
	dataOverlays := container.NewPadded(
		container.NewHBox(
			container.NewVBox(
				container.NewStack(
					canvas.NewRectangle(theme.Color(theme.ColorNameBackground)),
					container.NewPadded(
						statCurrentUsers,
					),
				),
				container.NewStack(
					canvas.NewRectangle(theme.Color(theme.ColorNameBackground)),
					container.NewPadded(
						canvas.NewText("38%", theme.Color(theme.ColorNameForeground)),
					),
				),
				layout.NewSpacer(),
			),
			layout.NewSpacer(),
		),
	)

	w.SetContent(container.New(
		layout.NewStackLayout(),
		background,
		dataOverlays,
		functionToolbar,
	))

	go func() {
		options := []string{"angry", "cool", "happy", "laughing", "look_left", "look_right", "sad", "surprised", "terminator"}
		// Other Bear Ideas: Winking, More Looking, Concerned Looking, Blushing?, ...
		glitches := []string{"glitch_001", "glitch_002", "glitch_003", "glitch_004"}

		for range time.Tick(5 * time.Second) {
			idx := rand.Intn(len(options))
			newBear := options[idx]
			if currentBear != newBear {
				currentBear = newBear

				glitch := glitches[rand.Intn(len(glitches))]
				background.Objects[0] = getBear(glitch)
				background.Refresh()

				time.Sleep(150 * time.Millisecond)
				background.Objects[0] = getBear(currentBear)
				background.Refresh()
			}

			statCurrentUsers.Text = fmt.Sprintf("%d / %d", honeypot.StatActiveUsers(), honeypot.StatMaxUsers())
			dataOverlays.Refresh()
		}
	}()

	w.Resize(fyne.NewSize(width, height)) // 1280x720 is the default size
	//w.SetFixedSize(true) // Don't allow resizing
	w.SetFullScreen(fullscreen) // Inital full screen state
	//w.SetMainMenu(systemMenu()) // Menu takes a lot of space on linux
	w.ShowAndRun()
	shutdown()
}

func getBear(label string) *canvas.Image {
	bearData, err := assets.Images.ReadFile("bear_" + label + ".jpg")
	if err != nil {
		return nil
	}

	bear := canvas.NewImageFromReader(bytes.NewReader(bearData), label)
	bear.FillMode = canvas.ImageFillStretch
	bear.SetMinSize(fyne.NewSize(width, height))
	bear.Move(fyne.NewPos(0, 0))

	return bear
}

func systemMenu() *fyne.MainMenu {
	return fyne.NewMainMenu(
		fyne.NewMenu(
			"Host",
			fyne.NewMenuItem("Restart", nil),
			fyne.NewMenuItem("Shutdown", nil),
		),
		fyne.NewMenu(
			"Honey Pot",
			fyne.NewMenuItem("Start", nil),
			fyne.NewMenuItem("Stop", nil),
		),
		fyne.NewMenu(
			"Help",
			fyne.NewMenuItem("___", nil),
		))
}

func shutdown() {
	// Run any clean up tasks here
	return
}

func aboutButton() *widget.Button {
	var logo *canvas.Image

	logoData, err := assets.Images.ReadFile("hydrox.png")
	if err != nil {
		fmt.Println("Error loading logo:", err)
		return nil
	}

	logo = canvas.NewImageFromReader(bytes.NewReader(logoData), "hydrox")
	logo.FillMode = canvas.ImageFillStretch
	logo.SetMinSize(fyne.NewSize(300, 350))

	link, _ := url.Parse("https://hydrox.fun")

	aboutButton := widget.NewButtonWithIcon("", theme.HelpIcon(), func() {
		var aboutPopup *widget.PopUp
		aboutPopup = widget.NewModalPopUp(
			container.NewVBox(
				container.NewHBox(
					widget.NewLabel("About"),
					layout.NewSpacer(),
					widget.NewButtonWithIcon("", theme.WindowCloseIcon(), func() {
						aboutPopup.Hide()
					}),
				),
				container.NewHBox(
					logo,
					container.NewVBox(
						widget.NewRichTextWithText("Honey Bear Honey Pot: "+version),
						widget.NewSeparator(),
						widget.NewRichTextWithText("Another useless hydrox project."),
						widget.NewHyperlink("hydrox.fun", link),
						widget.NewRichTextWithText("License: MIT"),
					),
				),
			),
			w.Canvas(),
		)
		aboutPopup.Show()
	})
	aboutButton.Importance = widget.LowImportance

	return aboutButton
}
