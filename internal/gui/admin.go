package gui

import (
	"fmt"

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
	fmt.Println(adminButton.MinSize().Width)

	return adminButton
}

func getAdminMenu() *fyne.Container {
	adminSettingsHeader := widget.NewLabel("SETTINGS")
	adminSettingsHeader.Resize(fyne.NewSize(600, 600))
	adminPopupContent := container.New(
		layout.NewCenterLayout(),
		container.NewVBox(
			container.NewHBox(
				adminSettingsHeader,

				widget.NewButtonWithIcon("", theme.WindowCloseIcon(), func() {
					adminPopup.Hide() // Function to hide the pop-up
				}),
			),
		),
	)
	adminPopupContent.Resize(fyne.NewSize(640, 400))

	return adminPopupContent
}
