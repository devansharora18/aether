package tui

import (
	"os"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func Run() error {
	app := tview.NewApplication()
	pages := tview.NewPages()

	header := tview.NewTextView().
		SetDynamicColors(true).
		SetText("[cyan::b]aether[-:-:-]  [gray]TUI mode • Enter to select • Esc to go back[-]")
	header.SetBorder(true)

	menu := tview.NewList().ShowSecondaryText(true)
	menu.SetBorder(true).SetTitle(" Actions ")
	menu.AddItem("Install package", "Equivalent to -S", 'i', func() { showPackageAction(app, pages, "Install package", "Package(s)", installRunner) })
	menu.AddItem("Remove package", "Equivalent to -R", 'r', func() { showPackageAction(app, pages, "Remove package", "Package(s)", removeRunner) })
	menu.AddItem("Purge package", "Remove + config files (-Rn)", 'p', func() { showPackageAction(app, pages, "Purge package", "Package(s)", purgeRunner) })
	menu.AddItem("Autoremove", "Remove unused dependencies (-Rc)", 'a', func() { runOperation(app, pages, "Autoremove", autoremoveRunner) })
	menu.AddItem("Sync package database", "Equivalent to -Sy", 's', func() { runOperation(app, pages, "Sync package database", updateRunner) })
	menu.AddItem("Sync + Upgrade", "Equivalent to -Syu", 'u', func() { runSyncUpgrade(app, pages) })
	menu.AddItem("Search packages", "Live results as you type", '/', func() { showSearch(app, pages) })
	menu.AddItem("Package info", "Show detailed package info (-Qi)", 'd', func() { showPackageInfo(app, pages) })
	menu.AddItem("List upgradable", "Show upgradable packages (-Qu)", 'l', func() { showListUpgradable(app, pages) })
	menu.AddItem("Clean cache", "Remove cached packages (-Sc)", 'c', func() { runOperation(app, pages, "Clean cache", cleanRunner) })
	menu.AddItem("Manage sources", "View/add/edit/delete APT sources", 'm', func() { showSources(app, pages) })
	menu.AddItem("Quit", "Exit TUI", 'q', func() { app.Stop() })

	if os.Geteuid() != 0 {
		menu.SetTitle(" Actions (read-only without root) ")
	}

	layout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(header, 3, 0, false).
		AddItem(menu, 0, 1, true)

	pages.AddPage("menu", layout, true, true)

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlC {
			app.Stop()
			return nil
		}
		return event
	})

	return app.SetRoot(pages, true).EnableMouse(true).Run()
}

func ensureRoot(app *tview.Application, pages *tview.Pages) bool {
	if os.Geteuid() == 0 {
		return true
	}
	showInfoModal(app, pages, "Root required", "Run this mode with sudo for install/remove/update actions.")
	return false
}

func showInfoModal(app *tview.Application, pages *tview.Pages, title, message string) {
	modal := tview.NewModal().
		SetText(message).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(_ int, _ string) {
			pages.RemovePage("modal")
		})
	modal.SetTitle(title).SetBorder(true)
	pages.AddPage("modal", modal, true, true)
}
