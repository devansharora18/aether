package libapt

import (
    "strings"
)

func Install(pkgs []string, stream bool) (*Result, error) {
    args := append([]string{"install", "-y"}, pkgs...)
    res, err := runElevated(args, stream)
    if err != nil {
        return res, classifyError("install", res.Output, err)
    }
    return res, nil
}

func Remove(pkgs []string, stream bool) (*Result, error) {
    args := append([]string{"remove", "-y"}, pkgs...)
    res, err := runElevated(args, stream)
    if err != nil {
        return res, classifyError("remove", res.Output, err)
    }
    return res, nil
}

func Update(stream bool) (*Result, error) {
    res, err := runElevated([]string{"update"}, stream)
    if err != nil {
        return res, classifyError("update", res.Output, err)
    }
    return res, nil
}

func Upgrade(stream bool) (*Result, error) {
    res, err := runElevated([]string{"upgrade", "-y"}, stream)
    if err != nil {
        return res, classifyError("upgrade", res.Output, err)
    }
    return res, nil
}

func Search(terms []string, stream bool) (*Result, error) {
    args := append([]string{"search"}, terms...)
    return run(args, stream)
}

// Purge removes packages AND their configuration files (apt remove --purge).
func Purge(pkgs []string, stream bool) (*Result, error) {
    args := append([]string{"remove", "--purge", "-y"}, pkgs...)
    res, err := runElevated(args, stream)
    if err != nil {
        return res, classifyError("purge", res.Output, err)
    }
    return res, nil
}

// AutoRemove removes packages that were automatically installed to satisfy
// dependencies and are no longer needed.
func AutoRemove(stream bool) (*Result, error) {
    res, err := runElevated([]string{"autoremove", "-y"}, stream)
    if err != nil {
        return res, classifyError("autoremove", res.Output, err)
    }
    return res, nil
}

// DistUpgrade performs a distribution upgrade (install/remove as needed).
func DistUpgrade(stream bool) (*Result, error) {
    res, err := run([]string{"full-upgrade", "-y"}, stream)
    if err != nil {
        return res, classifyError("dist-upgrade", res.Output, err)
    }
    return res, nil
}

// ShowRaw returns the raw `apt show` output for a package.
func ShowRaw(pkg string) (*Result, error) {
    return run([]string{"show", pkg}, false)
}

// Show fetches detailed package metadata
func Show(name string) (*Package, error) {
    res, err := run([]string{"show", name}, false)
    if err != nil {
        return nil, classifyError("show", res.Output, err)
    }
    p := parseShowOutput(res.Output)
    if p.Name == "" {
        p.Name = name
    }

    // augment with installed-version from dpkg if present
    iv, _ := getInstalledVersion(name)
    p.InstalledVersion = iv
    p.IsInstalled = iv != ""

    // check if upgradable
    if p.IsInstalled && p.Version != "" && p.InstalledVersion != p.Version {
        p.IsUpgradable = true
    }

    return p, nil
}

// ListInstalled returns all installed packages
func ListInstalled() ([]Package, error) {
    out, err := runDpkg([]string{"-W", "--showformat=${Package}\t${Version}\t${Architecture}\t${db:Status-Abbrev}\t${binary:Summary}\n"})
    if err != nil {
        return nil, classifyError("list-installed", out, err)
    }
    var pkgs []Package
    for _, line := range strings.Split(out, "\n") {
        fields := strings.SplitN(line, "\t", 5)
        if len(fields) < 4 {
            continue
        }
        status := strings.TrimSpace(fields[3])
        if !strings.HasPrefix(status, "ii") {
            continue // only fully installed
        }
        p := Package{
            Name:             fields[0],
            InstalledVersion: fields[1],
            Architecture:     fields[2],
            IsInstalled:      true,
        }
        if len(fields) == 5 {
            p.Summary = fields[4]
        }
        pkgs = append(pkgs, p)
    }
    return pkgs, nil
}

// ListUpgradable returns packages that have a newer version available
func ListUpgradable() ([]Package, error) {
    res, err := run([]string{"list", "--upgradable"}, false)
    if err != nil {
        return nil, classifyError("list-upgradable", res.Output, err)
    }
    return parseListOutput(res.Output, true), nil
}

// IsInstalled quickly checks whether a package is installed.
func IsInstalled(name string) bool {
    v, _ := getInstalledVersion(name)
    return v != ""
}

// GetDependencies returns the dependency list for a package.
func GetDependencies(name string) ([]Dependency, error) {
    p, err := Show(name)
    if err != nil {
        return nil, err
    }
    return p.Depends, nil
}

// CountInstalled returns the number of installed packages.
func CountInstalled() (int, error) {
    out, err := runDpkg([]string{"--list"})
    if err != nil {
        return 0, err
    }
    count := 0
    for _, line := range strings.Split(out, "\n") {
        if strings.HasPrefix(line, "ii") {
            count++
        }
    }
    return count, nil
}

// Clean removes cached .deb files.
func Clean() (*Result, error) {
    return runElevated([]string{"clean"}, false)
}

// FixBrokenInstall attempts to repair broken dependencies.
func FixBrokenInstall(stream bool) (*Result, error) {
    res, err := runElevated([]string{"--fix-broken", "install", "-y"}, stream)
    if err != nil {
        return res, classifyError("fix-broken", res.Output, err)
    }
    return res, nil
}

// InstallWithProgress installs packages while emitting progress events.
func InstallWithProgress(pkgs []string, onEvent ProgressFn) (*Result, error) {
    args := append([]string{"install", "-y"}, pkgs...)
    res, err := runWithProgressElevated(args, onEvent)
    if err != nil {
        return res, classifyError("install", res.Output, err)
    }
    return res, nil
}

// RemoveWithProgress removes packages while emitting progress events.
func RemoveWithProgress(pkgs []string, onEvent ProgressFn) (*Result, error) {
    args := append([]string{"remove", "-y"}, pkgs...)
    res, err := runWithProgressElevated(args, onEvent)
    if err != nil {
        return res, classifyError("remove", res.Output, err)
    }
    return res, nil
}

// PurgeWithProgress remove+purge while emitting progress events.
func PurgeWithProgress(pkgs []string, onEvent ProgressFn) (*Result, error) {
    args := append([]string{"remove", "--purge", "-y"}, pkgs...)
    res, err := runWithProgressElevated(args, onEvent)
    if err != nil {
        return res, classifyError("purge", res.Output, err)
    }
    return res, nil
}

// UpdateWithProgress updates package indexes with progress events.
func UpdateWithProgress(onEvent ProgressFn) (*Result, error) {
    res, err := runWithProgressElevated([]string{"update"}, onEvent)
    if err != nil {
        return res, classifyError("update", res.Output, err)
    }
    return res, nil
}

// UpgradeWithProgress upgrades packages with progress events.
func UpgradeWithProgress(onEvent ProgressFn) (*Result, error) {
    res, err := runWithProgressElevated([]string{"upgrade", "-y"}, onEvent)
    if err != nil {
        return res, classifyError("upgrade", res.Output, err)
    }
    return res, nil
}

// AutoRemoveWithProgress autoremoves packages with progress events.
func AutoRemoveWithProgress(onEvent ProgressFn) (*Result, error) {
    res, err := runWithProgressElevated([]string{"autoremove", "-y"}, onEvent)
    if err != nil {
        return res, classifyError("autoremove", res.Output, err)
    }
    return res, nil
}
