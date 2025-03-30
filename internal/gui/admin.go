package gui

import (
	"os"
	"time"

	fyne "fyne.io/fyne/v2"
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
					authPopup.Hide()
					return
				}

				isApproved, _ := approvalBinding.Get()
				if isApproved == authCancel {
					authPopup.Hide()
					return
				} else if isApproved == authSuccess {
					authPopup.Hide()

					adminPopup = widget.NewModalPopUp(getAdminMenu(), w.Canvas())
					adminPopup.Show()
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
				container.NewTabItem("Bear", adminBearTab()),
				container.NewTabItem("Pot", adminPotTab()),
				container.NewTabItem("App", adminSystemTab()),
			),
		),
	)
	adminPopupContent.Resize(fyne.NewSize(900, 400))

	return adminPopupContent
}

func adminBearTab() *fyne.Container {
	return container.NewVBox(
		container.NewGridWithRows(2,
			widget.NewButtonWithIcon("Toggle Fullscreen", theme.ViewFullScreenIcon(), func() {
				w.SetFullScreen(!w.FullScreen())
			}),
			layout.NewSpacer(),
			/*
				container.NewGridWithColumns(2,
					widget.NewButtonWithIcon("Nothing", theme.WarningIcon(), func() {}),
					widget.NewButtonWithIcon("Nothing", theme.WarningIcon(), func() {}),
				),
			*/
		),
	)
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
				widget.NewButtonWithIcon("Purge Logs/Stats", theme.DeleteIcon(), func() {
					// Do nothing.
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
			/*
				container.NewGridWithColumns(2,
					widget.NewButtonWithIcon("Nothing", theme.WarningIcon(), func() {}),
					widget.NewButtonWithIcon("Nothing", theme.WarningIcon(), func() {}),
				),
			*/
			layout.NewSpacer(),
		),
	)
}

func adminStatsTab() *fyne.Container {
	return container.NewVBox(
		container.NewGridWithRows(2,
			container.NewGridWithColumns(2,
				widget.NewButtonWithIcon("Leaderboard", theme.ContentPasteIcon(), func() {}),
				widget.NewButtonWithIcon("Top Commands", theme.ComputerIcon(), func() {}),
			),
			container.NewGridWithColumns(2,
				widget.NewButtonWithIcon("Activity Graph", theme.BrokenImageIcon(), func() {}),
				layout.NewSpacer(),
			),
		),
	)
}
