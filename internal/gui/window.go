package gui

// https://github.com/fyne-io/fyne

import (
	"bytes"
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"time"

	"github.com/charmbracelet/log"
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

	currentBear := bears.GetBearByCategory("boot", nil)
	if currentBear == nil {
		fmt.Println("Error loading boot bear")
		os.Exit(1)
	}

	buttonNose := widget.NewButton("", func() {
		log.Debug("Nose poke!")
	})
	buttonNose.Importance = widget.LowImportance

	background := container.NewStack(
		showBear(*currentBear),
		container.New(
			layout.NewGridLayoutWithColumns(3),
			layout.NewSpacer(),
			container.New(
				layout.NewGridLayoutWithRows(3),
				layout.NewSpacer(),
				buttonNose,
				layout.NewSpacer(),
			),
			layout.NewSpacer(),
		),
	)

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
			layout.NewSpacer(),
			container.NewVBox(
				container.NewStack(
					canvas.NewRectangle(theme.Color(theme.ColorNameOverlayBackground)),
					container.NewPadded(
						statCurrentUsers,
					),
				),
				container.NewStack(
					canvas.NewRectangle(theme.Color(theme.ColorNameOverlayBackground)),
					container.NewPadded(
						canvas.NewText("38%", theme.Color(theme.ColorNameForeground)),
					),
				),
				layout.NewSpacer(),
			),
		),
	)

	w.SetContent(container.New(
		layout.NewStackLayout(),
		background,
		dataOverlays,
		functionToolbar,
	))

	go func() {
		for range time.Tick(3 * time.Second) {
			r := rand.Intn(100)
			cat := "standard"
			if r < 10 { // Temporary hack to show an emote every 10th tick
				cat = "emote"
			}

			newBear := bears.GetBearByCategory(cat, nil)
			if newBear == nil {
				newBear = currentBear
			}

			if currentBear != newBear {
				if currentBear.Category != newBear.Category {
					glitch := bears.GetBearByCategory("glitch", nil)
					if glitch != nil {
						background.Objects[0] = showBear(*glitch)
						background.Refresh()
						time.Sleep(175 * time.Millisecond)
					}
				}

				currentBear = newBear
				background.Objects[0] = showBear(*currentBear)
				background.Refresh()
			}

			statCurrentUsers.Text = fmt.Sprintf("%d / %d", honeypot.StatActiveUsers(), honeypot.StatMaxUsers())
			dataOverlays.Refresh()
		}
	}()

	w.Resize(fyne.NewSize(width, height)) // 1280x720 is the default size
	//w.SetFixedSize(true) // Don't allow resizing
	w.SetFullScreen(fullscreen) // Inital full screen state
	w.SetPadded(false)
	w.ShowAndRun()
	shutdown()
}

func showBear(bear Bear) *canvas.Image {
	fileData, err := bear.FileData()
	if err != nil {
		return nil
	}

	image := canvas.NewImageFromReader(bytes.NewReader(fileData), bear.Name)
	image.FillMode = canvas.ImageFillStretch
	image.SetMinSize(fyne.NewSize(width, height))
	image.Move(fyne.NewPos(0, 0))

	return image
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
