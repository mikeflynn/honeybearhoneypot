package keypad

import (
	fyne "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

var (
	maxDigits = 9
)

func Keypad(successFunc func(val string), cancelFunc func(), hideTyped bool, initialValue string) *fyne.Container {
	typed := initialValue
	defaultLabel := initialValue
	if hideTyped {
		defaultLabel = ""
		for range initialValue {
			defaultLabel += "*"
		}
	}

	selectedLabel := canvas.NewText(defaultLabel, theme.Color(theme.ColorNameForeground))
	selectedLabel.Alignment = fyne.TextAlignCenter
	selectedLabel.TextStyle = fyne.TextStyle{Monospace: true}
	selectedLabel.TextSize = 48
	selectedLabel.Resize(fyne.NewSize(300, 50))

	refreshLabel := func() {
		if hideTyped {
			masked := ""
			for i := 0; i < len(typed); i++ {
				masked += "*"
			}
			selectedLabel.Text = masked
		} else {
			selectedLabel.Text = typed
		}
		selectedLabel.Refresh()
	}

	addDigit := func(digit string) {
		if len(typed) >= maxDigits {
			return
		}
		typed += digit
		refreshLabel()
	}

	clearTyped := func() {
		typed = ""
		refreshLabel()
	}

	backspace := func() {
		if len(typed) > 0 {
			typed = typed[:len(typed)-1]
			refreshLabel()
		}
	}

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
			container.NewGridWithColumns(3,
				layout.NewSpacer(),
				widget.NewButton("0", func() {
					addDigit("0")
				}),
				widget.NewButtonWithIcon("", theme.ContentUndoIcon(), func() {
					backspace()
				}),
			),
		),
		container.NewHBox(
			cancelBtn,
			submitBtn,
		),
	)
}
