package keypad

import (
	fyne "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	maxDigits = 9
)

var (
	selectedLabel *canvas.Text
)

func Keypad(successFunc func(), cancelFunc func(), defaultVal *string, password *string) *fyne.Container {
	defaultLabel := ""
	if defaultVal != nil {
		defaultLabel = *defaultVal
	}

	selectedLabel = canvas.NewText(defaultLabel, theme.Color(theme.ColorNameForeground))
	selectedLabel.Alignment = fyne.TextAlignCenter
	selectedLabel.TextStyle = fyne.TextStyle{Monospace: true}
	selectedLabel.TextSize = 48
	selectedLabel.Resize(fyne.NewSize(300, 50))

	cancelBtn := widget.NewButtonWithIcon("", theme.WindowCloseIcon(), cancelFunc)
	submitBtn := widget.NewButtonWithIcon("Submit", theme.ConfirmIcon(), func() {
		if password != nil {
			if selectedLabel.Text == *password {
				successFunc()
			}
		} else {
			successFunc()
		}

		selectedLabel.Text = ""
		selectedLabel.Refresh()
	})

	submitBtn.Importance = widget.HighImportance

	return container.NewVBox(
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
			cancelBtn,
			submitBtn,
		),
	)
}

func addDigit(digit string) {
	if len(selectedLabel.Text) >= maxDigits {
		return
	}

	selectedLabel.Text += digit
	selectedLabel.Refresh()
}
