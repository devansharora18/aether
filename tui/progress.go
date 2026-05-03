package tui

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/devansharora18/aether/libapt"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	progressBarWidth = 40
	logTailLines     = 60
)

var (
	reNeedToGet = regexp.MustCompile(`(?i)Need to get ([^.]+?B)\b`)
	reAfter     = regexp.MustCompile(`(?i)After this operation,? ([^.]+?B)\b`)
	reFreed     = regexp.MustCompile(`(?i)After this operation, ([^.]+?B) disk space will be freed`)
	reSummary   = regexp.MustCompile(`(\d+)\s+upgraded,\s+(\d+)\s+newly installed,\s+(\d+)\s+to remove`)
)

// streamRunner is a runner that emits libapt.ProgressEvent updates.
type streamRunner func(libapt.ProgressFn) (*libapt.Result, error)

// opState holds everything the operation view renders.
type opState struct {
	mu sync.Mutex

	title    string
	subtitle string // package name or extra context

	phase      libapt.Phase
	statusText string // current step description
	currentPkg string
	percent    float64
	indeterm   bool

	dlSize    string
	afterSize string
	plan      string // e.g. "1 upgraded, 2 new, 0 removed"

	log []string

	finished bool
	success  bool
	finalMsg string
}

func newOpState(title, subtitle string) *opState {
	return &opState{
		title:    title,
		subtitle: subtitle,
		indeterm: true,
	}
}

func (s *opState) appendLog(line string) {
	line = strings.TrimRight(line, " \r\n\t")
	if line == "" {
		return
	}
	s.log = append(s.log, line)
	if len(s.log) > logTailLines {
		s.log = s.log[len(s.log)-logTailLines:]
	}
}

// scanLog extracts size/plan info from a stdout line.
func (s *opState) scanLog(line string) {
	if m := reFreed.FindStringSubmatch(line); m != nil {
		s.afterSize = "−" + m[1]
	} else if m := reAfter.FindStringSubmatch(line); m != nil && s.afterSize == "" {
		s.afterSize = "+" + m[1]
	}
	if m := reNeedToGet.FindStringSubmatch(line); m != nil {
		s.dlSize = m[1]
	}
	if m := reSummary.FindStringSubmatch(line); m != nil {
		s.plan = fmt.Sprintf("%s upgraded · %s new · %s removed", m[1], m[2], m[3])
	}
}

// renderProgressBar draws a unicode block bar of the given width.
func renderProgressBar(pct float64, width int) string {
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	filled := int(float64(width)*pct/100 + 0.5)
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}
	return fmt.Sprintf("[%s]%s[-][%s]%s[-]",
		cAccent, strings.Repeat("█", filled),
		cBorder, strings.Repeat("░", width-filled),
	)
}

func renderIndeterminateBar(width int) string {
	return fmt.Sprintf("[%s]%s[-]", cBorder, strings.Repeat("░", width))
}

// cleanAPTCache removes cached .deb files
func cleanAPTCache() error {
	_, err := libapt.Clean()
	return err
}

// cleanAPTCacheWithProgress removes cached .deb files and reports progress
func cleanAPTCacheWithProgress(onProgress func(string)) error {
	onProgress("Cleaning apt cache…")
	_, err := libapt.Clean()
	if err == nil {
		onProgress("Cache cleaned successfully")
	}
	return err
}

// removeLockFile removes the APT lock files by killing any held processes first
func removeLockFile() error {
	// Kill any apt/apt-get processes that might be holding the lock
	killCmds := [][]string{
		{"sudo", "killall", "-9", "apt-get"},
		{"sudo", "killall", "-9", "apt"},
		{"sudo", "killall", "-9", "dpkg"},
	}

	for _, killCmd := range killCmds {
		exec.Command(killCmd[0], killCmd[1:]...).Run() // ignore errors, might not be running
	}

	// Wait a moment for processes to die
	time.Sleep(500 * time.Millisecond)

	// Now remove the lock files
	lockFiles := []string{
		"/var/lib/apt/apt.conf.d/lock-frontend",
		"/var/lib/dpkg/lock-frontend",
		"/var/lib/dpkg/lock",
		"/var/cache/apt/archives/lock",
	}

	var lastErr error
	for _, lockFile := range lockFiles {
		cmd := exec.Command("sudo", "rm", "-f", lockFile)
		if err := cmd.Run(); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// removeLockFileWithProgress removes lock files and reports progress
func removeLockFileWithProgress(onProgress func(string)) error {
	onProgress("Killing stale APT processes…")
	killCmds := [][]string{
		{"sudo", "killall", "-9", "apt-get"},
		{"sudo", "killall", "-9", "apt"},
		{"sudo", "killall", "-9", "dpkg"},
	}

	for _, killCmd := range killCmds {
		exec.Command(killCmd[0], killCmd[1:]...).Run()
	}

	onProgress("Waiting for processes to terminate…")
	time.Sleep(500 * time.Millisecond)

	lockFiles := []string{
		"/var/lib/apt/apt.conf.d/lock-frontend",
		"/var/lib/dpkg/lock-frontend",
		"/var/lib/dpkg/lock",
		"/var/cache/apt/archives/lock",
	}

	onProgress(fmt.Sprintf("Removing %d lock files…", len(lockFiles)))
	var lastErr error
	for i, lockFile := range lockFiles {
		onProgress(fmt.Sprintf("Removing lock files… [%d/%d]", i+1, len(lockFiles)))
		cmd := exec.Command("sudo", "rm", "-f", lockFile)
		if err := cmd.Run(); err != nil {
			lastErr = err
		}
	}

	onProgress("Lock removal complete")
	return lastErr
}

// showErrorRecoveryModal displays options to recover from an error
func showErrorRecoveryModal(app *tview.Application, pages *tview.Pages, onRetry func()) {
	text := `An error occurred during the operation.

Common causes:
• APT cache is locked (process holding lock)
• Stale lock files need cleanup

Choose an action:
[Auto-Fix & Retry] will kill any locked processes and
clean up lock files before retrying automatically.`

	modal := tview.NewModal().
		SetText(text).
		AddButtons([]string{"Auto-Fix & Retry", "Manual Fix", "Back"}).
		SetDoneFunc(func(buttonIndex int, _ string) {
			switch buttonIndex {
			case 0: // Auto-Fix & Retry
				pages.RemovePage("error-modal")
				showOperationResult(app, pages, "Fixing Lock Issues", func(onProgress func(string)) error {
					// Kill processes and remove locks
					if err := removeLockFileWithProgress(onProgress); err != nil {
						return err
					}
					// Also clean cache
					return cleanAPTCacheWithProgress(onProgress)
				}, onRetry)
			case 1: // Manual Fix - show individual options
				pages.RemovePage("error-modal")
				showManualFixModal(app, pages, onRetry)
			case 2: // Back
				pages.RemovePage("error-modal")
			}
		})
	styleModal(modal)
	modal.SetTitle(" Error Recovery ").SetBorder(true)
	pages.AddPage("error-modal", modal, true, true)
}

// showManualFixModal shows individual recovery options
func showManualFixModal(app *tview.Application, pages *tview.Pages, onRetry func()) {
	text := `Choose which recovery step to perform:

• Clean Cache: Remove cached .deb files
• Remove Lock: Kill processes and remove lock files
• Retry: Start the operation again`

	modal := tview.NewModal().
		SetText(text).
		AddButtons([]string{"Clean Cache", "Remove Lock", "Retry", "Back"}).
		SetDoneFunc(func(buttonIndex int, _ string) {
			switch buttonIndex {
			case 0: // Clean Cache
				pages.RemovePage("manual-modal")
				showOperationResult(app, pages, "Cleaning Cache", func(onProgress func(string)) error {
					return cleanAPTCacheWithProgress(onProgress)
				}, nil)
			case 1: // Remove Lock
				pages.RemovePage("manual-modal")
				showOperationResult(app, pages, "Removing Lock Files", func(onProgress func(string)) error {
					return removeLockFileWithProgress(onProgress)
				}, nil)
			case 2: // Retry
				pages.RemovePage("manual-modal")
				onRetry()
			case 3: // Back
				pages.RemovePage("manual-modal")
			}
		})
	styleModal(modal)
	modal.SetTitle(" Manual Fix Options ").SetBorder(true)
	pages.AddPage("manual-modal", modal, true, true)
}

// showOperationResult shows the result of a recovery operation with progress
func showOperationResult(app *tview.Application, pages *tview.Pages, title string, operation func(func(string)) error, onSuccess func()) {
	status := tview.NewTextView().SetDynamicColors(true)
	status.SetBorder(true).SetTitle(" " + title + " ")
	stylePanel(status.Box)
	status.SetText(fmt.Sprintf("\n  [%s]Starting…[-]", cInfo))
	status.SetChangedFunc(func() { app.Draw() })

	hints := []keyHint{commonBackHint}
	pages.AddAndSwitchToPage("op-result",
		chrome(status, title, hints), true)

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
			status_msg := currentStatus
			progressMu.Unlock()

			var b strings.Builder
			b.WriteString("\n")

			// Progress indicator with animation
			frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
			frame := frames[stepCount%len(frames)]
			b.WriteString(fmt.Sprintf("  [%s]%s[-] [%s::b]%s[-:-:-]\n\n", cInfo, frame, cInfo, status_msg))

			// Progress bar
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
				// If onSuccess callback provided, call it after a short delay
				if onSuccess != nil {
					time.Sleep(1 * time.Second)
					app.QueueUpdateDraw(func() {
						pages.RemovePage("op-result")
						onSuccess()
					})
					return
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

// runStreamOperation creates a richly rendered operation view and runs the
// streaming runner against it.
func runStreamOperation(app *tview.Application, pages *tview.Pages, title, subtitle string, runner streamRunner) {
	state := newOpState(title, subtitle)

	// Log view: scrollable
	logView := tview.NewTextView().SetDynamicColors(true)
	logView.SetBorder(true)
	stylePanel(logView.Box)
	if subtitle != "" {
		logView.SetTitle(fmt.Sprintf(" %s · %s ", title, subtitle))
	} else {
		logView.SetTitle(" " + title + " ")
	}
	logView.SetChangedFunc(func() { app.Draw() })

	// Progress bar view: fixed, non-scrollable
	progressView := tview.NewTextView().SetDynamicColors(true)
	progressView.SetBorder(true)
	stylePanel(progressView.Box)
	progressView.SetTitle(" Progress ")

	// Container: log on top (scrollable), progress bar on bottom (fixed)
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

		// status line
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

		// progress bar (skip when finished)
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

		// size / plan info
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

	hints := []keyHint{commonBackHint}
	pages.AddAndSwitchToPage("run", chrome(container, title, hints), true)

	var startOperation func() // forward declaration for recursion

	startOperation = func() {
		// Reset state for potential retry
		state.mu.Lock()
		state.finished = false
		state.success = false
		state.finalMsg = ""
		state.log = []string{}
		state.percent = 0
		state.indeterm = true
		state.statusText = ""
		state.currentPkg = ""
		state.mu.Unlock()

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
			state.finished = true
			if err != nil {
				state.success = false
				state.finalMsg = "Operation failed"
				out := strings.TrimSpace(res.Output)
				if out == "" {
					out = err.Error()
				}
				// surface error tail in log if not already there
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
		}()
	}

	startOperation()

	container.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			pages.SwitchToPage("menu")
			pages.RemovePage("run")
			return nil
		}
		return event
	})
}
