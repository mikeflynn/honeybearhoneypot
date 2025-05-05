package gui

import (
	"fmt"
	"os"
	"time"

	fyne "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/charmbracelet/log"
	"github.com/mikeflynn/honeybearhoneypot/internal/entity"
	"github.com/mikeflynn/honeybearhoneypot/internal/gui/keypad"
)

const (
	maxAuthSeconds = 30
	authSuccess    = 1
	authFailure    = 0
	authCancel     = -1
)

var adminPopup *widget.PopUp
var authPopup *widget.PopUp

func getAdminButton() *widget.Button {
	adminButton := widget.NewButtonWithIcon("", theme.MenuIcon(), func() {
		approved := 0
		approvalBinding := binding.BindInt(&approved)

		adminPIN := entity.OptionGet(entity.KeyAdminPIN)
		if adminPIN == "" {
			adminPIN = "1234"
		}

		passSuccessFunc := func(val string) {
			if val == adminPIN {
				approvalBinding.Set(authSuccess)
			}
		}

		passCancelFunc := func() {
			authPopup.Hide()
			approvalBinding.Set(authCancel)
		}

		keypad := keypad.Keypad(passSuccessFunc, passCancelFunc, true)
		authPopup = widget.NewModalPopUp(keypad, w.Canvas())
		go func() {
			i := 0
			for range time.Tick(time.Second) {

				i++
				if i >= maxAuthSeconds {
					fyne.Do(func() {
						authPopup.Hide()
					})
					return
				}

				isApproved, _ := approvalBinding.Get()
				if isApproved == authCancel {
					fyne.Do(func() {
						authPopup.Hide()
					})
					return
				} else if isApproved == authSuccess {
					fyne.Do(func() {
						authPopup.Hide()

						adminPopup = widget.NewModalPopUp(getAdminMenu(), w.Canvas())
						adminPopup.Show()
					})
					return
				}
			}
		}()

		authPopup.Show()
	})
	//adminButton.Resize(fyne.NewSize(75, 75))
	adminButton.Alignment = widget.ButtonAlignCenter
	adminButton.Importance = widget.LowImportance

	return adminButton
}

func getAdminMenu() *fyne.Container {
	adminSettingsHeader := widget.NewLabel("Admin Menu")

	adminCloseButton := widget.NewButtonWithIcon("", theme.WindowCloseIcon(), func() {
		adminPopup.Hide() // Function to hide the pop-up
	})
	adminCloseButton.Alignment = widget.ButtonAlignTrailing

	adminPopupContent := container.New(
		layout.NewCenterLayout(),
		container.NewVBox(
			container.NewGridWithColumns(3,
				adminSettingsHeader,
				layout.NewSpacer(),
				adminCloseButton,
			),
			container.NewAppTabs(
				container.NewTabItem("Stats", adminStatsTab()),
				container.NewTabItem("SSH", adminPotTab()),
				container.NewTabItem("App", adminSystemTab()),
			),
		),
	)
	adminPopupContent.Resize(fyne.NewSize(900, 400))

	return adminPopupContent
}

func adminPotTab() *fyne.Container {
	return container.NewVBox(
		container.NewGridWithRows(2,
			container.NewGridWithColumns(2,
				widget.NewButtonWithIcon("Set Max Users", theme.AccountIcon(), func() {
					var sp *widget.PopUp

					keypad := keypad.Keypad(
						func(val string) {
							log.Debug(entity.KeyPotMaxUsers, "val", val)
							entity.OptionSet(entity.KeyPotMaxUsers, val)
							sp.Hide()
						},
						func() {
							sp.Hide()
						},
						false,
					)

					sp = widget.NewModalPopUp(keypad, w.Canvas())
					sp.Show()
				}),
			),
			layout.NewSpacer(),
		),
	)
}

func adminSystemTab() *fyne.Container {
	return container.NewVBox(
		container.NewGridWithRows(2,
			container.NewGridWithColumns(2,
				widget.NewButtonWithIcon("Quit App", theme.LogoutIcon(), func() {
					os.Exit(0)
				}),
				widget.NewButtonWithIcon("Change PIN", theme.SettingsIcon(), func() {
					var sp *widget.PopUp

					keypad := keypad.Keypad(
						func(val string) {
							log.Debug(entity.KeyAdminPIN, "val", val)
							entity.OptionSet(entity.KeyAdminPIN, val)
							sp.Hide()
						},
						func() {
							sp.Hide()
						},
						false,
					)

					sp = widget.NewModalPopUp(keypad, w.Canvas())
					sp.Show()
				}),
			),
			widget.NewButtonWithIcon("Toggle Fullscreen", theme.ViewFullScreenIcon(), func() {
				w.SetFullScreen(!w.FullScreen())
			}),
		),
	)
}

func adminStatsTab() *fyne.Container {
	userCounts, err := entity.EventCountQuery(
		`SELECT
			"1D" AS duration,
			COUNT(*) AS total
		FROM events
		WHERE
			events.type = "login"
			AND events.timestamp >= datetime('now','-1 days')
		UNION
		SELECT
			"7D" AS duration,
			COUNT(*) AS total
		FROM events
		WHERE
			events.type = "login"
			AND events.timestamp >= datetime('now','-7 days')`,
	)
	if err != nil {
		log.Error("Error querying user counts", err)
	}

	userCountsLabels := []fyne.CanvasObject{
		widget.NewLabelWithStyle("Users:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
	}

	for _, e := range userCounts {
		userCountsLabels = append(userCountsLabels, widget.NewLabelWithStyle(fmt.Sprintf("%d (%s)", e.Count, e.Value), fyne.TextAlignCenter, fyne.TextStyle{Monospace: true}))
	}

	return container.NewVBox(
		container.NewGridWithRows(3,
			container.NewGridWithColumns(3,
				userCountsLabels...,
			),
			container.NewGridWithColumns(2,
				widget.NewButtonWithIcon("Leaderboard", theme.ContentPasteIcon(), func() {}),
				widget.NewButtonWithIcon("Recent", theme.HistoryIcon(), func() {
					var sp *widget.PopUp

					topCommands, err := entity.EventQuery(
						`SELECT *
						 FROM events
						 WHERE app = "ssh"
						 ORDER by timestamp DESC
						 LIMIT 100`,
						"typed",
					)
					if err != nil {
						log.Error("Error querying top commands", err)
						return
					}

					data := []string{}
					tz, _ := time.LoadLocation("America/Los_Angeles")

					for _, e := range topCommands {
						data = append(data, fmt.Sprintf("%s (%s) > %s", e.User, e.Timestamp.In(tz).Format(time.Kitchen), e.Action))
					}

					sp = adminListModal("Recent Events", data, func() {
						sp.Hide()
					})
					sp.Resize(fyne.NewSize(700, 400))
					sp.Show()
				}),
			),
			container.NewGridWithColumns(2,
				widget.NewButtonWithIcon("Top Commands", theme.ListIcon(), func() {
					var sp *widget.PopUp

					topCommands, err := entity.EventCountQuery(
						`SELECT
							action,
							count(*) AS total
						FROM events
						WHERE
							events.type = ?
						GROUP BY events.action
						ORDER by count(*) DESC
						LIMIT 25`,
						"typed",
					)
					if err != nil {
						log.Error("Error querying top commands", err)
						return
					}

					data := []string{}
					for _, e := range topCommands {
						data = append(data, e.Value)
					}

					sp = adminListModal("Top Commands", data, func() {
						sp.Hide()
					})
					sp.Resize(fyne.NewSize(700, 400))
					sp.Show()
				}),
				widget.NewButtonWithIcon("Top Users", theme.ListIcon(), func() {
					var sp *widget.PopUp

					topCommands, err := entity.EventCountQuery(
						`SELECT
							events.user,
							count(*) AS total
						FROM events
						WHERE
							events.type = "login"
						GROUP BY events.user
						ORDER by count(*) DESC
						LIMIT 25`,
						"typed",
					)
					if err != nil {
						log.Error("Error querying top users", err)
						return
					}

					data := []string{}
					for _, e := range topCommands {
						data = append(data, fmt.Sprintf("%s (%d)", e.Value, e.Count))
					}

					sp = adminListModal("Top Users", data, func() {
						sp.Hide()
					})
					sp.Resize(fyne.NewSize(700, 400))
					sp.Show()
				}),
			),
		),
	)
}

func adminListModal(title string, rows []string, closeFn func()) *widget.PopUp {
	// Add a blank line to the start of the list to get the #1 item to show below the modal header.
	rows = append([]string{""}, rows...)

	// Create new binding list
	listBinding := binding.BindStringList(&rows)

	data := widget.NewListWithData(
		listBinding,
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(i binding.DataItem, o fyne.CanvasObject) {
			o.(*widget.Label).Bind(i.(binding.String))
			o.(*widget.Label).TextStyle = fyne.TextStyle{Monospace: true}
			o.(*widget.Label).SizeName = theme.SizeNameCaptionText
		},
	)

	data.Resize(fyne.NewSize(700, 300))

	header := container.NewVBox(
		container.NewStack(
			canvas.NewRectangle(theme.Color(theme.ColorNameOverlayBackground)),
			container.NewHBox(
				container.NewPadded(widget.NewLabel(title)),
				layout.NewSpacer(),
				container.NewPadded((widget.NewButtonWithIcon("", theme.WindowCloseIcon(), closeFn))),
			),
		),
		layout.NewSpacer(),
	)

	return widget.NewModalPopUp(
		container.NewStack(
			data,
			header,
		),
		w.Canvas())
}
