package gui

// https://github.com/fyne-io/fyne

import (
	"fmt"
	"math/rand"
	"time"

	fyne "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

var currentBear string

func StartGUI() {
	a := app.New()
	w := a.NewWindow("Honey Bear Honey Pot")

	currentBear = "default"
	main := container.NewVBox(
		getBear(currentBear),
	)
	main.Resize(fyne.NewSize(1280, 720))

	var adminPopup *widget.PopUp
	adminSettingsHeader := widget.NewLabel("SETTINGS")
	adminSettingsHeader.Resize(fyne.NewSize(600, 600))
	adminPopupContent := container.NewVBox(
		container.NewHBox(
			adminSettingsHeader,

			widget.NewButtonWithIcon("", theme.WindowCloseIcon(), func() {
				adminPopup.Hide() // Function to hide the pop-up
			}),
		),
	)
	adminPopupContent.Resize(fyne.NewSize(640, 400))

	adminButton := widget.NewButtonWithIcon("", theme.MenuIcon(), func() {
		if adminPopup == nil {
			adminPopup = widget.NewModalPopUp(adminPopupContent, w.Canvas())
		}
		adminPopup.Show()
	})
	adminButton.Resize(fyne.NewSize(75, 75))
	adminButton.Move(fyne.NewPos(1195, 635))

	w.SetContent(container.New(
		layout.NewStackLayout(),
		main,
		container.NewWithoutLayout(
			adminButton,
		),
	))

	go func() {
		options := []string{"default", "happy", "very_happy", "laugh", "sad", "wtf"}
		for range time.Tick(5 * time.Second) {
			idx := rand.Intn(len(options))
			currentBear = options[idx]
			main.Objects[0] = getBear(currentBear)
			main.Refresh()
		}
	}()

	w.Resize(fyne.NewSize(1280, 720))
	w.SetFixedSize(true)
	w.SetMainMenu(systemMenu())
	w.ShowAndRun()
	shutdown()
}

func getBear(label string) *canvas.Image {
	bear := canvas.NewImageFromFile(fmt.Sprintf("./internal/gui/bears/%s.jpg", label))
	bear.FillMode = canvas.ImageFillStretch
	bear.SetMinSize(fyne.NewSize(1280, 720))

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
