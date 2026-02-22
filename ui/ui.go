package ui

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	reset        = "\033[0m"
	bold         = "\033[1m"
	dim          = "\033[2m"
	brightRed    = "\033[91m"
	brightGreen  = "\033[92m"
	brightYellow = "\033[93m"
	brightBlue   = "\033[94m"
	brightCyan   = "\033[96m"
	brightWhite  = "\033[97m"
)

func Header(msg string) {
	fmt.Printf("\n%s%s:: %s%s\n", bold, brightBlue, msg, reset)
	fmt.Printf("%s%s%s\n", dim, strings.Repeat("─", len(msg)+4), reset)
}

func Success(msg string) {
	fmt.Printf("%s%s ✓  %s%s%s\n", bold, brightGreen, reset, msg, reset)
}

func Info(msg string) {
	fmt.Printf("%s%s →  %s%s%s\n", bold, brightCyan, reset, msg, reset)
}

func Warn(msg string) {
	fmt.Printf("%s%s ⚠  %s%s%s\n", bold, brightYellow, reset, msg, reset)
}

func Fatal(msg string) {
	fmt.Fprintf(os.Stderr, "%s%s ✗  %s%s%s\n", bold, brightRed, reset, msg, reset)
}

type ProgressHandle struct {
	message string
	stopCh  chan struct{}
	done    chan struct{}
	once    sync.Once
}

func StartProgress(message string) *ProgressHandle {
	h := &ProgressHandle{
		message: message,
		stopCh:  make(chan struct{}),
		done:    make(chan struct{}),
	}

	go func() {
		defer close(h.done)
		dots := 0
		ticker := time.NewTicker(300 * time.Millisecond)
		defer ticker.Stop()

		for {
			fmt.Printf("\r%s%s%s", brightCyan, h.message+strings.Repeat(".", dots), reset)
			select {
			case <-h.stopCh:
				return
			case <-ticker.C:
				dots = (dots + 1) % 6
			}
		}
	}()

	return h
}

func (h *ProgressHandle) Stop() {
	h.once.Do(func() {
		close(h.stopCh)
		<-h.done
		fmt.Print("\r\033[K")
	})
}
