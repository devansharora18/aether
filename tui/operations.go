package tui

import (
	"fmt"
	"strings"

	"github.com/devansharora18/aether/libapt"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// streamRunner adapters used by the top-level menu — all run via apt-get with
// APT::Status-Fd so the operation view can render real progress.
func installStream(pkgs []string) streamRunner {
	return func(p libapt.ProgressFn) (*libapt.Result, error) {
		return libapt.InstallWithProgress(pkgs, p)
	}
}
func removeStream(pkgs []string) streamRunner {
	return func(p libapt.ProgressFn) (*libapt.Result, error) {
		return libapt.RemoveWithProgress(pkgs, p)
	}
}
func purgeStream(pkgs []string) streamRunner {
	return func(p libapt.ProgressFn) (*libapt.Result, error) {
		return libapt.PurgeWithProgress(pkgs, p)
	}
}
func updateStream() streamRunner {
	return func(p libapt.ProgressFn) (*libapt.Result, error) {
		return libapt.UpdateWithProgress(p)
	}
}
func upgradeStream() streamRunner {
	return func(p libapt.ProgressFn) (*libapt.Result, error) {
		return libapt.UpgradeWithProgress(p)
	}
}
func autoremoveStream() streamRunner {
	return func(p libapt.ProgressFn) (*libapt.Result, error) {
		return libapt.AutoRemoveWithProgress(p)
	}
}
func cleanStream() streamRunner {
	return func(_ libapt.ProgressFn) (*libapt.Result, error) {
		return libapt.Clean()
	}
}

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

func showPackageAction(app *tview.Application, pages *tview.Pages, title, label string, makeRunner func([]string) streamRunner) {
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
		ensureSudo(app, pages, func(ok bool) {
			if !ok {
				return
			}
			runStreamOperation(app, pages, title, strings.Join(pkgs, " "), makeRunner(pkgs))
		})
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

// runSyncUpgrade chains update then upgrade through the streaming view.
func runSyncUpgrade(app *tview.Application, pages *tview.Pages) {
	combined := func(p libapt.ProgressFn) (*libapt.Result, error) {
		updRes, err := libapt.UpdateWithProgress(p)
		if err != nil {
			return updRes, err
		}
		// surface a phase boundary in the log
		if p != nil {
			p(libapt.ProgressEvent{Phase: libapt.PhaseLog, LogLine: "── upgrade ──"})
		}
		return libapt.UpgradeWithProgress(p)
	}
	runStreamOperation(app, pages, "Sync + Upgrade", "", combined)
}
