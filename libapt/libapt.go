package libapt

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"time"
)

// Result contains captured output and error from apt calls.
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

func Install(pkgs []string, stream bool) (*Result, error) {
	args := append([]string{"install", "-y"}, pkgs...)
	return run(args, stream)
}

func Remove(pkgs []string, stream bool) (*Result, error) {
	args := append([]string{"remove", "-y"}, pkgs...)
	return run(args, stream)
}

func Update(stream bool) (*Result, error) {
	return run([]string{"update"}, stream)
}

func Upgrade(stream bool) (*Result, error) {
	return run([]string{"upgrade", "-y"}, stream)
}

func Search(terms []string, stream bool) (*Result, error) {
	args := append([]string{"search"}, terms...)
	return run(args, stream)
}
