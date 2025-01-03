package gui

import (
	fyne "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

var adminPopup *widget.PopUp

func getAdminButton() *widget.Button {
	adminButton := widget.NewButtonWithIcon("", theme.MenuIcon(), func() {
		if adminPopup == nil {
			adminPopup = widget.NewModalPopUp(getAdminMenu(), w.Canvas())
		}
		adminPopup.Show()
	})
	//adminButton.Resize(fyne.NewSize(75, 75))
	adminButton.Alignment = widget.ButtonAlignCenter
	adminButton.Importance = widget.LowImportance

	return adminButton
}

func getAdminMenu() *fyne.Container {
	adminSettingsHeader := widget.NewLabel("HBHP - Settings")
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
				container.NewTabItem("Honey Pot", widget.NewLabel("Honey Pot")),
				container.NewTabItem("Honey Bear", widget.NewLabel("Honey Bear")),
				container.NewTabItem("System", adminSystemMenu()),
				container.NewTabItem("About", widget.NewLabel("About")),
			),
		),
	)
	adminPopupContent.Resize(fyne.NewSize(900, 400))

	return adminPopupContent
}

func adminSystemMenu() *fyne.Container {
	pane := container.NewVBox(
		widget.NewLabel("System Menu"),
	)

	pane.Resize(fyne.NewSize(800, 600))

	return pane
}
