package tui

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/devansharora18/aether/libapt"
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
	menu.AddItem("Install package", "Equivalent to -S", 'i', func() { showPackageAction(app, pages, "Install package", "Package(s)", func(pkgs []string) (*libapt.Result, error) { return libapt.Install(pkgs, false) }) })
	menu.AddItem("Remove package", "Equivalent to -R", 'r', func() { showPackageAction(app, pages, "Remove package", "Package(s)", func(pkgs []string) (*libapt.Result, error) { return libapt.Remove(pkgs, false) }) })
	menu.AddItem("Sync package database", "Equivalent to -Sy", 's', func() { runOperation(app, pages, "Sync package database", func() (*libapt.Result, error) { return libapt.Update(false) }) })
	menu.AddItem("Sync + Upgrade", "Equivalent to -Syu", 'u', func() { runSyncUpgrade(app, pages) })
	menu.AddItem("Search packages", "Live results as you type", '/', func() { showSearch(app, pages) })
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
		details.SetText(fmt.Sprintf(
			"[green::b]%s[-]\n[gray]repo:[-] %s\n[gray]version:[-] %s\n[gray]arch:[-] %s\n\n%s\n\n[gray]Press Enter to install selected package[-]",
			r.name,
			r.repo,
			r.version,
			r.arch,
			r.description,
		))
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

func addBackHint(tv *tview.TextView) {
	current := tv.GetText(true)
	tv.SetText(current + "\n\n[gray]Press Esc to go back[-]")
}

func centered(p tview.Primitive, width, height int) tview.Primitive {
	return tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(p, height, 1, true).
			AddItem(nil, 0, 1, false), width, 1, true).
		AddItem(nil, 0, 1, false)
}

type packageResult struct {
	name        string
	repo        string
	version     string
	arch        string
	description string
}

func formatSearchResults(query, raw string) string {
	results := parseSearchResults(raw)
	if len(results) == 0 {
		return fmt.Sprintf("[yellow]No packages found[-] for [cyan]%s[-]", query)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].name < results[j].name
	})

	var b strings.Builder
	b.WriteString(fmt.Sprintf("[cyan::b]Results for:[-] [white]%s[-]\n\n", query))

	limit := len(results)
	if limit > 25 {
		limit = 25
	}

	for i := 0; i < limit; i++ {
		r := results[i]
		b.WriteString(fmt.Sprintf("[green::b]%s[-] [gray](%s %s %s)[-]\n", r.name, r.repo, r.version, r.arch))
		if r.description != "" {
			b.WriteString("  " + r.description + "\n")
		}
		b.WriteString("\n")
	}

	if len(results) > limit {
		b.WriteString(fmt.Sprintf("[gray]%d more result(s) not shown. Refine your query.[-]\n", len(results)-limit))
	}

	return b.String()
}

func parseSearchResults(raw string) []packageResult {
	lines := strings.Split(raw, "\n")
	out := make([]packageResult, 0)
	current := -1

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		if strings.HasPrefix(trimmed, "WARNING:") ||
			trimmed == "Sorting..." ||
			trimmed == "Full Text Search..." ||
			strings.HasPrefix(trimmed, "N: ") {
			continue
		}

		if strings.HasPrefix(line, "  ") {
			if current >= 0 {
				if out[current].description == "" {
					out[current].description = trimmed
				} else {
					out[current].description += " " + trimmed
				}
			}
			continue
		}

		fields := strings.Fields(trimmed)
		if len(fields) < 2 || !strings.Contains(fields[0], "/") {
			continue
		}

		nameRepo := strings.SplitN(fields[0], "/", 2)
		repo := ""
		if len(nameRepo) == 2 {
			repo = nameRepo[1]
			if comma := strings.Index(repo, ","); comma != -1 {
				repo = repo[:comma]
			}
		}

		version := fields[1]
		arch := ""
		if len(fields) > 2 {
			arch = fields[2]
		}

		out = append(out, packageResult{
			name:    nameRepo[0],
			repo:    repo,
			version: version,
			arch:    arch,
		})
		current = len(out) - 1
	}

	return out
}
