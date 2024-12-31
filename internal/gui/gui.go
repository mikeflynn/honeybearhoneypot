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

	//hello := widget.NewLabel("Hello Fyne!")

	currentBear = "default"
	main := container.NewVBox(
		getBear(currentBear),
	)
	main.Resize(fyne.NewSize(1280, 720))

	adminMenu := container.NewVBox(
		widget.NewLabel("Admin"),
	)
	adminMenu.Resize(fyne.NewSize(200, 600))
	adminMenu.Move(fyne.NewPos(1080, 10))
	adminMenu.Hide()

	adminButton := widget.NewButtonWithIcon("", theme.MenuIcon(), func() {
		if adminMenu.Hidden {
			adminMenu.Show()
		} else {
			adminMenu.Hide()
		}
	})
	adminButton.Resize(fyne.NewSize(75, 75))
	adminButton.Move(fyne.NewPos(1195, 635))

	w.SetContent(container.New(
		layout.NewStackLayout(),
		main,
		container.NewWithoutLayout(
			adminMenu,
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
	w.ShowAndRun()
}

func getBear(label string) *canvas.Image {
	bear := canvas.NewImageFromFile(fmt.Sprintf("./internal/gui/bears/%s.jpg", label))
	bear.FillMode = canvas.ImageFillStretch
	bear.SetMinSize(fyne.NewSize(1280, 720))

	return bear
}
