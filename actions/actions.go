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

func Install(pkgs []string) {
	if len(pkgs) == 0 {
		ui.Fatal("No package name given.")
		fmt.Fprintln(os.Stderr, "  Usage: aether -S <package> [package...]")
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

// ---------------------------------------------------------------------------
// New operations inspired by python-apt
// ---------------------------------------------------------------------------

func Purge(pkgs []string) {
	if len(pkgs) == 0 {
		ui.Fatal("No package name given.")
		fmt.Fprintln(os.Stderr, "  Usage: aether -Rn <package> [package...]")
		os.Exit(1)
	}
	ui.Header(fmt.Sprintf("Purging (remove + config): %s", strings.Join(pkgs, "  ")))
	if Verbose {
		ui.Info("Invoking: apt remove --purge " + strings.Join(pkgs, " "))
		fmt.Println()
		if _, err := libapt.Purge(pkgs, true); err != nil {
			fmt.Println()
			ui.Fatal("Purge failed — apt exited with an error.")
			os.Exit(1)
		}
		fmt.Println()
		ui.Success("Done. Package(s) purged successfully.")
	} else {
		progress := ui.StartProgress("purging")
		res, err := libapt.Purge(pkgs, false)
		progress.Stop()
		if err != nil {
			if strings.TrimSpace(res.Output) != "" {
				fmt.Print(res.Output)
			}
			ui.Fatal("Purge failed — apt exited with an error.")
			os.Exit(1)
		}
		ui.Success("Done. Package(s) purged successfully.")
	}
}

func AutoRemoveAction() {
	ui.Header("Removing unused dependencies")
	if Verbose {
		ui.Info("Invoking: apt autoremove")
		fmt.Println()
		if _, err := libapt.AutoRemove(true); err != nil {
			fmt.Println()
			ui.Fatal("Autoremove failed — apt exited with an error.")
			os.Exit(1)
		}
		fmt.Println()
		ui.Success("Unused dependencies removed.")
	} else {
		progress := ui.StartProgress("cleaning")
		res, err := libapt.AutoRemove(false)
		progress.Stop()
		if err != nil {
			if strings.TrimSpace(res.Output) != "" {
				fmt.Print(res.Output)
			}
			ui.Fatal("Autoremove failed — apt exited with an error.")
			os.Exit(1)
		}
		ui.Success("Unused dependencies removed.")
	}
}

func ShowInfo(pkgs []string) {
	if len(pkgs) == 0 {
		ui.Fatal("No package name given.")
		fmt.Fprintln(os.Stderr, "  Usage: aether -Qi <package>")
		os.Exit(1)
	}

	for i, name := range pkgs {
		pkg, err := libapt.Show(name)
		if err != nil {
			ui.Fatal(fmt.Sprintf("Could not find package: %s", name))
			continue
		}

		ui.Header(pkg.Name)
		fmt.Printf("  %-18s %s\n", "Version:", pkg.Version)
		if pkg.InstalledVersion != "" {
			fmt.Printf("  %-18s %s\n", "Installed:", pkg.InstalledVersion)
		} else {
			fmt.Printf("  %-18s %s\n", "Installed:", "(not installed)")
		}
		fmt.Printf("  %-18s %s\n", "Architecture:", pkg.Architecture)
		if pkg.Section != "" {
			fmt.Printf("  %-18s %s\n", "Section:", pkg.Section)
		}
		if pkg.Priority != "" {
			fmt.Printf("  %-18s %s\n", "Priority:", pkg.Priority)
		}
		if pkg.Origin != "" {
			fmt.Printf("  %-18s %s\n", "Origin:", pkg.Origin)
		}
		if pkg.Maintainer != "" {
			fmt.Printf("  %-18s %s\n", "Maintainer:", pkg.Maintainer)
		}
		if pkg.Homepage != "" {
			fmt.Printf("  %-18s %s\n", "Homepage:", pkg.Homepage)
		}
		if pkg.PackageSize > 0 {
			fmt.Printf("  %-18s %s\n", "Download Size:", libapt.FormatSize(pkg.PackageSize))
		}
		if pkg.InstalledSize > 0 {
			fmt.Printf("  %-18s %s\n", "Installed Size:", libapt.FormatSize(pkg.InstalledSize))
		}
		if pkg.Source != "" {
			fmt.Printf("  %-18s %s\n", "Source:", pkg.Source)
		}
		if len(pkg.Depends) > 0 {
			deps := make([]string, len(pkg.Depends))
			for j, d := range pkg.Depends {
				if d.Relation != "" {
					deps[j] = fmt.Sprintf("%s (%s %s)", d.Name, d.Relation, d.Version)
				} else {
					deps[j] = d.Name
				}
			}
			fmt.Printf("  %-18s %s\n", "Depends:", strings.Join(deps, ", "))
		}
		if pkg.IsUpgradable {
			ui.Warn(fmt.Sprintf("Upgrade available: %s → %s", pkg.InstalledVersion, pkg.Version))
		}
		if pkg.Description != "" && pkg.Description != pkg.Summary {
			fmt.Printf("\n  %s\n", pkg.Description)
		} else if pkg.Summary != "" {
			fmt.Printf("\n  %s\n", pkg.Summary)
		}

		if i < len(pkgs)-1 {
			fmt.Println()
		}
	}
}

func ListInstalledAction() {
	ui.Header("Installed packages")
	pkgs, err := libapt.ListInstalled()
	if err != nil {
		ui.Fatal("Failed to list installed packages.")
		os.Exit(1)
	}
	for _, p := range pkgs {
		desc := p.Summary
		if len(desc) > 60 {
			desc = desc[:57] + "..."
		}
		fmt.Printf("  %-30s %-20s %s\n", p.Name, p.InstalledVersion, desc)
	}
	ui.Success(fmt.Sprintf("%d packages installed.", len(pkgs)))
}

func ListUpgradableAction() {
	ui.Header("Upgradable packages")
	pkgs, err := libapt.ListUpgradable()
	if err != nil {
		ui.Fatal("Failed to list upgradable packages.")
		os.Exit(1)
	}
	if len(pkgs) == 0 {
		ui.Success("All packages are up to date.")
		return
	}
	for _, p := range pkgs {
		installed := p.InstalledVersion
		if installed == "" {
			installed = "?"
		}
		fmt.Printf("  %-30s %s → %s\n", p.Name, installed, p.Version)
	}
	ui.Success(fmt.Sprintf("%d package(s) can be upgraded.", len(pkgs)))
}

func CleanCache() {
	ui.Header("Cleaning package cache")
	_, err := libapt.Clean()
	if err != nil {
		ui.Fatal("Failed to clean cache.")
		os.Exit(1)
	}
	ui.Success("Package cache cleaned.")
}
