package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ANSI color/style codes
const (
	reset        = "\033[0m"
	bold         = "\033[1m"
	dim          = "\033[2m"
	red          = "\033[31m"
	green        = "\033[32m"
	yellow       = "\033[33m"
	blue         = "\033[34m"
	cyan         = "\033[36m"
	brightRed    = "\033[91m"
	brightGreen  = "\033[92m"
	brightYellow = "\033[93m"
	brightBlue   = "\033[94m"
	brightCyan   = "\033[96m"
	brightWhite  = "\033[97m"
)

// ── UI helpers ────────────────────────────────────────────────────────────────

func header(msg string) {
	fmt.Printf("\n%s%s:: %s%s\n", bold, brightBlue, msg, reset)
	fmt.Printf("%s%s%s\n", dim, strings.Repeat("─", len(msg)+4), reset)
}

func success(msg string) {
	fmt.Printf("%s%s ✓  %s%s%s\n", bold, brightGreen, reset, msg, reset)
}

func info(msg string) {
	fmt.Printf("%s%s →  %s%s%s\n", bold, brightCyan, reset, msg, reset)
}

func warn(msg string) {
	fmt.Printf("%s%s ⚠  %s%s%s\n", bold, brightYellow, reset, msg, reset)
}

func fatal(msg string) {
	fmt.Fprintf(os.Stderr, "%s%s ✗  %s%s%s\n", bold, brightRed, reset, msg, reset)
}

// ── Core runner ───────────────────────────────────────────────────────────────

// runApt executes apt with the given arguments, streaming all output directly
// to the user's terminal so interactive prompts still work.
func runApt(args ...string) error {
	cmd := exec.Command("apt", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func needsRoot(op string) bool {
	if os.Geteuid() != 0 {
		fatal("This operation requires root privileges.")
		warn(fmt.Sprintf("Re-run with: sudo aether %s", op))
		return false
	}
	return true
}

// ── Command handlers ──────────────────────────────────────────────────────────

func cmdInstall(pkgs []string) {
	if len(pkgs) == 0 {
		fatal("No package name given.")
		fmt.Fprintln(os.Stderr, "  Usage: aether -S <package> [package...]")
		os.Exit(1)
	}
	if !needsRoot("-S " + strings.Join(pkgs, " ")) {
		os.Exit(1)
	}

	header(fmt.Sprintf("Installing: %s%s%s%s", bold, brightWhite, strings.Join(pkgs, "  "), reset))
	info("Invoking: apt install " + strings.Join(pkgs, " "))
	fmt.Println()

	if err := runApt(append([]string{"install", "-y"}, pkgs...)...); err != nil {
		fmt.Println()
		fatal("Installation failed — apt exited with an error.")
		os.Exit(1)
	}
	fmt.Println()
	success("Done. Package(s) installed successfully.")
}

func cmdRemove(pkgs []string) {
	if len(pkgs) == 0 {
		fatal("No package name given.")
		fmt.Fprintln(os.Stderr, "  Usage: aether -R <package> [package...]")
		os.Exit(1)
	}
	if !needsRoot("-R " + strings.Join(pkgs, " ")) {
		os.Exit(1)
	}

	header(fmt.Sprintf("Removing: %s%s%s%s", bold, brightWhite, strings.Join(pkgs, "  "), reset))
	info("Invoking: apt remove " + strings.Join(pkgs, " "))
	fmt.Println()

	if err := runApt(append([]string{"remove", "-y"}, pkgs...)...); err != nil {
		fmt.Println()
		fatal("Removal failed — apt exited with an error.")
		os.Exit(1)
	}
	fmt.Println()
	success("Done. Package(s) removed successfully.")
}

func cmdSync() {
	if !needsRoot("-Sy") {
		os.Exit(1)
	}

	header("Synchronizing package databases")
	info("Invoking: apt update")
	fmt.Println()

	if err := runApt("update"); err != nil {
		fmt.Println()
		fatal("Database sync failed — apt exited with an error.")
		os.Exit(1)
	}
	fmt.Println()
	success("Package databases are up to date.")
}

func cmdSyncUpgrade() {
	if !needsRoot("-Syu") {
		os.Exit(1)
	}

	header("Synchronizing package databases")
	info("Invoking: apt update")
	fmt.Println()

	if err := runApt("update"); err != nil {
		fmt.Println()
		fatal("Database sync failed.")
		os.Exit(1)
	}
	fmt.Println()
	success("Package databases synchronized.")

	header("Upgrading all installed packages")
	info("Invoking: apt upgrade")
	fmt.Println()

	if err := runApt("upgrade", "-y"); err != nil {
		fmt.Println()
		fatal("Upgrade failed — apt exited with an error.")
		os.Exit(1)
	}
	fmt.Println()
	success("System upgraded successfully.")
}

func cmdSearch(terms []string) {
	if len(terms) == 0 {
		fatal("No search query given.")
		fmt.Fprintln(os.Stderr, "  Usage: aether -Ss <query>")
		os.Exit(1)
	}

	header(fmt.Sprintf("Searching for: %s%s%s%s", bold, brightWhite, strings.Join(terms, " "), reset))
	info("Invoking: apt search " + strings.Join(terms, " "))
	fmt.Println()

	if err := runApt(append([]string{"search"}, terms...)...); err != nil {
		fmt.Println()
		fatal("Search failed — apt exited with an error.")
		os.Exit(1)
	}
}

// ── Usage ─────────────────────────────────────────────────────────────────────

func printUsage() {
	fmt.Printf(`
%s%saether%s %s— a modern APT frontend%s
%s%s%s

%s%sUsage%s
  aether %s<flag>%s [package(s)]

%s%sFlags%s
  %s-S%s  <pkg...>   %sInstall%s one or more packages
  %s-R%s  <pkg...>   %sRemove%s one or more packages
  %s-Sy%s            %sSync%s package databases        (apt update)
  %s-Syu%s           %sSync%s databases + upgrade all  (apt update && apt upgrade)
  %s-Ss%s <query>    %sSearch%s for a package

%s%sExamples%s
  sudo aether -S neovim curl
  sudo aether -R firefox
  sudo aether -Sy
  sudo aether -Syu
  aether -Ss ripgrep

`,
		bold, brightCyan, reset,
		dim, reset,
		dim, strings.Repeat("─", 40), reset,

		bold, yellow, reset,
		brightWhite, reset,

		bold, yellow, reset,
		bold+brightGreen, reset, bold, reset,
		bold+brightGreen, reset, bold, reset,
		bold+brightGreen, reset, bold, reset,
		bold+brightGreen, reset, bold, reset,
		bold+brightGreen, reset, bold, reset,

		bold, yellow, reset,
	)
}

// ── Entry point ───────────────────────────────────────────────────────────────

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}

	flag := os.Args[1]
	rest := os.Args[2:]

	switch flag {
	case "-S":
		cmdInstall(rest)
	case "-R":
		cmdRemove(rest)
	case "-Sy":
		cmdSync()
	case "-Syu":
		cmdSyncUpgrade()
	case "-Ss":
		cmdSearch(rest)
	case "-h", "--help", "help":
		printUsage()
	default:
		fatal("Unknown flag: " + flag)
		printUsage()
		os.Exit(1)
	}
}
