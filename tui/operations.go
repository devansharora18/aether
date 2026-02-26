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
func updateRunner() (*libapt.Result, error)                { return libapt.Update(false) }
func upgradeRunner() (*libapt.Result, error)               { return libapt.Upgrade(false) }
func autoremoveRunner() (*libapt.Result, error)            { return libapt.AutoRemove(false) }
func cleanRunner() (*libapt.Result, error)                 { return libapt.Clean() }

// showPackageInfo prompts for a package name and displays rich details.
func showPackageInfo(app *tview.Application, pages *tview.Pages) {
	input := tview.NewInputField().SetLabel("Package: ")
	form := tview.NewForm()
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
	pages.AddAndSwitchToPage("info-form", centered(form, 70, 12), true)
}

func showPkgDetails(app *tview.Application, pages *tview.Pages, name string) {
	status := tview.NewTextView().SetDynamicColors(true)
	status.SetBorder(true).SetTitle(" Package Info: " + name + " ")
	status.SetText("[cyan]loading[-]")
	pages.AddAndSwitchToPage("pkg-detail", status, true)

	go func() {
		pkg, err := libapt.Show(name)
		app.QueueUpdateDraw(func() {
			if err != nil {
				status.SetText("[red]Could not find package[-]: " + name)
				addBackHint(status)
				return
			}

			var b strings.Builder
			b.WriteString(fmt.Sprintf("[green::b]%s[-]\n\n", pkg.Name))
			b.WriteString(fmt.Sprintf("  %-20s %s\n", "Version:", pkg.Version))
			if pkg.InstalledVersion != "" {
				b.WriteString(fmt.Sprintf("  %-20s %s\n", "Installed:", pkg.InstalledVersion))
			} else {
				b.WriteString(fmt.Sprintf("  %-20s %s\n", "Installed:", "(not installed)"))
			}
			b.WriteString(fmt.Sprintf("  %-20s %s\n", "Architecture:", pkg.Architecture))
			if pkg.Section != "" {
				b.WriteString(fmt.Sprintf("  %-20s %s\n", "Section:", pkg.Section))
			}
			if pkg.Priority != "" {
				b.WriteString(fmt.Sprintf("  %-20s %s\n", "Priority:", pkg.Priority))
			}
			if pkg.Origin != "" {
				b.WriteString(fmt.Sprintf("  %-20s %s\n", "Origin:", pkg.Origin))
			}
			if pkg.Maintainer != "" {
				b.WriteString(fmt.Sprintf("  %-20s %s\n", "Maintainer:", pkg.Maintainer))
			}
			if pkg.Homepage != "" {
				b.WriteString(fmt.Sprintf("  %-20s %s\n", "Homepage:", pkg.Homepage))
			}
			if pkg.PackageSize > 0 {
				b.WriteString(fmt.Sprintf("  %-20s %s\n", "Download Size:", libapt.FormatSize(pkg.PackageSize)))
			}
			if pkg.InstalledSize > 0 {
				b.WriteString(fmt.Sprintf("  %-20s %s\n", "Installed Size:", libapt.FormatSize(pkg.InstalledSize)))
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
				b.WriteString(fmt.Sprintf("\n  [yellow]Dependencies:[-] %s\n", strings.Join(deps, ", ")))
			}
			if pkg.IsUpgradable {
				b.WriteString(fmt.Sprintf("\n  [yellow::b]⚠ Upgrade available:[-] %s → %s\n", pkg.InstalledVersion, pkg.Version))
			}
			if pkg.Description != "" && pkg.Description != pkg.Summary {
				b.WriteString(fmt.Sprintf("\n  %s\n", pkg.Description))
			} else if pkg.Summary != "" {
				b.WriteString(fmt.Sprintf("\n  %s\n", pkg.Summary))
			}

			status.SetText(b.String())
			addBackHint(status)
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
	status.SetText("[cyan]loading[-]")
	pages.AddAndSwitchToPage("upgradable", status, true)

	go func() {
		pkgs, err := libapt.ListUpgradable()
		app.QueueUpdateDraw(func() {
			if err != nil {
				status.SetText("[red]Failed to list upgradable packages[-]")
				addBackHint(status)
				return
			}
			if len(pkgs) == 0 {
				status.SetText("[green]All packages are up to date![-]")
				addBackHint(status)
				return
			}

			var b strings.Builder
			b.WriteString(fmt.Sprintf("[yellow::b]%d package(s) can be upgraded:[-]\n\n", len(pkgs)))
			for _, p := range pkgs {
				installed := p.InstalledVersion
				if installed == "" {
					installed = "?"
				}
				b.WriteString(fmt.Sprintf("  %-30s %s → %s\n", p.Name, installed, p.Version))
			}
			status.SetText(b.String())
			addBackHint(status)
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
