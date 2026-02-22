package ui

import (
	"fmt"
	"os"
	"strings"
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
