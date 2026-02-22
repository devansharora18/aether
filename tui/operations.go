package tui

import (
	"strings"
	"github.com/devansharora18/aether/libapt"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// runner adapters used by the top-level menu to keep Run() simple
func installRunner(pkgs []string) (*libapt.Result, error) { return libapt.Install(pkgs, false) }
func removeRunner(pkgs []string) (*libapt.Result, error)  { return libapt.Remove(pkgs, false) }
func updateRunner() (*libapt.Result, error)                { return libapt.Update(false) }
func upgradeRunner() (*libapt.Result, error)               { return libapt.Upgrade(false) }

func showPackageAction(app *tview.Application, pages *tview.Pages, title, label string, runner func([]string) (*libapt.Result, error)) {
	if !ensureRoot(app, pages) {
		return
	}

	input := tview.NewInputField().SetLabel(label + ": ")
	form := tview.NewForm()
	form.AddFormItem(input)
	form.AddButton("Run", func() {
		text := strings.TrimSpace(input.GetText())
		if text == "" {
			showInfoModal(app, pages, "Missing input", "Please enter at least one package.")
			return
		}
		pkgs := strings.Fields(text)
		runOperation(app, pages, title, func() (*libapt.Result, error) { return runner(pkgs) })
	})
	form.AddButton("Back", func() {
		pages.SwitchToPage("menu")
		pages.RemovePage("form")
	})
	form.SetBorder(true).SetTitle(" " + title + " ")

	pages.AddAndSwitchToPage("form", centered(form, 70, 12), true)
}

func runSyncUpgrade(app *tview.Application, pages *tview.Pages) {
	if !ensureRoot(app, pages) {
		return
	}
	status := tview.NewTextView().SetDynamicColors(true)
	status.SetBorder(true).SetTitle(" Running Sync + Upgrade ")
	status.SetText("[cyan]fetching[-]")
	pages.AddAndSwitchToPage("run", centered(status, 80, 18), true)

	go func() {
		updateRes, updateErr := libapt.Update(false)
		if updateErr != nil {
			app.QueueUpdateDraw(func() {
				status.SetText("[red]Update failed[-]\n\n" + updateRes.Output)
				addBackHint(status)
			})
			return
		}
		app.QueueUpdateDraw(func() {
			status.SetText("[cyan]upgrading[-]")
		})

		upgradeRes, upgradeErr := libapt.Upgrade(false)
		app.QueueUpdateDraw(func() {
			if upgradeErr != nil {
				status.SetText("[red]Upgrade failed[-]\n\n" + upgradeRes.Output)
			} else {
				status.SetText("[green]System upgraded successfully.[-]")
			}
			addBackHint(status)
		})
	}()

	status.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			pages.SwitchToPage("menu")
			pages.RemovePage("run")
			return nil
		}
		return event
	})
}

func runOperation(app *tview.Application, pages *tview.Pages, title string, runner func() (*libapt.Result, error)) {
	status := tview.NewTextView().SetDynamicColors(true)
	status.SetBorder(true).SetTitle(" " + title + " ")
	status.SetText("[cyan]working[-]")
	pages.AddAndSwitchToPage("run", centered(status, 80, 18), true)

	go func() {
		res, err := runner()
		app.QueueUpdateDraw(func() {
			if err != nil {
				out := strings.TrimSpace(res.Output)
				if out == "" {
					out = err.Error()
				}
				status.SetText("[red]Operation failed[-]\n\n" + out)
			} else {
				status.SetText("[green]Done successfully.[-]")
			}
			addBackHint(status)
		})
	}()

	status.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			pages.SwitchToPage("menu")
			pages.RemovePage("run")
			return nil
		}
		return event
	})
}
