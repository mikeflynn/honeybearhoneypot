package gui

import (
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/mikeflynn/honeybearhoneypot/internal/honeypot"
)

func broadcastButton() *widget.Button {
	btn := widget.NewButtonWithIcon("", theme.MailSendIcon(), func() {
		showBroadcastModal()
	})
	btn.Importance = widget.LowImportance
	return btn
}

func showBroadcastModal() {
	knock := widget.NewButtonWithIcon("Knock", theme.QuestionIcon(), honeypot.ActionKnock)
	notice := widget.NewButtonWithIcon("Notice", theme.WarningIcon(), honeypot.ActionSystemNotice)
	join := widget.NewButtonWithIcon("Fake Join", theme.AccountIcon(), honeypot.ActionFakeJoin)
	mtx := widget.NewButtonWithIcon("Matrix", theme.VisibilityIcon(), honeypot.ActionMatrix)
	conf := widget.NewButtonWithIcon("Confetti", theme.ColorPaletteIcon(), honeypot.ActionConfetti)
	shuffle := widget.NewButtonWithIcon("Shuffle CWD", theme.FolderIcon(), honeypot.ActionShuffleDirs)
	gltch := widget.NewButtonWithIcon("Glitch", theme.BrokenImageIcon(), honeypot.ActionGlitch)

	kick := widget.NewButtonWithIcon("Kick All", theme.LogoutIcon(), func() {
		dialog.NewConfirm(
			"Kick All Sessions",
			"This will disconnect every currently logged-in user. Continue?",
			func(ok bool) {
				if ok {
					honeypot.ActionKickAll()
				}
			},
			w,
		).Show()
	})
	kick.Importance = widget.DangerImportance

	grid := container.NewGridWithColumns(3, knock, notice, join, mtx, conf, shuffle, gltch, kick)
	dialog.NewCustom("Broadcast Actions", "Close", grid, w).Show()
}
