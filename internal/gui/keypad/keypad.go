package keypad

import (
	fyne "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	maxDigits = 9
)

var (
	selectedLabel *canvas.Text
	typed         string
)

func Keypad(successFunc func(val string), cancelFunc func(), hideTyped bool) *fyne.Container {
	defaultLabel := ""

	selectedLabel = canvas.NewText(defaultLabel, theme.Color(theme.ColorNameForeground))
	selectedLabel.Alignment = fyne.TextAlignCenter
	selectedLabel.TextStyle = fyne.TextStyle{Monospace: true}
	selectedLabel.TextSize = 48
	selectedLabel.Resize(fyne.NewSize(300, 50))

	cancelBtn := widget.NewButtonWithIcon("", theme.WindowCloseIcon(), cancelFunc)
	submitBtn := widget.NewButtonWithIcon("Submit", theme.ConfirmIcon(), func() {
		successFunc(typed)
		clearTyped()
	})

	submitBtn.Importance = widget.HighImportance

	return container.NewVBox(
		selectedLabel,
		container.NewGridWithRows(4,
			container.NewGridWithColumns(3,
				widget.NewButton("1", func() {
					addDigit("1", hideTyped)
				}),
				widget.NewButton("2", func() {
					addDigit("2", hideTyped)
				}),
				widget.NewButton("3", func() {
					addDigit("3", hideTyped)
				}),
			),
			container.NewGridWithColumns(3,
				widget.NewButton("4", func() {
					addDigit("4", hideTyped)
				}),
				widget.NewButton("5", func() {
					addDigit("5", hideTyped)
				}),
				widget.NewButton("6", func() {
					addDigit("6", hideTyped)
				}),
			),
			container.NewGridWithColumns(3,
				widget.NewButton("7", func() {
					addDigit("7", hideTyped)
				}),
				widget.NewButton("8", func() {
					addDigit("8", hideTyped)
				}),
				widget.NewButton("9", func() {
					addDigit("9", hideTyped)
				}),
			),
			container.NewGridWithColumns(3,
				layout.NewSpacer(),
				widget.NewButton("0", func() {
					addDigit("0", hideTyped)
				}),
				layout.NewSpacer(),
			),
		),
		container.NewHBox(
			cancelBtn,
			submitBtn,
		),
	)
}

func addDigit(digit string, hideTyped bool) {
	if len(selectedLabel.Text) >= maxDigits {
		return
	}

	if hideTyped {
		selectedLabel.Text += "*"
	} else {
		selectedLabel.Text += digit
	}

	typed += digit
	selectedLabel.Refresh()
}

func clearTyped() {
	typed = ""
	selectedLabel.Text = typed
	selectedLabel.Refresh()
}
