package libapt

import (
    "bytes"
    "context"
    "os"
    "os/exec"
    "time"
)

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

// SearchContext runs `apt search` with a provided context (timeout/cancel).
func SearchContext(ctx context.Context, terms []string) (*Result, error) {
    args := append([]string{"search"}, terms...)
    var out bytes.Buffer
    cmd := exec.CommandContext(ctx, "apt", args...)
    cmd.Stdout = &out
    cmd.Stderr = &out
    err := cmd.Run()
    return &Result{Output: out.String(), Err: err}, err
}
