package libapt

import (
	"bufio"
	"bytes"
	"context"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Phase describes the kind of progress event coming from apt-get.
type Phase string

const (
	PhaseDownload Phase = "download"
	PhaseInstall  Phase = "install"
	PhaseError    Phase = "error"
	PhaseConfFile Phase = "conffile"
	PhaseMedia    Phase = "media"
	PhaseLog      Phase = "log"
)

// ProgressEvent is one streamed update from a running apt-get operation.
type ProgressEvent struct {
	Phase       Phase
	Package     string  // package name (when applicable)
	Description string  // human-readable status from apt
	Percent     float64 // overall progress for this phase, 0–100
	LogLine     string  // raw stdout/stderr line (PhaseLog only)
}

// ProgressFn is the callback invoked for each event during a streamed run.
type ProgressFn func(ev ProgressEvent)

// runWithProgress executes apt-get, streaming both stdout/stderr (as PhaseLog
// events) and the machine-readable APT::Status-Fd channel (as PhaseDownload /
// PhaseInstall / PhaseError events). The returned Result aggregates all log
// output the same way run() does.
func runWithProgress(args []string, onEvent ProgressFn) (*Result, error) {
	if onEvent == nil {
		return run(args, false)
	}

	pr, pw, err := os.Pipe()
	if err != nil {
		return run(args, false)
	}

	fullArgs := append([]string{"-o", "APT::Status-Fd=3", "-o", "Dpkg::Use-Pty=0"}, args...)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "apt-get", fullArgs...)
	cmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
	cmd.ExtraFiles = []*os.File{pw}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		pr.Close()
		pw.Close()
		return &Result{Output: "", Err: err}, err
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		pr.Close()
		pw.Close()
		return &Result{Output: "", Err: err}, err
	}

	if err := cmd.Start(); err != nil {
		pr.Close()
		pw.Close()
		return &Result{Output: "", Err: err}, err
	}
	pw.Close() // child has its own copy on FD 3

	var (
		bufMu sync.Mutex
		out   bytes.Buffer
	)
	writeLog := func(line string) {
		bufMu.Lock()
		out.WriteString(line)
		out.WriteByte('\n')
		bufMu.Unlock()
	}

	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		s := bufio.NewScanner(pr)
		s.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for s.Scan() {
			onEvent(parseStatusFd(s.Text()))
		}
	}()

	go func() {
		defer wg.Done()
		s := bufio.NewScanner(stdoutPipe)
		s.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for s.Scan() {
			line := s.Text()
			writeLog(line)
			onEvent(ProgressEvent{Phase: PhaseLog, LogLine: line})
		}
	}()

	go func() {
		defer wg.Done()
		s := bufio.NewScanner(stderrPipe)
		s.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for s.Scan() {
			line := s.Text()
			writeLog(line)
			onEvent(ProgressEvent{Phase: PhaseLog, LogLine: line})
		}
	}()

	waitErr := cmd.Wait()
	pr.Close()
	wg.Wait()

	bufMu.Lock()
	output := out.String()
	bufMu.Unlock()

	return &Result{Output: output, Err: waitErr}, waitErr
}

// parseStatusFd parses a single line from the APT::Status-Fd channel.
// Format: <type>:<id-or-pkg>:<percent>:<description>
func parseStatusFd(line string) ProgressEvent {
	parts := strings.SplitN(line, ":", 4)
	if len(parts) < 4 {
		return ProgressEvent{Phase: PhaseLog, LogLine: line}
	}
	pct, _ := strconv.ParseFloat(strings.TrimSpace(parts[2]), 64)
	pkg := strings.TrimSpace(parts[1])
	desc := strings.TrimSpace(parts[3])

	var phase Phase
	switch parts[0] {
	case "dlstatus":
		phase = PhaseDownload
	case "pmstatus":
		phase = PhaseInstall
	case "pmerror":
		phase = PhaseError
	case "pmconffile":
		phase = PhaseConfFile
	case "media-change":
		phase = PhaseMedia
	default:
		return ProgressEvent{Phase: PhaseLog, LogLine: line}
	}

	return ProgressEvent{
		Phase:       phase,
		Package:     pkg,
		Description: desc,
		Percent:     pct,
	}
}
