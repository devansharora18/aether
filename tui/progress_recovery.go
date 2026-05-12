package tui

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// showErrorRecoveryModal displays options to recover from an error.
func showErrorRecoveryModal(app *tview.Application, pages *tview.Pages, onRetry func()) {
	text := `An error occurred during the operation.

Common causes:
• APT cache is locked (process holding lock)
• Stale lock files need cleanup
• Unmet dependencies need repair

Choose an action:
[Auto-Fix & Retry] will kill any locked processes and
clean up lock files, clear cache, and repair dependencies before retrying automatically.`

	modal := tview.NewModal().
		SetText(text).
		AddButtons([]string{"Auto-Fix & Retry", "Manual Fix", "Back"}).
		SetDoneFunc(func(buttonIndex int, _ string) {
			pages.RemovePage("error-modal")

			switch buttonIndex {
			case 0:
				go func() {
					time.Sleep(50 * time.Millisecond)
					app.QueueUpdateDraw(func() {
						showOperationResult(app, pages, "Fixing Lock Issues", func(onProgress func(string)) error {
							if err := removeLockFileWithProgress(onProgress); err != nil {
								return err
							}
							if err := cleanAPTCacheWithProgress(onProgress); err != nil {
								return err
							}
							return fixBrokenDependenciesWithProgress(onProgress)
						}, onRetry)
					})
				}()
			case 1:
				go func() {
					time.Sleep(50 * time.Millisecond)
					app.QueueUpdateDraw(func() {
						showManualFixModal(app, pages, onRetry)
					})
				}()
			case 2:
				// do nothing, modal is already removed
			}
		})
	styleModal(modal)
	modal.SetTitle(" Error Recovery ").SetBorder(true)
	pages.AddPage("error-modal", modal, true, true)
}

// showManualFixModal shows individual recovery options.
func showManualFixModal(app *tview.Application, pages *tview.Pages, onRetry func()) {
	text := `Choose which recovery step to perform:

• Clean Cache: Remove cached .deb files
• Remove Lock: Kill processes and remove lock files
• Fix Broken: Repair unmet dependencies
• Retry: Start the operation again`
	modal := tview.NewModal().
		SetText(text).
		AddButtons([]string{"Clean Cache", "Remove Lock", "Fix Broken", "Retry", "Back"}).
		SetDoneFunc(func(buttonIndex int, _ string) {
			pages.RemovePage("manual-modal")

			switch buttonIndex {
			case 0:
				go func() {
					time.Sleep(50 * time.Millisecond)
					app.QueueUpdateDraw(func() {
						showOperationResult(app, pages, "Cleaning Cache", func(onProgress func(string)) error {
							return cleanAPTCacheWithProgress(onProgress)
						}, nil)
					})
				}()
			case 1:
				go func() {
					time.Sleep(50 * time.Millisecond)
					app.QueueUpdateDraw(func() {
						showOperationResult(app, pages, "Removing Lock Files", func(onProgress func(string)) error {
							return removeLockFileWithProgress(onProgress)
						}, nil)
					})
				}()
			case 2:
				go func() {
					time.Sleep(50 * time.Millisecond)
					app.QueueUpdateDraw(func() {
						showOperationResult(app, pages, "Fixing Broken Dependencies", func(onProgress func(string)) error {
							return fixBrokenDependenciesWithProgress(onProgress)
						}, nil)
					})
				}()
			case 3:
				go func() {
					time.Sleep(50 * time.Millisecond)
					app.QueueUpdateDraw(func() {
						onRetry()
					})
				}()
			case 4:
				// just close the modal
			}
		})
	styleModal(modal)
	modal.SetTitle(" Manual Fix Options ").SetBorder(true)
	pages.AddPage("manual-modal", modal, true, true)
}

// showActiveOperations displays all running background operations.
func showActiveOperations(app *tview.Application, pages *tview.Pages) {
	list := tview.NewList()
	styleList(list)
	list.SetBorder(true).SetTitle(" Background Operations ")
	visibleOpIDs := make([]string, 0)

	updateList := func() {
		list.Clear()
		visibleOpIDs = visibleOpIDs[:0]
		ops := getActiveOps()

		if len(ops) == 0 {
			list.AddItem("No active operations", "", 0, nil)
			return
		}

		idx := 0
		for id, op := range ops {
			op.mu.Lock()
			title := op.title
			subtitle := op.subtitle
			percent := op.percent
			finished := op.finished
			canceled := op.canceled
			op.mu.Unlock()

			var status string
			if canceled {
				status = "✗ Cancelled"
			} else if finished {
				status = "✓ Completed"
			} else {
				status = fmt.Sprintf("⟳ %5.1f%%", percent)
			}

			displayText := fmt.Sprintf("%s %s", status, title)
			if subtitle != "" {
				displayText += fmt.Sprintf(" · %s", subtitle)
			}

			visibleOpIDs = append(visibleOpIDs, id)
			list.AddItem(displayText, "", rune('1'+idx), func() {
				pages.RemovePage("active-ops")
				pages.ShowPage("run")
				pages.SwitchToPage("run")
			})
			idx++
		}
	}

	updateList()

	go func() {
		for {
			time.Sleep(500 * time.Millisecond)
			app.QueueUpdateDraw(func() {
				updateList()
			})
		}
	}()

	hints := []keyHint{commonBackHint, {"↑↓", "navigate"}, {"↵", "view"}, {"c", "cancel"}, {"v", "view run"}}
	pages.AddAndSwitchToPage("active-ops", chrome(list, "Background Operations", hints), true)

	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			pages.RemovePage("active-ops")
			pages.SwitchToPage("menu")
			return nil
		}
		if event.Rune() == 'v' || event.Rune() == 'V' {
			pages.ShowPage("run")
			pages.SwitchToPage("run")
			return nil
		}
		if event.Rune() == 'c' || event.Rune() == 'C' {
			idx := list.GetCurrentItem()
			if idx >= 0 && idx < len(visibleOpIDs) {
				ops := getActiveOps()
				if op, ok := ops[visibleOpIDs[idx]]; ok {
					modal := tview.NewModal().
						SetText("Cancel selected operation?").
						AddButtons([]string{"Cancel Operation", "Keep Running"}).
						SetDoneFunc(func(buttonIndex int, _ string) {
							pages.RemovePage("cancel-op-modal")
							if buttonIndex == 0 {
								go func() {
									cancelOperation(op)
									app.QueueUpdateDraw(func() {
										updateList()
									})
								}()
							}
						})
					styleModal(modal)
					modal.SetTitle(" Confirm Cancel ").SetBorder(true)
					pages.AddPage("cancel-op-modal", modal, true, true)
				}
			}
			return nil
		}
		if event.Key() == tcell.KeyEnter {
			if idx := list.GetCurrentItem(); idx >= 0 && idx < len(visibleOpIDs) {
				pages.SwitchToPage("run")
			}
			return nil
		}
		return event
	})
}

// showOperationResult shows the result of a recovery operation with progress.
func showOperationResult(app *tview.Application, pages *tview.Pages, title string, operation func(func(string)) error, onSuccess func()) {
	status := tview.NewTextView().SetDynamicColors(true)
	status.SetBorder(true).SetTitle(" " + title + " ")
	stylePanel(status.Box)
	status.SetText(fmt.Sprintf("\n  [%s]Starting…[-]", cInfo))
	status.SetChangedFunc(func() { app.Draw() })

	hints := []keyHint{commonBackHint}
	pages.AddAndSwitchToPage("op-result", chrome(status, title, hints), true)

	var progressMu sync.Mutex
	var currentStatus string
	var stepCount int

	onProgress := func(msg string) {
		progressMu.Lock()
		currentStatus = msg
		stepCount++
		progressMu.Unlock()

		app.QueueUpdateDraw(func() {
			progressMu.Lock()
			statusMsg := currentStatus
			progressMu.Unlock()

			var b strings.Builder
			b.WriteString("\n")

			frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
			frame := frames[stepCount%len(frames)]
			b.WriteString(fmt.Sprintf("  [%s]%s[-] [%s::b]%s[-:-:-]\n\n", cInfo, frame, cInfo, statusMsg))

			b.WriteString(fmt.Sprintf("  [%s]", cBorder))
			for i := 0; i < 30; i++ {
				if i < stepCount%30 {
					b.WriteString("█")
				} else {
					b.WriteString("░")
				}
			}
			b.WriteString("[-]\n")

			status.SetText(b.String())
		})
	}

	go func() {
		err := operation(onProgress)
		successCallback := onSuccess
		app.QueueUpdateDraw(func() {
			if err != nil {
				var b strings.Builder
				b.WriteString(fmt.Sprintf("\n  [%s::b]✗ %s failed[-:-:-]\n\n", cError, title))
				b.WriteString(fmt.Sprintf("  [%s]%s[-]\n", cMuted, err.Error()))
				status.SetText(b.String())
			} else {
				var b strings.Builder
				b.WriteString(fmt.Sprintf("\n  [%s::b]✓ %s succeeded[-:-:-]\n", cSuccess, title))
				status.SetText(b.String())

				if successCallback != nil {
					go func() {
						time.Sleep(1 * time.Second)
						app.QueueUpdateDraw(func() {
							if pages.HasPage("run") {
								pages.SwitchToPage("run")
							}
							pages.RemovePage("op-result")
							successCallback()
						})
					}()
				}
			}
		})
	}()

	status.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			pages.SwitchToPage("menu")
			pages.RemovePage("op-result")
			return nil
		}
		return event
	})
}
