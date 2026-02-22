package actions

import (
	"fmt"
	"os"
	"strings"

	"github.com/devansharora18/aether/libapt"
	"github.com/devansharora18/aether/ui"
)

// Verbose controls whether apt output is streamed live. When false, some
// noisy commands (update/upgrade) will show a minimal "fetching..." message
// instead of the full apt progress lines.
var Verbose = false

func needsRoot(op string) bool {
	if os.Geteuid() != 0 {
		ui.Fatal("This operation requires root privileges.")
		ui.Warn(fmt.Sprintf("Re-run with: sudo aether %s", op))
		return false
	}
	return true
}

func Install(pkgs []string) {
	if len(pkgs) == 0 {
		ui.Fatal("No package name given.")
		fmt.Fprintln(os.Stderr, "  Usage: aether -S <package> [package...]")
		os.Exit(1)
	}
	if !needsRoot("-S " + strings.Join(pkgs, " ")) {
		os.Exit(1)
	}

	ui.Header(fmt.Sprintf("Installing: %s", strings.Join(pkgs, "  ")))
	if Verbose {
		ui.Info("Invoking: apt install " + strings.Join(pkgs, " "))
		fmt.Println()
		if _, err := libapt.Install(pkgs, true); err != nil {
			fmt.Println()
			ui.Fatal("Installation failed — apt exited with an error.")
			os.Exit(1)
		}
		fmt.Println()
		ui.Success("Done. Package(s) installed successfully.")
	} else {
		progress := ui.StartProgress("fetching")
		res, err := libapt.Install(pkgs, false)
		progress.Stop()
		if err != nil {
			if strings.TrimSpace(res.Output) != "" {
				fmt.Print(res.Output)
			}
			ui.Fatal("Installation failed — apt exited with an error.")
			os.Exit(1)
		}
		ui.Success("Done. Package(s) installed successfully.")
	}
}

func Remove(pkgs []string) {
	if len(pkgs) == 0 {
		ui.Fatal("No package name given.")
		fmt.Fprintln(os.Stderr, "  Usage: aether -R <package> [package...]")
		os.Exit(1)
	}
	if !needsRoot("-R " + strings.Join(pkgs, " ")) {
		os.Exit(1)
	}

	ui.Header(fmt.Sprintf("Removing: %s", strings.Join(pkgs, "  ")))
	if Verbose {
		ui.Info("Invoking: apt remove " + strings.Join(pkgs, " "))
		fmt.Println()
		if _, err := libapt.Remove(pkgs, true); err != nil {
			fmt.Println()
			ui.Fatal("Removal failed — apt exited with an error.")
			os.Exit(1)
		}
		fmt.Println()
		ui.Success("Done. Package(s) removed successfully.")
	} else {
		progress := ui.StartProgress("removing")
		res, err := libapt.Remove(pkgs, false)
		progress.Stop()
		if err != nil {
			if strings.TrimSpace(res.Output) != "" {
				fmt.Print(res.Output)
			}
			ui.Fatal("Removal failed — apt exited with an error.")
			os.Exit(1)
		}
		ui.Success("Done. Package(s) removed successfully.")
	}
}

func Sync() {
	if !needsRoot("-Sy") {
		os.Exit(1)
	}

	ui.Header("Synchronizing package databases")
	if Verbose {
		ui.Info("Invoking: apt update")
		fmt.Println()
		if _, err := libapt.Update(true); err != nil {
			fmt.Println()
			ui.Fatal("Database sync failed — apt exited with an error.")
			os.Exit(1)
		}
		fmt.Println()
		ui.Success("Package databases are up to date.")
	} else {
		progress := ui.StartProgress("fetching")
		res, err := libapt.Update(false)
		progress.Stop()
		if err != nil {
			if strings.TrimSpace(res.Output) != "" {
				fmt.Print(res.Output)
			}
			ui.Fatal("Database sync failed — apt exited with an error.")
			os.Exit(1)
		}
		ui.Success("Package databases are up to date.")
	}
}

func SyncUpgrade() {
	if !needsRoot("-Syu") {
		os.Exit(1)
	}
	ui.Header("Synchronizing package databases")
	if Verbose {
		ui.Info("Invoking: apt update")
		fmt.Println()
		if _, err := libapt.Update(true); err != nil {
			fmt.Println()
			ui.Fatal("Database sync failed.")
			os.Exit(1)
		}
		fmt.Println()
		ui.Success("Package databases synchronized.")
	} else {
		progress := ui.StartProgress("fetching")
		res, err := libapt.Update(false)
		progress.Stop()
		if err != nil {
			if strings.TrimSpace(res.Output) != "" {
				fmt.Print(res.Output)
			}
			ui.Fatal("Database sync failed.")
			os.Exit(1)
		}
		ui.Success("Package databases synchronized.")
	}

	ui.Header("Upgrading all installed packages")
	if Verbose {
		ui.Info("Invoking: apt upgrade")
		fmt.Println()
		if _, err := libapt.Upgrade(true); err != nil {
			fmt.Println()
			ui.Fatal("Upgrade failed — apt exited with an error.")
			os.Exit(1)
		}
		fmt.Println()
		ui.Success("System upgraded successfully.")
	} else {
		progress := ui.StartProgress("upgrading")
		res, err := libapt.Upgrade(false)
		progress.Stop()
		if err != nil {
			if strings.TrimSpace(res.Output) != "" {
				fmt.Print(res.Output)
			}
			ui.Fatal("Upgrade failed — apt exited with an error.")
			os.Exit(1)
		}
		ui.Success("System upgraded successfully.")
	}
}

func Search(terms []string) {
	if len(terms) == 0 {
		ui.Fatal("No search query given.")
		fmt.Fprintln(os.Stderr, "  Usage: aether -Ss <query>")
		os.Exit(1)
	}

	ui.Header(fmt.Sprintf("Searching for: %s", strings.Join(terms, " ")))
	ui.Info("Invoking: apt search " + strings.Join(terms, " "))
	fmt.Println()

	res, err := libapt.Search(terms, false)
	if err != nil {
		fmt.Println()
		ui.Fatal("Search failed — apt exited with an error.")
		os.Exit(1)
	}
	fmt.Print(res.Output)
}
