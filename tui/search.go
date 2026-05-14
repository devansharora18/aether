package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/devansharora18/aether/libapt"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func showSearch(app *tview.Application, pages *tview.Pages) {
	input := tview.NewInputField().
		SetLabel("  ").
		SetPlaceholder("Type to search packages…").
		SetFieldBackgroundColor(tcell.ColorDefault).
		SetFieldTextColor(colText).
		SetPlaceholderTextColor(colMuted)
	input.SetBorder(true).SetTitle(" 🔍  Search ")
	stylePanel(input.Box)

	results := styleList(tview.NewList().ShowSecondaryText(true))
	results.SetBorder(true).SetTitle(" Results ")
	stylePanel(results.Box)

	details := tview.NewTextView().SetDynamicColors(true)
	details.SetBorder(true).SetTitle(" Details ")
	stylePanel(details.Box)
	details.SetText(fmt.Sprintf("\n  [%s]Type a query to search packages.[-]", cMuted))

	var mu sync.Mutex
	var cancel context.CancelFunc
	var timer *time.Timer
	requestID := 0
	activeResults := make([]packageResult, 0)

	renderDetails := func(idx int) {
		if idx < 0 || idx >= len(activeResults) {
			details.SetText(fmt.Sprintf("\n  [%s]No package selected[-]", cMuted))
			return
		}
		r := activeResults[idx]
		details.SetText(fmt.Sprintf(
			"\n  [%s::b]%s[-:-:-]\n  [%s]%s · %s · %s[-]\n\n  [%s]%s[-]\n\n  [%s]Loading full details…[-]",
			cTitle, r.name,
			cSubtext, r.repo, r.version, r.arch,
			cText, r.description,
			cMuted,
		))

		go func(name string) {
			pkg, err := libapt.Show(name)
			app.QueueUpdateDraw(func() {
				cur := results.GetCurrentItem()
				if cur < 0 || cur >= len(activeResults) || activeResults[cur].name != name {
					return
				}
				if err != nil || pkg == nil {
					details.SetText(fmt.Sprintf(
						"\n  [%s::b]%s[-:-:-]\n  [%s]%s · %s · %s[-]\n\n  [%s]%s[-]",
						cTitle, r.name,
						cSubtext, r.repo, r.version, r.arch,
						cText, r.description,
					))
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
					b.WriteString(fmt.Sprintf("  [%s]%-14s[-] %s\n", cSubtext, k, v))
				}
				row("Version", pkg.Version)
				if pkg.InstalledVersion != "" {
					row("Installed", fmt.Sprintf("[%s]%s[-]", cSuccess, pkg.InstalledVersion))
				} else {
					row("Installed", fmt.Sprintf("[%s](not installed)[-]", cMuted))
				}
				row("Architecture", pkg.Architecture)
				row("Section", pkg.Section)
				row("Origin", pkg.Origin)
				row("Homepage", pkg.Homepage)
				if pkg.PackageSize > 0 {
					row("Download", libapt.FormatSize(pkg.PackageSize))
				}
				if pkg.InstalledSize > 0 {
					row("Install Size", libapt.FormatSize(pkg.InstalledSize))
				}

				if len(pkg.Depends) > 0 {
					deps := make([]string, 0, len(pkg.Depends))
					for _, d := range pkg.Depends {
						if d.Relation != "" {
							deps = append(deps, fmt.Sprintf("%s (%s %s)", d.Name, d.Relation, d.Version))
						} else {
							deps = append(deps, d.Name)
						}
					}
					b.WriteString(fmt.Sprintf("\n  [%s]Depends[-]\n  %s\n", cSubtext, strings.Join(deps, ", ")))
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

				details.SetText(b.String())
			})
		}(r.name)
	}

	installSelected := func(idx int) {
		if idx < 0 || idx >= len(activeResults) {
			return
		}

		pkg := activeResults[idx]
		modal := tview.NewModal().
			SetText(fmt.Sprintf("Install package '%s'?", pkg.name)).
			AddButtons([]string{"Install", "Cancel"}).
			SetDoneFunc(func(buttonIndex int, buttonLabel string) {
				pages.RemovePage("search-install-modal")
				if buttonIndex != 0 || buttonLabel != "Install" {
					return
				}
				runStreamOperation(app, pages, "Install package", pkg.name, installStream([]string{pkg.name}))
			})
		styleModal(modal)
		modal.SetTitle(" Confirm Install ").SetBorder(true)
		pages.AddPage("search-install-modal", modal, true, true)
	}

	results.SetChangedFunc(func(index int, _, _ string, _ rune) {
		renderDetails(index)
	})
	results.SetSelectedFunc(func(index int, _, _ string, _ rune) {
		installSelected(index)
	})

	kickSearch := func(text string) {
		query := strings.TrimSpace(text)
		if query == "" {
			results.Clear()
			activeResults = activeResults[:0]
			details.SetText(fmt.Sprintf("\n  [%s]Type a query to search packages.[-]", cMuted))
			return
		}

		mu.Lock()
		requestID++
		id := requestID
		if cancel != nil {
			cancel()
		}
		ctx, c := context.WithTimeout(context.Background(), 20*time.Second)
		cancel = c
		mu.Unlock()

		results.Clear()
		results.AddItem("Searching…", query, 0, nil)
		details.SetText(fmt.Sprintf("\n  [%s]Searching for[-] [%s::b]%s[-:-:-]…", cMuted, cInfo, query))

		go func(localID int, localQuery string, localCtx context.Context) {
			res, err := libapt.SearchContext(localCtx, []string{localQuery})
			if localCtx.Err() != nil {
				return
			}
			app.QueueUpdateDraw(func() {
				mu.Lock()
				latest := requestID
				mu.Unlock()
				if localID != latest {
					return
				}

				results.Clear()
				if err != nil {
					activeResults = activeResults[:0]
					results.AddItem("Search failed", "see details", 0, nil)
					details.SetText(fmt.Sprintf("\n  [%s::b]✗ Search failed[-:-:-]\n\n%s", cError, strings.TrimSpace(res.Output)))
					return
				}

				parsed := parseSearchResults(res.Output)
				if len(parsed) == 0 {
					activeResults = activeResults[:0]
					results.AddItem("No packages found", localQuery, 0, nil)
					details.SetText(fmt.Sprintf("\n  [%s]No packages found for[-] [%s::b]%s[-:-:-]", cMuted, cInfo, localQuery))
					return
				}

				sort.Slice(parsed, func(i, j int) bool {
					return parsed[i].name < parsed[j].name
				})
				activeResults = parsed

				limit := len(activeResults)
				if limit > 50 {
					limit = 50
				}

				for i := 0; i < limit; i++ {
					r := activeResults[i]
					secondary := strings.TrimSpace(fmt.Sprintf("%s · %s · %s", r.repo, r.version, r.arch))
					if secondary == "" {
						secondary = r.description
					}
					results.AddItem(r.name, secondary, 0, nil)
				}

				if len(activeResults) > limit {
					results.AddItem("…", fmt.Sprintf("%d more results — refine query", len(activeResults)-limit), 0, nil)
				}

				results.SetTitle(fmt.Sprintf(" Results · %d ", len(activeResults)))
				results.SetCurrentItem(0)
				renderDetails(0)
			})
		}(id, query, ctx)
	}

	input.SetChangedFunc(func(text string) {
		mu.Lock()
		if timer != nil {
			timer.Stop()
		}
		timer = time.AfterFunc(300*time.Millisecond, func() {
			app.QueueUpdateDraw(func() {
				kickSearch(text)
			})
		})
		mu.Unlock()
	})

	body := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(results, 0, 2, false).
		AddItem(details, 0, 3, false)

	searchFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(input, 3, 0, true).
		AddItem(body, 0, 1, false)

	searchFlex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyTAB:
			if app.GetFocus() == input {
				app.SetFocus(results)
			} else {
				app.SetFocus(input)
			}
			return nil
		case tcell.KeyEscape:
			mu.Lock()
			if timer != nil {
				timer.Stop()
			}
			if cancel != nil {
				cancel()
			}
			mu.Unlock()
			pages.SwitchToPage("menu")
			pages.RemovePage("search")
			return nil
		}

		// 'd' to remove selected package
		if app.GetFocus() == results && event.Rune() == 'd' {
			idx := results.GetCurrentItem()
			if idx >= 0 && idx < len(activeResults) {
				pkg := activeResults[idx]
				modal := tview.NewModal().
					SetText(fmt.Sprintf("Remove package '%s'?", pkg.name)).
					AddButtons([]string{"Remove", "Purge", "Cancel"}).
					SetDoneFunc(func(buttonIndex int, buttonLabel string) {
						pages.RemovePage("search-remove-modal")
						switch buttonLabel {
						case "Remove":
							runStreamOperation(app, pages, "Remove package", pkg.name, removeStream([]string{pkg.name}))
						case "Purge":
							runStreamOperation(app, pages, "Purge package", pkg.name, purgeStream([]string{pkg.name}))
						}
					})
				styleModal(modal)
				modal.SetTitle(" Confirm Remove ").SetBorder(true)
				pages.AddPage("search-remove-modal", modal, true, true)
			}
			return nil
		}

		if app.GetFocus() == input && (event.Key() == tcell.KeyDown || event.Key() == tcell.KeyUp) {
			if results.GetItemCount() > 0 {
				app.SetFocus(results)
			}
			return nil
		}

		return event
	})

	hints := []keyHint{
		commonBackHint,
		{"tab", "switch focus"},
		{"↑↓", "navigate"},
		{"↵", "install"},
		{"d", "remove"},
	}
	pages.AddAndSwitchToPage("search", chrome(searchFlex, "Search", hints), true)
	app.SetFocus(input)
}
