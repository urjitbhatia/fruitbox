// Package runner abstracts execution of Apple's `container` CLI so that the
// orchestration engine can be unit-tested without the binary present. The real
// implementation shells out; tests substitute a fake.
package runner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
)

// Result captures the outcome of a single container CLI invocation.
type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// Runner executes container CLI commands.
type Runner interface {
	// Run executes `container <args...>`, capturing stdout/stderr. A non-zero
	// exit code is returned in Result and also surfaced as an error.
	Run(ctx context.Context, args ...string) (Result, error)
	// RunInteractive executes `container <args...>` wired directly to the
	// process stdio (for attached/interactive containers and log following).
	RunInteractive(ctx context.Context, args ...string) error
	// RunWithOutput executes `container <args...>`, streaming stdout and stderr
	// to the given writers (no stdin). It is used for concurrent log
	// multiplexing, where each container's output is prefixed.
	RunWithOutput(ctx context.Context, stdout, stderr io.Writer, args ...string) error
}

// Exec is the production Runner that invokes the container binary.
type Exec struct {
	// Binary is the executable to invoke; defaults to "container".
	Binary string
	// Stdout/Stderr are used by RunInteractive; default to the process streams.
	Stdout *os.File
	Stderr *os.File
	Stdin  *os.File
}

// NewExec returns an Exec runner using the given binary name (or "container"
// when empty) wired to the process standard streams.
func NewExec(binary string) *Exec {
	if binary == "" {
		binary = "container"
	}
	return &Exec{Binary: binary, Stdout: os.Stdout, Stderr: os.Stderr, Stdin: os.Stdin}
}

// Run implements Runner.
func (e *Exec) Run(ctx context.Context, args ...string) (Result, error) {
	cmd := exec.CommandContext(ctx, e.Binary, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	res := Result{Stdout: stdout.String(), Stderr: stderr.String()}
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			res.ExitCode = exitErr.ExitCode()
			return res, fmt.Errorf("%s %v: exit %d: %s", e.Binary, args, res.ExitCode, stderr.String())
		}
		return res, fmt.Errorf("%s %v: %w", e.Binary, args, err)
	}
	return res, nil
}

// RunInteractive implements Runner. Standard output and error are wired to the
// process streams. Standard input is attached only when it is a terminal: the
// container runtime rejects some non-terminal stdin devices (ENODEV), and a
// non-interactive invocation (script/pipe/CI) doesn't need it.
func (e *Exec) RunInteractive(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, e.Binary, args...)
	cmd.Stdout = e.Stdout
	cmd.Stderr = e.Stderr
	if isTerminal(e.Stdin) {
		cmd.Stdin = e.Stdin
	}
	return cmd.Run()
}

// RunWithOutput implements Runner.
func (e *Exec) RunWithOutput(ctx context.Context, stdout, stderr io.Writer, args ...string) error {
	cmd := exec.CommandContext(ctx, e.Binary, args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

// isTerminal reports whether f is an interactive character device.
func isTerminal(f *os.File) bool {
	if f == nil {
		return false
	}
	fi, err := f.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}
