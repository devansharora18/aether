package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/devansharora18/aether/libapt"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// runStreamOperation creates a richly rendered operation view and runs the
// streaming runner against it.
func runStreamOperation(app *tview.Application, pages *tview.Pages, title, subtitle string, runner streamRunner) {
	state := newOpState(title, subtitle)

	logView := tview.NewTextView().SetDynamicColors(true)
	logView.SetBorder(true)
	stylePanel(logView.Box)
	if subtitle != "" {
		logView.SetTitle(fmt.Sprintf(" %s · %s ", title, subtitle))
	} else {
		logView.SetTitle(" " + title + " ")
	}
	logView.SetChangedFunc(func() { app.Draw() })

	progressView := tview.NewTextView().SetDynamicColors(true)
	progressView.SetBorder(true)
	stylePanel(progressView.Box)
	progressView.SetTitle(" Progress ")

	container := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(logView, 0, 1, true).
		AddItem(progressView, 6, 0, false)

	renderLog := func() {
		state.mu.Lock()
		defer state.mu.Unlock()

		var b strings.Builder
		b.WriteString(fmt.Sprintf("  [%s]── log ─────────────────────────────────────────────[-]\n", cBorder))
		for _, line := range state.log {
			b.WriteString(fmt.Sprintf("  [%s]%s[-]\n", cMuted, tview.Escape(line)))
		}

		logView.SetText(b.String())
		logView.ScrollToEnd()
	}

	renderProgress := func() {
		state.mu.Lock()
		defer state.mu.Unlock()

		var b strings.Builder
		b.WriteString("\n")

		switch {
		case state.finished && state.success:
			b.WriteString(fmt.Sprintf("  [%s::b]✓ %s[-:-:-]\n", cSuccess, state.finalMsg))
		case state.finished && !state.success:
			b.WriteString(fmt.Sprintf("  [%s::b]✗ %s[-:-:-]\n", cError, state.finalMsg))
		default:
			label := state.statusText
			if label == "" {
				label = "Starting"
			}
			b.WriteString(fmt.Sprintf("  [%s]›[-] [%s::b]%s[-:-:-]", cInfo, cText, label))
			if state.currentPkg != "" {
				b.WriteString(fmt.Sprintf("  [%s]%s[-]", cTitle, state.currentPkg))
			}
			b.WriteString("\n")
		}

		if !state.finished {
			b.WriteString("\n  ")
			if state.indeterm {
				b.WriteString(renderIndeterminateBar(progressBarWidth))
				b.WriteString(fmt.Sprintf("  [%s]…[-]\n", cMuted))
			} else {
				b.WriteString(renderProgressBar(state.percent, progressBarWidth))
				b.WriteString(fmt.Sprintf("  [%s]%5.1f%%[-]\n", cSubtext, state.percent))
			}
		}

		details := make([]string, 0, 3)
		if state.dlSize != "" {
			details = append(details, fmt.Sprintf("[%s]download[-] [%s]%s[-]", cSubtext, cText, state.dlSize))
		}
		if state.afterSize != "" {
			details = append(details, fmt.Sprintf("[%s]disk[-] [%s]%s[-]", cSubtext, cText, state.afterSize))
		}
		if state.plan != "" {
			details = append(details, fmt.Sprintf("[%s]%s[-]", cSubtext, state.plan))
		}
		if len(details) > 0 {
			b.WriteString(fmt.Sprintf("\n  %s\n", strings.Join(details, fmt.Sprintf("   [%s]·[-]   ", cBorder))))
		}

		progressView.SetText(b.String())
	}

	renderLog()
	renderProgress()

	hints := []keyHint{commonBackHint, {"c", "cancel"}}
	pages.AddAndSwitchToPage("run", chrome(container, title, hints), true)

	var startOperation func()
	startOperation = func() {
		state.mu.Lock()
		state.finished = false
		state.success = false
		state.finalMsg = ""
		state.log = []string{}
		state.percent = 0
		state.indeterm = true
		state.statusText = ""
		state.currentPkg = ""
		state.canceled = false
		state.mu.Unlock()

		pages.SwitchToPage("run")

		renderLog()
		renderProgress()

		onEvent := func(ev libapt.ProgressEvent) {
			state.mu.Lock()
			switch ev.Phase {
			case libapt.PhaseDownload:
				state.indeterm = false
				state.phase = libapt.PhaseDownload
				state.statusText = "Downloading"
				state.percent = ev.Percent
				if ev.Package != "" && ev.Package != "0" {
					state.currentPkg = ev.Package
				}
				if ev.Description != "" {
					state.appendLog(ev.Description)
				}
			case libapt.PhaseInstall:
				state.indeterm = false
				state.phase = libapt.PhaseInstall
				state.statusText = "Installing"
				state.percent = ev.Percent
				state.currentPkg = ev.Package
				if ev.Description != "" {
					state.appendLog(ev.Description)
				}
			case libapt.PhaseError:
				state.appendLog("✗ " + ev.Description)
			case libapt.PhaseConfFile:
				state.appendLog("conffile: " + ev.Description)
			case libapt.PhaseLog:
				state.scanLog(ev.LogLine)
				state.appendLog(ev.LogLine)
			}
			state.mu.Unlock()
			app.QueueUpdateDraw(func() {
				renderLog()
				renderProgress()
			})
		}

		go func() {
			res, err := runner(onEvent)
			state.mu.Lock()
			if state.canceled {
				state.mu.Unlock()
				app.QueueUpdateDraw(func() {
					renderLog()
					renderProgress()
				})
				return
			}
			state.finished = true
			if err != nil {
				state.success = false
				state.finalMsg = "Operation failed"
				out := strings.TrimSpace(res.Output)
				if out == "" {
					out = err.Error()
				}
				tail := strings.Split(out, "\n")
				if len(tail) > 5 {
					tail = tail[len(tail)-5:]
				}
				for _, l := range tail {
					state.appendLog(l)
				}
				state.mu.Unlock()
				app.QueueUpdateDraw(func() {
					renderLog()
					renderProgress()
					showErrorRecoveryModal(app, pages, startOperation)
				})
			} else {
				state.success = true
				state.finalMsg = "Done successfully"
				state.mu.Unlock()
				app.QueueUpdateDraw(func() {
					renderLog()
					renderProgress()
				})
			}
			state.cancel()
		}()
	}

	startOperation()

	opID := state.title + "-" + state.subtitle
	state.id = opID
	addActiveOp(opID, state)

	container.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			state.mu.Lock()
			if !state.finished {
				state.appendLog("moved to background")
			}
			state.mu.Unlock()
			go func() {
				app.QueueUpdateDraw(func() {
					pages.HidePage("run")
					pages.SwitchToPage("menu")
				})
			}()
			return nil
		}
		if event.Rune() == 'c' || event.Rune() == 'C' {
			go func() {
				cancelOperation(state)
				app.QueueUpdateDraw(func() {
					renderLog()
					renderProgress()
				})
			}()
			return nil
		}
		return event
	})

	go func() {
		<-state.ctx.Done()
		if state.finished {
			time.Sleep(2 * time.Second)
		}
		removeActiveOp(opID)
	}()
}