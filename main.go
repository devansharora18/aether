package main

import (
	"fmt"
	"os"

	"github.com/devansharora18/aether/actions"
	"github.com/devansharora18/aether/tui"
)

func printUsage() {
	fmt.Printf("Usage: aether [flag] [package(s)]\n\nNo flags launches interactive TUI mode.\n\nFlags:\n  -S <pkg...>     Install packages\n  -R <pkg...>     Remove packages\n  -Rn <pkg...>    Purge packages (remove + config files)\n  -Rc             Remove unused dependencies (autoremove)\n  -Sy             Update package database\n  -Syu            Update + upgrade all packages\n  -Ss <query>     Search packages\n  -Qi <pkg...>    Show detailed package info\n  -Ql             List installed packages\n  -Qu             List upgradable packages\n  -Sc             Clean package cache\n  -v              Verbose apt output\n")
}

func main() {
	// parse optional global -v (verbose) flag anywhere in args
	args := os.Args[1:]
	verbose := false
	filtered := make([]string, 0, len(args))
	for _, a := range args {
		if a == "-v" {
			verbose = true
			continue
		}
		filtered = append(filtered, a)
	}

	// expose verbosity to actions package
	actions.Verbose = verbose

	if len(filtered) < 1 {
		if err := tui.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "TUI failed: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	flag := filtered[0]
	rest := filtered[1:]

	switch flag {
	case "-S":
		actions.Install(rest)
	case "-R":
		actions.Remove(rest)
	case "-Rn":
		actions.Purge(rest)
	case "-Rc":
		actions.AutoRemoveAction()
	case "-Sy":
		actions.Sync()
	case "-Syu":
		actions.SyncUpgrade()
	case "-Ss":
		actions.Search(rest)
	case "-Qi":
		actions.ShowInfo(rest)
	case "-Ql":
		actions.ListInstalledAction()
	case "-Qu":
		actions.ListUpgradableAction()
	case "-Sc":
		actions.CleanCache()
	case "-h", "--help", "help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown flag: %s\n", flag)
		printUsage()
		os.Exit(1)
	}
}
