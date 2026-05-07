package gui

import (
	"fmt"

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
	title := fmt.Sprintf("Broadcast Actions — %d connected", honeypot.SessionCount())

	knock := widget.NewButton("Knock Knock", func() { honeypot.ActionKnock() })
	notice := widget.NewButton("System Notice", func() { honeypot.ActionSystemNotice() })
	join := widget.NewButton("Fake Join", func() { honeypot.ActionFakeJoin() })
	mtx := widget.NewButton("Matrix Storm", func() { honeypot.ActionMatrix() })
	conf := widget.NewButton("Confetti", func() { honeypot.ActionConfetti() })

	kick := widget.NewButton("Kick All", func() {
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

	content := container.NewVBox(knock, notice, join, mtx, conf, kick)
	dialog.NewCustom(title, "Close", content, w).Show()
}
