package gui

import (
	"fmt"

	fyne "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/mikeflynn/honeybearhoneypot/internal/db/export"
)

func adminDataTab() *fyne.Container {
	return container.NewVBox(
		layout.NewSpacer(),
		container.NewCenter(
			widget.NewButtonWithIcon("Export", theme.DocumentSaveIcon(), func() {
				showExportModal()
			}),
		),
		layout.NewSpacer(),
	)
}

func showExportModal() {
	var popup *widget.PopUp

	// 1. Data Types Checkboxes
	checkEvents := widget.NewCheck("Events", nil)
	checkEvents.Checked = true
	checkOptions := widget.NewCheck("Options", nil)
	checkOptions.Checked = true
	checkCTF := widget.NewCheck("CTF", nil)
	checkCTF.Checked = true

	// 2. Format Select
	formatSelect := widget.NewSelect([]string{"JSON", "CSV", "Raw"}, nil)
	formatSelect.Selected = "JSON"

	// 3. Output Path
	pathEntry := widget.NewEntry()
	pathEntry.Disable()
	pathEntry.SetPlaceHolder("Select output directory...")

	pathBtn := widget.NewButtonWithIcon("", theme.FolderOpenIcon(), func() {
		dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
			if err == nil && uri != nil {
				pathEntry.SetText(uri.Path())
			}
		}, w).Show()
	})

	// 4. Action Buttons
	exportBtn := widget.NewButtonWithIcon("Export", theme.ConfirmIcon(), func() {
		// Collect types
		var types []export.DataType
		if checkEvents.Checked {
			types = append(types, export.DataTypeEvents)
		}
		if checkOptions.Checked {
			types = append(types, export.DataTypeOptions)
		}
		if checkCTF.Checked {
			types = append(types, export.DataTypeCTFs)
		}

		// Collect format
		var format export.ExportFormat
		switch formatSelect.Selected {
		case "JSON":
			format = export.JSON
		case "CSV":
			format = export.CSV
		case "Raw":
			format = export.RAW
		}

		// Collect path
		path := pathEntry.Text
		if path == "" {
			dialog.ShowError(fmt.Errorf("please select an output path"), w)
			return
		}

		// Run Export
		err := export.ExportDatabase(types, format, path)
		if err != nil {
			dialog.ShowError(err, w)
		} else {
			dialog.ShowInformation("Export Success", "Data exported successfully to "+path, w)
			popup.Hide()
		}
	})
	exportBtn.Importance = widget.HighImportance

	cancelBtn := widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), func() {
		popup.Hide()
	})

	form := container.NewVScroll(
		container.NewVBox(
			widget.NewLabel("Select Data Types:"),
			container.NewHBox(checkEvents, checkOptions, checkCTF),
			widget.NewLabel("Select Format:"),
			formatSelect,
			widget.NewLabel("Output Directory:"),
			container.NewBorder(nil, nil, nil, pathBtn, pathEntry),
			layout.NewSpacer(),
		),
	)
	form.SetMinSize(fyne.NewSize(480, 300))

	// Content Layout
	content := container.NewVBox(
		form,
		widget.NewSeparator(),
		container.NewGridWithColumns(2, cancelBtn, exportBtn),
	)

	// Wrap in a sized container to ensure it looks good as a modal
	modalContent := container.NewPadded(content)

	popup = widget.NewModalPopUp(modalContent, w.Canvas())
	popup.Resize(fyne.NewSize(500, 400))
	popup.Show()
}
