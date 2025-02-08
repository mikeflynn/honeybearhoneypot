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
	"github.com/mikeflynn/hardhat-honeybear/internal/gui/keypad"
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

		passSuccessFunc := func() {
			approvalBinding.Set(authSuccess)
		}

		passCancelFunc := func() {
			authPopup.Hide()
			approvalBinding.Set(authCancel)
		}

		password := "1234"
		keypad := keypad.Keypad(passSuccessFunc, passCancelFunc, nil, &password)
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
	adminSettingsHeader := widget.NewLabel("Settings")
	adminPopupContent := container.New(
		layout.NewCenterLayout(),
		container.NewVBox(
			container.NewGridWithColumns(4,
				adminSettingsHeader,
				layout.NewSpacer(),
				layout.NewSpacer(),
				widget.NewButtonWithIcon("", theme.WindowCloseIcon(), func() {
					adminPopup.Hide() // Function to hide the pop-up
				}),
			),
			container.NewAppTabs(
				container.NewTabItem("Bear", adminBearTab()),
				container.NewTabItem("Pot", adminPotTab()),
				container.NewTabItem("System", adminSystemTab()),
			),
		),
	)
	adminPopupContent.Resize(fyne.NewSize(900, 400))

	return adminPopupContent
}

func adminBearTab() *fyne.Container {
	return container.NewVBox(
		container.NewGridWithRows(2,
			container.NewGridWithColumns(2,
				widget.NewButtonWithIcon("Nothing", theme.WarningIcon(), func() {}),
				widget.NewButtonWithIcon("Nothing", theme.WarningIcon(), func() {}),
			),
			widget.NewButtonWithIcon("Toggle Fullscreen", theme.ViewFullScreenIcon(), func() {
				w.SetFullScreen(!w.FullScreen())
			}),
		),
	)
}

func adminPotTab() *fyne.Container {
	return container.NewVBox(
		container.NewGridWithRows(2,
			container.NewGridWithColumns(2,
				widget.NewButtonWithIcon("Nothing", theme.WarningIcon(), func() {}),
				widget.NewButtonWithIcon("Nothing", theme.WarningIcon(), func() {}),
			),
			container.NewGridWithColumns(2,
				widget.NewButtonWithIcon("Nothing", theme.WarningIcon(), func() {}),
				widget.NewButtonWithIcon("Nothing", theme.WarningIcon(), func() {}),
			),
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
				widget.NewButtonWithIcon("Nothing", theme.WarningIcon(), func() {}),
			),
			container.NewGridWithColumns(2,
				widget.NewButtonWithIcon("Nothing", theme.WarningIcon(), func() {}),
				widget.NewButtonWithIcon("Nothing", theme.WarningIcon(), func() {}),
			),
		),
	)
}
