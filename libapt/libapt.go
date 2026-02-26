package libapt

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)


// AptError is the base error type returned by libapt operations.
type AptError struct {
	Op     string // high-level operation name (install, remove, …)
	Output string // captured apt output
	Inner  error  // underlying exec error, if any
}

func (e *AptError) Error() string {
	if e.Inner != nil {
		return fmt.Sprintf("%s: %v", e.Op, e.Inner)
	}
	return e.Op
}

func (e *AptError) Unwrap() error { return e.Inner }

// LockError is returned when apt cannot acquire the dpkg lock.
type LockError struct{ AptError }

// DependencyError is returned when apt fails due to unmet dependencies.
type DependencyError struct{ AptError }

// NotFoundError is returned when a requested package does not exist.
type NotFoundError struct {
	AptError
	Package string
}

// classifyError inspects apt output and wraps the raw error in a more specific type.
func classifyError(op, output string, err error) error {
	if err == nil {
		return nil
	}
	base := AptError{Op: op, Output: output, Inner: err}
	low := strings.ToLower(output)
	switch {
	case strings.Contains(low, "unable to acquire the dpkg frontend lock") ||
		strings.Contains(low, "could not get lock"):
		return &LockError{base}
	case strings.Contains(low, "unmet dependencies") ||
		strings.Contains(low, "dependency problems"):
		return &DependencyError{base}
	case strings.Contains(low, "unable to locate package"):
		// try to pull out the package name
		pkg := ""
		if idx := strings.Index(output, "Unable to locate package "); idx >= 0 {
			pkg = strings.TrimSpace(output[idx+len("Unable to locate package "):])
			if nl := strings.IndexByte(pkg, '\n'); nl > 0 {
				pkg = pkg[:nl]
			}
		}
		return &NotFoundError{AptError: base, Package: pkg}
	default:
		return &base
	}
}


// Dependency describes a single dependency relationship.
type Dependency struct {
	Name     string // package name
	Relation string // version relation (>=, <<, =, …)
	Version  string // version string
}

// Package holds metadata for a single package, similar to python-apt's Package
// class which exposes name, version, architecture, description, dependencies,
// size, installed status, upgradability, etc.
type Package struct {
	Name              string
	Version           string       // candidate (available) version
	InstalledVersion  string       // currently installed version ("" if not installed)
	Architecture      string
	Section           string
	Priority          string
	Origin            string
	Maintainer        string
	Homepage          string
	Summary           string       // one-line description
	Description       string       // full description
	PackageSize       int64        // download size in bytes
	InstalledSize     int64        // installed size in bytes
	IsInstalled       bool
	IsUpgradable      bool
	IsAutomatic       bool         // installed as automatic dependency
	Source            string       // source package name
	Depends           []Dependency // runtime dependencies
	Recommends        []Dependency
	Suggests          []Dependency
	Provides          []string
	DownloadURL       string
}


type Result struct {
	Output string
	Err    error
}


func run(args []string, stream bool) (*Result, error) {
	if stream {
		cmd := exec.Command("apt", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		err := cmd.Run()
		return &Result{Output: "", Err: err}, err
	}

	var out bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "apt", args...)
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return &Result{Output: out.String(), Err: err}, err
}

func runDpkg(args []string) (string, error) {
	var out bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "dpkg-query", args...)
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return out.String(), err
}


func Install(pkgs []string, stream bool) (*Result, error) {
	args := append([]string{"install", "-y"}, pkgs...)
	res, err := run(args, stream)
	if err != nil {
		return res, classifyError("install", res.Output, err)
	}
	return res, nil
}

func Remove(pkgs []string, stream bool) (*Result, error) {
	args := append([]string{"remove", "-y"}, pkgs...)
	res, err := run(args, stream)
	if err != nil {
		return res, classifyError("remove", res.Output, err)
	}
	return res, nil
}

func Update(stream bool) (*Result, error) {
	res, err := run([]string{"update"}, stream)
	if err != nil {
		return res, classifyError("update", res.Output, err)
	}
	return res, nil
}

func Upgrade(stream bool) (*Result, error) {
	res, err := run([]string{"upgrade", "-y"}, stream)
	if err != nil {
		return res, classifyError("upgrade", res.Output, err)
	}
	return res, nil
}

func Search(terms []string, stream bool) (*Result, error) {
	args := append([]string{"search"}, terms...)
	return run(args, stream)
}

func SearchContext(ctx context.Context, terms []string) (*Result, error) {
	args := append([]string{"search"}, terms...)
	var out bytes.Buffer
	cmd := exec.CommandContext(ctx, "apt", args...)
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return &Result{Output: out.String(), Err: err}, err
}


// Purge removes packages AND their configuration files (apt remove --purge).
func Purge(pkgs []string, stream bool) (*Result, error) {
	args := append([]string{"remove", "--purge", "-y"}, pkgs...)
	res, err := run(args, stream)
	if err != nil {
		return res, classifyError("purge", res.Output, err)
	}
	return res, nil
}

// AutoRemove removes packages that were automatically installed to satisfy
func AutoRemove(stream bool) (*Result, error) {
	res, err := run([]string{"autoremove", "-y"}, stream)
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
	return run([]string{"clean"}, false)
}

// ---------------------------------------------------------------------------
// Parsers
// ---------------------------------------------------------------------------

// parseDependencyString parses a dependency line like "libc6 (>= 2.17), libgcc-s1"
func parseDependencyString(raw string) []Dependency {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var deps []Dependency
	depRe := regexp.MustCompile(`([a-zA-Z0-9][a-zA-Z0-9.+\-]+)(?:\s*\(([<>=!]+)\s*([^)]+)\))?`)
	for _, part := range strings.Split(raw, ",") {
		// handle alternatives (|)
		alternatives := strings.Split(part, "|")
		for _, alt := range alternatives {
			matches := depRe.FindStringSubmatch(strings.TrimSpace(alt))
			if len(matches) >= 2 {
				d := Dependency{Name: matches[1]}
				if len(matches) >= 4 {
					d.Relation = matches[2]
					d.Version = matches[3]
				}
				deps = append(deps, d)
			}
		}
	}
	return deps
}

// parseShowOutput parses the output of `apt show <pkg>`.
func parseShowOutput(raw string) *Package {
	p := &Package{}
	lines := strings.Split(raw, "\n")
	var currentKey string
	var descLines []string

	for _, line := range lines {
		// continuation line (starts with space)
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			if currentKey == "Description" {
				descLines = append(descLines, strings.TrimSpace(line))
			}
			continue
		}

		idx := strings.Index(line, ":")
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		currentKey = key

		switch key {
		case "Package":
			p.Name = val
		case "Version":
			p.Version = val
		case "Architecture":
			p.Architecture = val
		case "Section":
			p.Section = val
		case "Priority":
			p.Priority = val
		case "Origin":
			p.Origin = val
		case "Maintainer":
			p.Maintainer = val
		case "Homepage":
			p.Homepage = val
		case "Source":
			p.Source = val
		case "Depends":
			p.Depends = parseDependencyString(val)
		case "Recommends":
			p.Recommends = parseDependencyString(val)
		case "Suggests":
			p.Suggests = parseDependencyString(val)
		case "Provides":
			for _, prov := range strings.Split(val, ",") {
				prov = strings.TrimSpace(prov)
				if prov != "" {
					p.Provides = append(p.Provides, prov)
				}
			}
		case "Size", "Download-Size":
			p.PackageSize = parseSize(val)
		case "Installed-Size":
			p.InstalledSize = parseSize(val)
		case "Description":
			p.Summary = val
			descLines = nil
		}
	}

	if len(descLines) > 0 {
		p.Description = strings.Join(descLines, "\n")
	} else if p.Summary != "" {
		p.Description = p.Summary
	}

	return p
}

// parseSize handles apt size strings like "1,234 kB" or "5.2 MB".
func parseSize(raw string) int64 {
	raw = strings.TrimSpace(raw)
	raw = strings.ReplaceAll(raw, ",", "")

	parts := strings.Fields(raw)
	if len(parts) == 0 {
		return 0
	}
	numStr := parts[0]
	n, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0
	}
	if len(parts) > 1 {
		switch strings.ToUpper(parts[1]) {
		case "KB":
			n *= 1024
		case "MB":
			n *= 1024 * 1024
		case "GB":
			n *= 1024 * 1024 * 1024
		case "B":
			// already bytes
		}
	}
	return int64(n)
}

func getInstalledVersion(name string) (string, error) {
	out, err := runDpkg([]string{"-W", "--showformat=${Version}", name})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// parseListOutput parses `apt list` output (used for --upgradable, --installed).
func parseListOutput(raw string, upgradable bool) []Package {
	var pkgs []Package
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Listing") {
			continue
		}
		// format: name/repo version arch [status]
		slash := strings.Index(line, "/")
		if slash < 0 {
			continue
		}
		name := line[:slash]
		rest := line[slash+1:]
		fields := strings.Fields(rest)
		if len(fields) < 3 {
			continue
		}
		p := Package{
			Name:         name,
			Version:      fields[1],
			Architecture: fields[2],
			IsUpgradable: upgradable,
			IsInstalled:  true,
		}
		// try extracting installed version from [upgradable from: x.y.z]
		if upgradable {
			if idx := strings.Index(line, "[upgradable from: "); idx >= 0 {
				tail := line[idx+len("[upgradable from: "):]
				if end := strings.IndexByte(tail, ']'); end > 0 {
					p.InstalledVersion = tail[:end]
				}
			}
		}
		pkgs = append(pkgs, p)
	}
	return pkgs
}

// FormatSize returns a human-readable file size.
func FormatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMG"[exp])
}
