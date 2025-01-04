package gui

// https://github.com/fyne-io/fyne

import (
	"bytes"
	"fmt"
	"math/rand"
	"time"

	"github.com/mikeflynn/hardhat-honeybear/internal/gui/assets"

	fyne "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

var currentBear string
var w fyne.Window

func StartGUI() {
	a := app.New()
	w = a.NewWindow("Honey Bear Honey Pot")

	currentBear = "default"
	background := container.NewStack(
		getBear(currentBear),
	)
	//background.Resize(fyne.NewSize(1280, 720))

	overlays := container.NewPadded(
		container.NewHBox(
			layout.NewSpacer(),
			container.NewVBox(
				layout.NewSpacer(),
				aboutButton(),
				getAdminButton(),
			),
		),
	)

	w.SetContent(container.New(
		layout.NewStackLayout(),
		background,
		overlays,
	))

	go func() {
		options := []string{"default", "happy", "very_happy", "laugh", "sad", "wtf"}
		for range time.Tick(5 * time.Second) {
			idx := rand.Intn(len(options))
			currentBear = options[idx]
			background.Objects[0] = getBear(currentBear)
			background.Refresh()
		}
	}()

	w.Resize(fyne.NewSize(1280, 720))
	//w.SetFixedSize(true) // Don't allow resizing
	//w.SetFullScreen(true) // Mandate full screen
	w.SetMainMenu(systemMenu())
	w.ShowAndRun()
	shutdown()
}

func getBear(label string) *canvas.Image {
	bearData, err := assets.Images.ReadFile(label + ".jpg")
	if err != nil {
		return nil
	}

	bear := canvas.NewImageFromReader(bytes.NewReader(bearData), label)
	bear.FillMode = canvas.ImageFillStretch
	bear.SetMinSize(fyne.NewSize(1280, 720))
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
	logo.SetMinSize(fyne.NewSize(300, 300))

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
						widget.NewLabel("The Honey Bear Honey Pot"),
						widget.NewLabel("v0.1.0"),
						widget.NewLabel("Another useless hydrox project."),
						widget.NewLabel("https://hydrox.fun"),
						widget.NewLabel("License: MIT"),
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
