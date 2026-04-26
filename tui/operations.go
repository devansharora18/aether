package tui

import (
	"fmt"
	"strings"

	"github.com/devansharora18/aether/libapt"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// runner adapters used by the top-level menu to keep Run() simple
func installRunner(pkgs []string) (*libapt.Result, error) { return libapt.Install(pkgs, false) }
func removeRunner(pkgs []string) (*libapt.Result, error)  { return libapt.Remove(pkgs, false) }
func purgeRunner(pkgs []string) (*libapt.Result, error)   { return libapt.Purge(pkgs, false) }
func updateRunner() (*libapt.Result, error)               { return libapt.Update(false) }
func upgradeRunner() (*libapt.Result, error)              { return libapt.Upgrade(false) }
func autoremoveRunner() (*libapt.Result, error)           { return libapt.AutoRemove(false) }
func cleanRunner() (*libapt.Result, error)                { return libapt.Clean() }

// showPackageInfo prompts for a package name and displays rich details.
func showPackageInfo(app *tview.Application, pages *tview.Pages) {
	input := tview.NewInputField().SetLabel("Package: ")
	form := styleForm(tview.NewForm())
	form.AddFormItem(input)
	form.AddButton("Show", func() {
		name := strings.TrimSpace(input.GetText())
		if name == "" {
			showInfoModal(app, pages, "Missing input", "Please enter a package name.")
			return
		}
		showPkgDetails(app, pages, name)
	})
	form.AddButton("Back", func() {
		pages.SwitchToPage("menu")
		pages.RemovePage("info-form")
	})
	form.SetBorder(true).SetTitle(" Package Info ")

	hints := []keyHint{commonBackHint, {"tab", "next field"}, {"↵", "submit"}}
	pages.AddAndSwitchToPage("info-form",
		chrome(centered(form, 70, 9), "Package Info", hints), true)
}

func showPkgDetails(app *tview.Application, pages *tview.Pages, name string) {
	status := tview.NewTextView().SetDynamicColors(true)
	status.SetBorder(true).SetTitle(" " + name + " ")
	stylePanel(status.Box)
	status.SetText(fmt.Sprintf("\n  [%s]Loading package details…[-]", cInfo))

	hints := []keyHint{commonBackHint}
	pages.AddAndSwitchToPage("pkg-detail",
		chrome(status, "Package Info › "+name, hints), true)

	go func() {
		pkg, err := libapt.Show(name)
		app.QueueUpdateDraw(func() {
			if err != nil {
				status.SetText(fmt.Sprintf("\n  [%s::b]✗ Could not find package[-:-:-] %s\n", cError, name))
				return
			}

			var b strings.Builder
			b.WriteString(fmt.Sprintf("\n  [%s::b]%s[-:-:-]\n", cTitle, pkg.Name))
			if pkg.Summary != "" {
				b.WriteString(fmt.Sprintf("  [%s]%s[-]\n", cSubtext, pkg.Summary))
			}
			b.WriteString("\n")

			row := func(k, v string) {
				if v == "" {
					return
				}
				b.WriteString(fmt.Sprintf("  [%s]%-16s[-] %s\n", cSubtext, k, v))
			}
			row("Version", pkg.Version)
			if pkg.InstalledVersion != "" {
				row("Installed", fmt.Sprintf("[%s]%s[-]", cSuccess, pkg.InstalledVersion))
			} else {
				row("Installed", fmt.Sprintf("[%s](not installed)[-]", cMuted))
			}
			row("Architecture", pkg.Architecture)
			row("Section", pkg.Section)
			row("Priority", pkg.Priority)
			row("Origin", pkg.Origin)
			row("Maintainer", pkg.Maintainer)
			row("Homepage", pkg.Homepage)
			if pkg.PackageSize > 0 {
				row("Download Size", libapt.FormatSize(pkg.PackageSize))
			}
			if pkg.InstalledSize > 0 {
				row("Installed Size", libapt.FormatSize(pkg.InstalledSize))
			}

			if len(pkg.Depends) > 0 {
				deps := make([]string, len(pkg.Depends))
				for i, d := range pkg.Depends {
					if d.Relation != "" {
						deps[i] = fmt.Sprintf("%s (%s %s)", d.Name, d.Relation, d.Version)
					} else {
						deps[i] = d.Name
					}
				}
				b.WriteString(fmt.Sprintf("\n  [%s]Dependencies[-]\n", cSubtext))
				b.WriteString(fmt.Sprintf("  %s\n", strings.Join(deps, ", ")))
			}

			if pkg.IsUpgradable {
				b.WriteString(fmt.Sprintf("\n  [%s::b]⚠ Upgrade available[-:-:-]  %s [%s]→[-] %s\n",
					cWarning, pkg.InstalledVersion, cMuted, pkg.Version))
			}

			if pkg.Description != "" && pkg.Description != pkg.Summary {
				b.WriteString(fmt.Sprintf("\n  [%s]Description[-]\n", cSubtext))
				for _, line := range strings.Split(pkg.Description, "\n") {
					b.WriteString("  " + line + "\n")
				}
			}

			status.SetText(b.String())
		})
	}()

	status.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			pages.SwitchToPage("menu")
			pages.RemovePage("pkg-detail")
			pages.RemovePage("info-form")
			return nil
		}
		return event
	})
}

// showListUpgradable displays all upgradable packages.
func showListUpgradable(app *tview.Application, pages *tview.Pages) {
	status := tview.NewTextView().SetDynamicColors(true)
	status.SetBorder(true).SetTitle(" Upgradable Packages ")
	stylePanel(status.Box)
	status.SetText(fmt.Sprintf("\n  [%s]Checking for upgradable packages…[-]", cInfo))

	hints := []keyHint{commonBackHint}
	pages.AddAndSwitchToPage("upgradable",
		chrome(status, "List Upgradable", hints), true)

	go func() {
		pkgs, err := libapt.ListUpgradable()
		app.QueueUpdateDraw(func() {
			if err != nil {
				status.SetText(fmt.Sprintf("\n  [%s::b]✗ Failed to list upgradable packages[-:-:-]\n", cError))
				return
			}
			if len(pkgs) == 0 {
				status.SetText(fmt.Sprintf("\n  [%s::b]✓ All packages are up to date[-:-:-]\n", cSuccess))
				return
			}

			var b strings.Builder
			b.WriteString(fmt.Sprintf("\n  [%s::b]%d package(s) can be upgraded[-:-:-]\n\n", cWarning, len(pkgs)))
			for _, p := range pkgs {
				installed := p.InstalledVersion
				if installed == "" {
					installed = "?"
				}
				b.WriteString(fmt.Sprintf("  [%s]%-32s[-] %s [%s]→[-] [%s]%s[-]\n",
					cText, p.Name, installed, cMuted, cSuccess, p.Version))
			}
			status.SetText(b.String())
		})
	}()

	status.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			pages.SwitchToPage("menu")
			pages.RemovePage("upgradable")
			return nil
		}
		return event
	})
}

func showPackageAction(app *tview.Application, pages *tview.Pages, title, label string, runner func([]string) (*libapt.Result, error)) {
	if !ensureRoot(app, pages) {
		return
	}

	input := tview.NewInputField().SetLabel(label + ": ")
	form := styleForm(tview.NewForm())
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

	hints := []keyHint{commonBackHint, {"tab", "next field"}, {"↵", "run"}}
	pages.AddAndSwitchToPage("form",
		chrome(centered(form, 70, 9), title, hints), true)
}

func runSyncUpgrade(app *tview.Application, pages *tview.Pages) {
	if !ensureRoot(app, pages) {
		return
	}
	status := tview.NewTextView().SetDynamicColors(true)
	status.SetBorder(true).SetTitle(" Sync + Upgrade ")
	stylePanel(status.Box)
	status.SetText(fmt.Sprintf("\n  [%s]› Step 1/2 · Fetching package indexes…[-]", cInfo))

	hints := []keyHint{commonBackHint}
	pages.AddAndSwitchToPage("run",
		chrome(centered(status, 90, 18), "Sync + Upgrade", hints), true)

	go func() {
		updateRes, updateErr := libapt.Update(false)
		if updateErr != nil {
			app.QueueUpdateDraw(func() {
				status.SetText(fmt.Sprintf("\n  [%s::b]✗ Update failed[-:-:-]\n\n%s\n", cError, updateRes.Output))
			})
			return
		}
		app.QueueUpdateDraw(func() {
			status.SetText(fmt.Sprintf("\n  [%s]› Step 2/2 · Upgrading packages…[-]", cInfo))
		})

		upgradeRes, upgradeErr := libapt.Upgrade(false)
		app.QueueUpdateDraw(func() {
			if upgradeErr != nil {
				status.SetText(fmt.Sprintf("\n  [%s::b]✗ Upgrade failed[-:-:-]\n\n%s\n", cError, upgradeRes.Output))
			} else {
				status.SetText(fmt.Sprintf("\n  [%s::b]✓ System upgraded successfully[-:-:-]\n", cSuccess))
			}
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
	stylePanel(status.Box)
	status.SetText(fmt.Sprintf("\n  [%s]› Working…[-]", cInfo))

	hints := []keyHint{commonBackHint}
	pages.AddAndSwitchToPage("run",
		chrome(centered(status, 90, 18), title, hints), true)

	go func() {
		res, err := runner()
		app.QueueUpdateDraw(func() {
			if err != nil {
				out := strings.TrimSpace(res.Output)
				if out == "" {
					out = err.Error()
				}
				status.SetText(fmt.Sprintf("\n  [%s::b]✗ Operation failed[-:-:-]\n\n%s\n", cError, out))
			} else {
				status.SetText(fmt.Sprintf("\n  [%s::b]✓ Done successfully[-:-:-]\n", cSuccess))
			}
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
