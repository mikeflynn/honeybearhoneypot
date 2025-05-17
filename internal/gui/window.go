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

	"github.com/charmbracelet/log"
	"github.com/mikeflynn/honeybearhoneypot/internal/entity"
	"github.com/mikeflynn/honeybearhoneypot/internal/gui/assets"
	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot"

	fyne "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	version       = "v1.0.1"
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
	liveImage     *canvas.Image
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

	currentBear := bears.GetBearByCategory("boot", "")
	if currentBear == nil {
		fmt.Println("Error loading boot bear")
		os.Exit(1)
	}

	buttonNose := widget.NewButton("", func() {
		overrideBear = bears.GetBearByCategory("standard", "react").Name
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

	statCurrentUsers := canvas.NewText("----", theme.Color(theme.ColorNameForeground))
	statCurrentUsers.TextStyle.Monospace = true
	statTotalUsers := canvas.NewText("----", theme.Color(theme.ColorNameForeground))
	statTotalUsers.TextStyle.Monospace = true
	status := container.NewVBox(
		textLabel("NOW", 12),
		statCurrentUsers,
		textLabel("ALL TIME", 12),
		statTotalUsers,
	)

	dataOverlays := container.NewPadded(
		container.NewHBox(
			layout.NewSpacer(),
			container.NewVBox(
				container.NewStack(
					canvas.NewRectangle(theme.Color(theme.ColorNameOverlayBackground)),
					container.NewPadded(
						status,
					),
				),
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

				log.Debug(
					"Event: ",
					"User", event.User,
					"Host", event.Host,
					"App", event.App,
					"Source", event.Source,
					"Type", event.Type,
					"Action", event.Action,
					"Timestamp", event.Timestamp,
				)

				emotionFactor += 1
			case <-time.After(60 * time.Second):
				emotionFactor -= 1
			}

			log.Debug("Emotion Status:", "factor", emotionFactor)
		}
	}()

	// UI update loop
	go func() {
		for {
			loopWait := time.Duration(2 * time.Second) // Default wait in seconds

			// Randomly change the bear
			cat := "standard"
			subcat := "idle"

			if shouldShowEmotion() {
				cat = "emote"
				subcat = ""
			}

			var newBear *Bear
			if overrideBear != "" {
				newBear = bears.GetBear(overrideBear)
			} else {
				newBear = bears.GetBearByCategory(cat, subcat)
			}

			if newBear == nil {
				log.Debug("No bear found. Using current bear.")
				newBear = currentBear
			} else {
				if overrideBear == "" && currentBear.Category != newBear.Category {
					overrideBear = newBear.Name // Set the new bear to load after the glitch
					log.Debug("Loading glitch bear")
					newBear = bears.GetBearByCategory("glitch", "")

					loopWait = time.Duration(600 * time.Millisecond) // Brief wait for the glitch bear
				} else {
					currentBear = newBear // Set the current bear to the new bear
					overrideBear = ""     // Reset the override bear
				}
			}

			fyne.Do(func() {
				// Update the current user count
				statCurrentUsers.Text = fmt.Sprintf("%04d", honeypot.StatActiveUsers())
				statTotalUsers.Text = fmt.Sprintf("%04d", honeypot.StatUsersAllTime())

				status.Objects = []fyne.CanvasObject{
					textLabel("NOW", 12),
					statCurrentUsers,
					textLabel("ALL TIME", 12),
					statTotalUsers,
				}

				// Update the tunnel button
				tunnelStatus := tunnelStatus()
				if tunnelStatus != nil {
					// Prepend the tunnel status to the status.Objects
					status.Objects = append([]fyne.CanvasObject{tunnelStatus, separator()}, status.Objects...)
				}

				dataOverlays.Refresh()

				// Update the notifications
				notifications.RemoveAll()
				for _, container := range nq.Draw() {
					notifications.Add(container)
				}
				notifications.Refresh()

				// Update the bear
				background.Objects[0] = showBear(*newBear)
				background.Refresh()
			})

			time.Sleep(loopWait)
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

func textLabel(text string, fontSize int) *canvas.Text {
	label := canvas.NewText(text, theme.Color(theme.ColorNameForeground))
	label.TextSize = float32(fontSize)
	label.TextStyle = fyne.TextStyle{Bold: true}

	return label
}

func separator() *canvas.Rectangle {
	sep := canvas.NewRectangle(theme.Color(theme.ColorNameDisabled))
	sep.SetMinSize(fyne.NewSize(1, 1))
	sep.Resize(fyne.NewSize(1, 20))

	return sep
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

func shouldShowEmotion() bool {
	max := (honeypot.StatActiveUsers() * 3) + (honeypot.StatMaxUsers() * 2)
	r := rand.IntN(max)
	log.Debug("Bear update:", "max", max, "r", r, "emotionFactor", emotionFactor)

	resp := r <= emotionFactor
	if resp {
		emotionFactor -= 10
		if emotionFactor < 0 {
			emotionFactor = 1
		}
	}

	return resp
}

func aboutButton() *widget.Button {
	var logo *canvas.Image

	logoData, err := assets.Images.ReadFile("qr.png")
	if err != nil {
		fmt.Println("Error loading logo:", err)
		return nil
	}

	logo = canvas.NewImageFromReader(bytes.NewReader(logoData), "qr")
	logo.FillMode = canvas.ImageFillStretch
	logo.SetMinSize(fyne.NewSize(300, 300))

	link, _ := url.Parse("https://honeybear.hydrox.fun")

	aboutButton := widget.NewButtonWithIcon("", theme.HelpIcon(), func() {
		var aboutPopup *widget.PopUp
		aboutPopup = widget.NewModalPopUp(
			container.NewVBox(
				container.NewHBox(
					widget.NewLabel("Honey Bear Honey Pot: "+version),
					layout.NewSpacer(),
					widget.NewButtonWithIcon("", theme.WindowCloseIcon(), func() {
						aboutPopup.Hide()
					}),
				),
				container.NewHBox(
					logo,
					container.NewVBox(
						widget.NewRichTextWithText("Questions?\nCheck out the website\nfor answers, build process,\nand how to connect!"),
						widget.NewSeparator(),
						widget.NewHyperlink("honeybear.hydrox.fun", link),
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

func tunnelStatus() *canvas.Image {
	status := honeypot.StatTunnelActive()
	if status == -1 {
		return nil
	}

	if status == 1 {
		if liveImage == nil {
			liveBytes, err := assets.Images.ReadFile("live.png")
			if err != nil {
				fmt.Println("Error loading logo:", err)
				return canvas.NewImageFromResource(theme.ErrorIcon())
			}

			liveImage = canvas.NewImageFromReader(bytes.NewReader(liveBytes), "live")
			liveImage.FillMode = canvas.ImageFillStretch
			liveImage.SetMinSize(fyne.NewSize(20, 25))
		}

		return liveImage
	}

	return nil
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
