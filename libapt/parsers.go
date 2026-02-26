package libapt

import (
    "fmt"
    "regexp"
    "strconv"
    "strings"
)

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
