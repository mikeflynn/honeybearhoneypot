package gui

// https://github.com/fyne-io/fyne

import (
	"bytes"
	"fmt"
	"image/color"
	rand "math/rand/v2"
	"net/url"
	"os"
	"time"

	"github.com/mikeflynn/hardhat-honeybear/internal/entity"
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
	currentBear   string
	overrideBear  string
	emotionFactor int = 1
	w             fyne.Window
	width         float32 = defaultWidth
	height        float32 = defaultHeight
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
		subcat := "react"
		overrideBear = bears.GetBearByCategory("standard", &subcat).Name
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
				/*
					container.NewStack(
						canvas.NewRectangle(theme.Color(theme.ColorNameOverlayBackground)),
						container.NewPadded(
							canvas.NewText("38%", theme.Color(theme.ColorNameForeground)),
						),
					),
				*/
				layout.NewSpacer(),
			),
		),
	)

	notifications := container.NewVBox()

	w.SetContent(container.New(
		layout.NewStackLayout(),
		background,
		dataOverlays,
		container.NewPadded(
			container.NewHBox(
				notifications,
				layout.NewSpacer(),
			),
		),
		functionToolbar,
	))

	// Pot Event Channel
	eventChan := entity.EventSubscribe("notifications")
	nq := &notificationQueue{maxLength: 5, maxAge: 30 * time.Second}

	go func() {
		for {
			select {
			case event := <-eventChan:
				if event == nil {
					break
				}

				nq.Push(event)

				//log.Info("Event: ", "User", event.User, "Host", event.Host, "App", event.App, "Source", event.Source, "Type", event.Type, "Action", event.Action, "Timestamp", event.Timestamp)

				emotionFactor += 1
			case <-time.After(60 * time.Second):
				emotionFactor = 1
			}
		}
	}()

	// UI update loop
	go func() {
		for range time.Tick(2 * time.Second) {
			// Randomly change the bear
			r := rand.IntN(100)
			cat := "standard"
			subcat := "idle"
			if r < emotionFactor {
				cat = "emote"
			}

			var newBear *Bear
			if overrideBear != "" {
				newBear = bears.GetBear(overrideBear)
				overrideBear = ""
			} else {
				newBear = bears.GetBearByCategory(cat, &subcat)
			}

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

			// Update the current user count
			statCurrentUsers.Text = fmt.Sprintf("%d / %d", honeypot.StatActiveUsers(), honeypot.StatMaxUsers())
			dataOverlays.Refresh()

			// Update the notifications
			notifications.RemoveAll()
			for _, container := range nq.Draw() {
				notifications.Add(container)
			}
			notifications.Refresh()
		}
	}()

	w.Resize(fyne.NewSize(width, height)) // 1280x720 is the default size
	//w.SetFixedSize(true) // Don't allow resizing
	w.SetFullScreen(fullscreen) // Inital full screen state
	w.SetPadded(false)
	w.ShowAndRun()

	// Cleanup
	entity.EventUnsubscribe("notifications")
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

func maxStringLen(s string, l int) string {
	if len(s) > l {
		return s[:l-3] + "..."
	}

	return s
}

type notificationQueue struct {
	notifications []*entity.Event
	maxLength     int
	maxAge        time.Duration
}

func (n *notificationQueue) Push(event *entity.Event) {
	if len(n.notifications) >= n.maxLength {
		n.Pop()
	}

	n.notifications = append(n.notifications, event)
}

func (n *notificationQueue) Pop() *entity.Event {
	if len(n.notifications) == 0 {
		return nil
	}

	event := n.notifications[0]
	n.notifications = n.notifications[1:]

	return event
}

func (n *notificationQueue) Draw() []*fyne.Container {
	containers := []*fyne.Container{}

	for i := len(n.notifications) - 1; i >= 0; i-- {
		event := n.notifications[i]
		if event.Timestamp.Add(n.maxAge).Before(time.Now()) {
			continue
		}

		fontSize := float32(18)

		bg := canvas.NewRectangle(color.RGBA{255, 255, 255, 64})
		bg.Resize(fyne.NewSize(240, 40))

		from := canvas.NewText(maxStringLen(fmt.Sprintf("%s@%s", event.User, event.Host), 25), color.Black)
		from.TextSize = fontSize
		from.TextStyle = fyne.TextStyle{Bold: true}

		what := canvas.NewText(maxStringLen(fmt.Sprintf("> %s", event.Action), 25), color.Black)
		what.TextSize = fontSize
		what.TextStyle = fyne.TextStyle{Bold: true}

		containers = append(containers, container.NewStack(
			//canvas.NewRectangle(theme.Color(theme.ColorNameOverlayBackground)),
			bg,
			container.NewPadded(
				container.NewVBox(
					from,
					what,
				),
			),
		))
	}

	return containers
}
