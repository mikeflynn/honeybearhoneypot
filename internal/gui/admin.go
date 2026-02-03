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
	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot"
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
			// If no PIN, just open the admin menu directly
			adminPopup = widget.NewModalPopUp(getAdminMenu(), w.Canvas())
			adminPopup.Show()
			return
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

		keypad := keypad.Keypad(passSuccessFunc, passCancelFunc, true, "")
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
				container.NewTabItem("Data", adminDataTab()),
			),
		),
	)
	adminPopupContent.Resize(fyne.NewSize(900, 400))

	return adminPopupContent
}

func adminPotTab() *fyne.Container {
	return container.NewVBox(
		container.NewGridWithRows(3,
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
						entity.OptionGet(entity.KeyPotMaxUsers),
					)

					sp = widget.NewModalPopUp(keypad, w.Canvas())
					sp.Show()
				}),
				widget.NewButtonWithIcon("Rate - Max Conn", theme.WarningIcon(), func() {
					var sp *widget.PopUp

					keypad := keypad.Keypad(
						func(val string) {
							log.Debug(entity.KeyRateLimitMax, "val", val)
							entity.OptionSet(entity.KeyRateLimitMax, val)
							honeypot.GetRateLimiter().Reload()
							sp.Hide()
						},
						func() {
							sp.Hide()
						},
						false,
						entity.OptionGet(entity.KeyRateLimitMax),
					)

					sp = widget.NewModalPopUp(keypad, w.Canvas())
					sp.Show()
				}),
			),
			container.NewGridWithColumns(2,
				widget.NewButtonWithIcon("Rate - Window (s)", theme.HistoryIcon(), func() {
					var sp *widget.PopUp

					keypad := keypad.Keypad(
						func(val string) {
							log.Debug(entity.KeyRateLimitWindow, "val", val)
							entity.OptionSet(entity.KeyRateLimitWindow, val)
							honeypot.GetRateLimiter().Reload()
							sp.Hide()
						},
						func() {
							sp.Hide()
						},
						false,
						entity.OptionGet(entity.KeyRateLimitWindow),
					)

					sp = widget.NewModalPopUp(keypad, w.Canvas())
					sp.Show()
				}),
				widget.NewButtonWithIcon("Rate - Ban Time (s)", theme.ErrorIcon(), func() {
					var sp *widget.PopUp

					keypad := keypad.Keypad(
						func(val string) {
							log.Debug(entity.KeyRateLimitBan, "val", val)
							entity.OptionSet(entity.KeyRateLimitBan, val)
							honeypot.GetRateLimiter().Reload()
							sp.Hide()
						},
						func() {
							sp.Hide()
						},
						false,
						entity.OptionGet(entity.KeyRateLimitBan),
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
						entity.OptionGet(entity.KeyAdminPIN),
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
