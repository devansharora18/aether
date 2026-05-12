package tui

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/devansharora18/aether/libapt"
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

// Background operations tracking
var (
	activeOpsMu sync.Mutex
	activeOps   = make(map[string]*opState)
)

func addActiveOp(id string, op *opState) {
	activeOpsMu.Lock()
	activeOps[id] = op
	activeOpsMu.Unlock()
}

func removeActiveOp(id string) {
	activeOpsMu.Lock()
	delete(activeOps, id)
	activeOpsMu.Unlock()
}

func getActiveOps() map[string]*opState {
	activeOpsMu.Lock()
	defer activeOpsMu.Unlock()
	result := make(map[string]*opState)
	for k, v := range activeOps {
		result[k] = v
	}
	return result
}

// streamRunner is a runner that emits libapt.ProgressEvent updates.
type streamRunner func(libapt.ProgressFn) (*libapt.Result, error)

// opState holds everything the operation view renders.
type opState struct {
	mu sync.Mutex
	id string

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
	canceled bool

	// Cancellation
	ctx    context.Context
	cancel context.CancelFunc
}

func newOpState(title, subtitle string) *opState {
	ctx, cancel := context.WithCancel(context.Background())
	return &opState{
		title:    title,
		subtitle: subtitle,
		indeterm: true,
		ctx:      ctx,
		cancel:   cancel,
	}
}

func cancelAptProcesses() {
	// Best effort cancellation for apt/dpkg subprocesses spawned by operations.
	for _, c := range [][]string{{"killall", "-2", "apt-get"}, {"killall", "-2", "apt"}, {"killall", "-2", "dpkg"}} {
		exec.Command("sudo", c...).Run()
	}
}

func cancelOperation(state *opState) {
	state.mu.Lock()
	if state.finished {
		state.mu.Unlock()
		return
	}
	state.canceled = true
	state.finished = true
	state.success = false
	state.finalMsg = "Cancelled by user"
	state.appendLog("operation cancelled by user")
	state.mu.Unlock()

	state.cancel()
	cancelAptProcesses()
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

// fixBrokenDependenciesWithProgress runs apt --fix-broken install.
func fixBrokenDependenciesWithProgress(onProgress func(string)) error {
	onProgress("Fixing broken dependencies…")
	_, err := libapt.FixBrokenInstall(false)
	if err == nil {
		onProgress("Dependency repair completed")
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
