package main

import (
	"fmt"
	"os"

	"github.com/devansharora18/aether/actions"
)

func printUsage() {
	fmt.Printf("Usage: aether <flag> [package(s)]\n\nFlags:\n  -S <pkg...>     Install packages\n  -R <pkg...>     Remove packages\n  -Sy             Update package database\n  -Syu            Update + upgrade\n  -Ss <query>     Search packages\n")
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}

	flag := os.Args[1]
	rest := os.Args[2:]

	switch flag {
	case "-S":
		actions.Install(rest)
	case "-R":
		actions.Remove(rest)
	case "-Sy":
		actions.Sync()
	case "-Syu":
		actions.SyncUpgrade()
	case "-Ss":
		actions.Search(rest)
	case "-h", "--help", "help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown flag: %s\n", flag)
		printUsage()
		os.Exit(1)
	}
}
