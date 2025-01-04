package gui

import (
	"time"

	fyne "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
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

		authPopup = widget.NewModalPopUp(adminAuthenticate(approvalBinding), w.Canvas())
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

func adminAuthenticate(approved binding.ExternalInt) *fyne.Container {
	selectedLabel := canvas.NewText("", theme.Color(theme.ColorNameForeground))
	selectedLabel.Alignment = fyne.TextAlignCenter
	selectedLabel.TextStyle = fyne.TextStyle{Monospace: true}
	selectedLabel.TextSize = 48
	selectedLabel.Resize(fyne.NewSize(300, 50))

	maxDigits := 9
	addDigit := func(digit string) {
		if len(selectedLabel.Text) >= maxDigits {
			return
		}

		selectedLabel.Text += digit
		selectedLabel.Refresh()
	}

	keypad := container.NewVBox(
		selectedLabel,
		container.NewGridWithRows(3,
			container.NewGridWithColumns(3,
				widget.NewButton("1", func() {
					addDigit("1")
				}),
				widget.NewButton("2", func() {
					addDigit("2")
				}),
				widget.NewButton("3", func() {
					addDigit("3")
				}),
			),
			container.NewGridWithColumns(3,
				widget.NewButton("4", func() {
					addDigit("4")
				}),
				widget.NewButton("5", func() {
					addDigit("5")
				}),
				widget.NewButton("6", func() {
					addDigit("6")
				}),
			),
			container.NewGridWithColumns(3,
				widget.NewButton("7", func() {
					addDigit("7")
				}),
				widget.NewButton("8", func() {
					addDigit("8")
				}),
				widget.NewButton("9", func() {
					addDigit("9")
				}),
			),
		),
		container.NewHBox(
			widget.NewButtonWithIcon("", theme.WindowCloseIcon(), func() {
				authPopup.Hide()
				approved.Set(authCancel)
			}),
			container.NewStack(
				//canvas.NewRectangle(theme.Color(theme.ColorNameSuccess)),
				widget.NewButtonWithIcon("Submit", theme.ConfirmIcon(), func() {
					if selectedLabel.Text == "1234" {
						approved.Set(authSuccess)
					} else {
						selectedLabel.Text = ""
					}

					selectedLabel.Refresh()
				}),
			),
		),
	)

	return keypad
}

func adminBearTab() *fyne.Container {
	return container.NewVBox(
		widget.NewLabel("Bear Actions"),
	)
}

func adminPotTab() *fyne.Container {
	return container.NewVBox(
		widget.NewLabel("Pot Actions"),
	)
}

func adminSystemTab() *fyne.Container {
	return container.NewVBox(
		widget.NewLabel("System Actions"),
	)
}
