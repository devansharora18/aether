package libapt

import (
    "fmt"
    "strings"
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
