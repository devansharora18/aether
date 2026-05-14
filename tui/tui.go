package tui

import (
	"os"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func Run() error {
	setupTheme()

	app := tview.NewApplication()
	pages := tview.NewPages()

	menu := styleList(tview.NewList().ShowSecondaryText(true))
	menu.SetBorder(true).SetTitle(" Actions ")
	stylePanel(menu.Box)

	menu.AddItem("Install package", "Equivalent to -S", 'i', func() { showPackageAction(app, pages, "Install package", "Package(s)", installStream) })
	menu.AddItem("Remove package", "Equivalent to -R", 'r', func() { showPackageAction(app, pages, "Remove package", "Package(s)", removeStream) })
	menu.AddItem("Purge package", "Remove + config files (-Rn)", 'p', func() { showPackageAction(app, pages, "Purge package", "Package(s)", purgeStream) })
	menu.AddItem("Autoremove", "Remove unused dependencies (-Rc)", 'a', func() {
		runStreamOperation(app, pages, "Autoremove", "", autoremoveStream())
	})
	menu.AddItem("Sync package database", "Equivalent to -Sy", 's', func() {
		runStreamOperation(app, pages, "Sync package database", "", updateStream())
	})
	menu.AddItem("Sync + Upgrade", "Equivalent to -Syu", 'u', func() { runSyncUpgrade(app, pages) })
	menu.AddItem("Search packages", "Live results as you type", '/', func() { showSearch(app, pages) })
	menu.AddItem("Package info", "Show detailed package info (-Qi)", 'd', func() { showPackageInfo(app, pages) })
	menu.AddItem("List upgradable", "Show upgradable packages (-Qu)", 'l', func() { showListUpgradable(app, pages) })
	menu.AddItem("Clean cache", "Remove cached packages (-Sc)", 'c', func() {
		runStreamOperation(app, pages, "Clean cache", "", cleanStream())
	})
	menu.AddItem("Manage sources", "View/add/edit/delete APT sources", 'm', func() { showSources(app, pages) })
	menu.AddItem("Background Operations", "View running downloads/updates", 'b', func() {
		showActiveOperations(app, pages)
	})
	menu.AddItem("Quit", "Exit aether", 'q', func() { app.Stop() })

	if os.Geteuid() != 0 {
		menu.SetTitle(" Actions  ·  read-only without root ")
	}

	hints := []keyHint{
		{"↑↓", "navigate"},
		{"↵", "select"},
		{"b", "background"},
		{"q", "quit"},
	}

	pages.AddPage("menu", chrome(menu, "Main Menu", hints), true, true)

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlC {
			app.Stop()
			return nil
		}
		return event
	})

	return app.SetRoot(pages, true).EnableMouse(true).Run()
}

func showInfoModal(app *tview.Application, pages *tview.Pages, title, message string) {
	modal := tview.NewModal().
		SetText(message).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(_ int, _ string) {
			pages.RemovePage("modal")
		})
	styleModal(modal)
	modal.SetTitle(" " + title + " ").SetBorder(true)
	pages.AddPage("modal", modal, true, true)
}
