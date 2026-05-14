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
	return runWithProgressBinary("apt-get", args, onEvent, false)
}

func runWithProgressElevated(args []string, onEvent ProgressFn) (*Result, error) {
	return runWithProgressBinary("apt-get", args, onEvent, true)
}

func runWithProgressBinary(binary string, args []string, onEvent ProgressFn, elevated bool) (*Result, error) {
	if onEvent == nil {
		if elevated {
			return runElevated(args, false)
		}
		return run(args, false)
	}

	useStatusOnStdout := elevated && os.Geteuid() != 0
	statusFd := "3"
	if useStatusOnStdout {
		statusFd = "1"
	}

	fullArgs := append([]string{"-o", "APT::Status-Fd=" + statusFd, "-o", "Dpkg::Use-Pty=0"}, args...)
	cmdBinary := binary
	cmdArgs := fullArgs
	if elevated && os.Geteuid() != 0 {
		cmdBinary = "sudo"
		cmdArgs = append([]string{binary}, fullArgs...)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, cmdBinary, cmdArgs...)
	cmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")

	var pr *os.File
	if !useStatusOnStdout {
		pipeReader, pipeWriter, err := os.Pipe()
		if err != nil {
			if elevated {
				return runElevated(args, false)
			}
			return run(args, false)
		}
		pr = pipeReader
		cmd.ExtraFiles = []*os.File{pipeWriter}
		defer pipeWriter.Close()
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		if pr != nil {
			pr.Close()
		}
		return &Result{Output: "", Err: err}, err
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		if pr != nil {
			pr.Close()
		}
		return &Result{Output: "", Err: err}, err
	}

	if err := cmd.Start(); err != nil {
		if pr != nil {
			pr.Close()
		}
		return &Result{Output: "", Err: err}, err
	}
	if pr != nil {
		// child has its own copy on FD 3
		defer pr.Close()
	}

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

	if pr != nil {
		go func() {
			defer wg.Done()
			s := bufio.NewScanner(pr)
			s.Buffer(make([]byte, 0, 64*1024), 1024*1024)
			for s.Scan() {
				onEvent(parseStatusFd(s.Text()))
			}
		}()
	} else {
		wg.Done()
	}

	go func() {
		defer wg.Done()
		s := bufio.NewScanner(stdoutPipe)
		s.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for s.Scan() {
			line := s.Text()
			if useStatusOnStdout {
				ev := parseStatusFd(line)
				if ev.Phase != PhaseLog {
					onEvent(ev)
					continue
				}
			}
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
	if pr != nil {
		pr.Close()
	}
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
