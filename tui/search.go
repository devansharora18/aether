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
	input := tview.NewInputField().SetLabel("Search: ")
	results := tview.NewList().ShowSecondaryText(true)
	results.SetBorder(true).SetTitle(" Results (↑/↓ navigate, Enter install, Tab switch focus) ")
	details := tview.NewTextView().SetDynamicColors(true)
	details.SetBorder(true).SetTitle(" Details ")
	details.SetText("Type to search packages...")

	var mu sync.Mutex
	var cancel context.CancelFunc
	var timer *time.Timer
	requestID := 0
	activeResults := make([]packageResult, 0)

	renderDetails := func(idx int) {
		if idx < 0 || idx >= len(activeResults) {
			details.SetText("No package selected")
			return
		}
		r := activeResults[idx]
		// Show basic info immediately, then fetch full details in background
		details.SetText(fmt.Sprintf(
			"[green::b]%s[-]\n[gray]repo:[-] %s\n[gray]version:[-] %s\n[gray]arch:[-] %s\n\n%s\n\n[gray]Loading full details...[-]",
			r.name, r.repo, r.version, r.arch, r.description,
		))

		// Fetch rich package info asynchronously (like python-apt Package properties)
		go func(name string) {
			pkg, err := libapt.Show(name)
			app.QueueUpdateDraw(func() {
				// make sure we're still looking at the same package
				cur := results.GetCurrentItem()
				if cur < 0 || cur >= len(activeResults) || activeResults[cur].name != name {
					return
				}
				if err != nil || pkg == nil {
					details.SetText(fmt.Sprintf(
						"[green::b]%s[-]\n[gray]repo:[-] %s\n[gray]version:[-] %s\n[gray]arch:[-] %s\n\n%s\n\n[gray]Press Enter to install selected package[-]",
						r.name, r.repo, r.version, r.arch, r.description,
					))
					return
				}

				var b strings.Builder
				b.WriteString(fmt.Sprintf("[green::b]%s[-]\n\n", pkg.Name))
				b.WriteString(fmt.Sprintf("  [gray]Version:[-]       %s\n", pkg.Version))
				if pkg.InstalledVersion != "" {
					b.WriteString(fmt.Sprintf("  [gray]Installed:[-]     %s\n", pkg.InstalledVersion))
				} else {
					b.WriteString("  [gray]Installed:[-]     (not installed)\n")
				}
				b.WriteString(fmt.Sprintf("  [gray]Architecture:[-]  %s\n", pkg.Architecture))
				if pkg.Section != "" {
					b.WriteString(fmt.Sprintf("  [gray]Section:[-]       %s\n", pkg.Section))
				}
				if pkg.Origin != "" {
					b.WriteString(fmt.Sprintf("  [gray]Origin:[-]        %s\n", pkg.Origin))
				}
				if pkg.Homepage != "" {
					b.WriteString(fmt.Sprintf("  [gray]Homepage:[-]      %s\n", pkg.Homepage))
				}
				if pkg.PackageSize > 0 {
					b.WriteString(fmt.Sprintf("  [gray]Download:[-]      %s\n", libapt.FormatSize(pkg.PackageSize)))
				}
				if pkg.InstalledSize > 0 {
					b.WriteString(fmt.Sprintf("  [gray]Installed Size:[-] %s\n", libapt.FormatSize(pkg.InstalledSize)))
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
					b.WriteString(fmt.Sprintf("\n  [yellow]Depends:[-] %s\n", strings.Join(deps, ", ")))
				}
				if pkg.IsUpgradable {
					b.WriteString(fmt.Sprintf("\n  [yellow::b]⚠ Upgrade available:[-] %s → %s\n", pkg.InstalledVersion, pkg.Version))
				}
				if pkg.Description != "" && pkg.Description != pkg.Summary {
					b.WriteString(fmt.Sprintf("\n  %s\n", pkg.Description))
				} else if pkg.Summary != "" {
					b.WriteString(fmt.Sprintf("\n  %s\n", pkg.Summary))
				}
				b.WriteString("\n[gray]Press Enter to install • 'd' for remove • Esc to go back[-]")
				details.SetText(b.String())
			})
		}(r.name)
	}

	installSelected := func(idx int) {
		if idx < 0 || idx >= len(activeResults) {
			return
		}
		if !ensureRoot(app, pages) {
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
				runOperation(app, pages, "Install package", func() (*libapt.Result, error) {
					return libapt.Install([]string{pkg.name}, false)
				})
			})
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
			details.SetText("Type to search packages...")
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
		results.AddItem("Searching...", query, 0, nil)
		details.SetText(fmt.Sprintf("Searching for [cyan]%s[-]", query))

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
					results.AddItem("Search failed", "See details panel", 0, nil)
					details.SetText("[red]Search failed[-]\n\n" + strings.TrimSpace(res.Output))
					return
				}

				parsed := parseSearchResults(res.Output)
				if len(parsed) == 0 {
					activeResults = activeResults[:0]
					results.AddItem("No packages found", localQuery, 0, nil)
					details.SetText(fmt.Sprintf("[yellow]No packages found[-] for [cyan]%s[-]", localQuery))
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
					secondary := strings.TrimSpace(fmt.Sprintf("%s  %s  %s", r.repo, r.version, r.arch))
					if secondary == "" {
						secondary = r.description
					}
					results.AddItem(r.name, secondary, 0, nil)
				}

				if len(activeResults) > limit {
					results.AddItem("…", fmt.Sprintf("%d more results (refine query)", len(activeResults)-limit), 0, nil)
				}

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
	searchFlex.SetBorder(true).SetTitle(" Live Search (-Ss) ")
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
				if !ensureRoot(app, pages) {
					return nil
				}
				pkg := activeResults[idx]
				modal := tview.NewModal().
					SetText(fmt.Sprintf("Remove package '%s'?", pkg.name)).
					AddButtons([]string{"Remove", "Purge", "Cancel"}).
					SetDoneFunc(func(buttonIndex int, buttonLabel string) {
						pages.RemovePage("search-remove-modal")
						switch buttonLabel {
						case "Remove":
							runOperation(app, pages, "Remove package", func() (*libapt.Result, error) {
								return libapt.Remove([]string{pkg.name}, false)
							})
						case "Purge":
							runOperation(app, pages, "Purge package", func() (*libapt.Result, error) {
								return libapt.Purge([]string{pkg.name}, false)
							})
						}
					})
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

	pages.AddAndSwitchToPage("search", searchFlex, true)
	app.SetFocus(input)
}
