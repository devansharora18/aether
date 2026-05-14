package tui

import (
	"bytes"
	"os"
	"os/exec"
	"strings"

	"github.com/rivo/tview"
)

func ensureSudo(app *tview.Application, pages *tview.Pages, onDone func(bool)) {
	if os.Geteuid() == 0 {
		onDone(true)
		return
	}

	go func() {
		if sudoCached() {
			app.QueueUpdateDraw(func() {
				onDone(true)
			})
			return
		}
		app.QueueUpdateDraw(func() {
			showSudoPrompt(app, pages, onDone)
		})
	}()
}

func sudoCached() bool {
	cmd := exec.Command("sudo", "-n", "-v")
	return cmd.Run() == nil
}

func showSudoPrompt(app *tview.Application, pages *tview.Pages, onDone func(bool)) {
	status := tview.NewTextView().SetDynamicColors(true)
	status.SetText("  Enter your sudo password to continue.")
	stylePanel(status.Box).SetBorder(true).SetTitle(" Sudo Required ")

	form := styleForm(tview.NewForm())
	password := tview.NewInputField().SetLabel("Password: ").SetMaskCharacter('*')
	form.AddFormItem(password)

	form.AddButton("Authenticate", func() {
		pwd := strings.TrimSpace(password.GetText())
		if pwd == "" {
			status.SetText("  Password is required.")
			return
		}
		status.SetText("  Authenticating…")
		go func() {
			err := sudoAuthenticate(pwd)
			app.QueueUpdateDraw(func() {
				if err == nil {
					pages.RemovePage("sudo-auth")
					onDone(true)
					return
				}
				status.SetText("  Authentication failed. Try again.")
				password.SetText("")
				app.SetFocus(password)
			})
		}()
	})

	form.AddButton("Cancel", func() {
		pages.RemovePage("sudo-auth")
		onDone(false)
	})

	form.SetBorder(true).SetTitle(" Authentication ")

	body := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(status, 3, 0, false).
		AddItem(form, 0, 1, true)

	pages.AddPage("sudo-auth", centered(body, 70, 11), true, true)
	app.SetFocus(password)
}

func sudoAuthenticate(password string) error {
	cmd := exec.Command("sudo", "-S", "-v")
	cmd.Stdin = bytes.NewBufferString(password + "\n")
	cmd.Stdout = &bytes.Buffer{}
	cmd.Stderr = &bytes.Buffer{}
	return cmd.Run()
}