package libapt

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

// Result contains captured output and error from apt calls.
type Result struct {
    Output string
    Err    error
}
