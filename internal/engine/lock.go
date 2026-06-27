package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// LockProject acquires an exclusive, advisory, cross-process lock for a project
// so that two mutating fruitbox commands (e.g. two `up`s, or `up` and `down`)
// cannot race into half-created/half-removed state. It returns a release
// function, or an error if another process already holds the lock.
//
// The lock must be acquired ONCE per command at the CLI boundary — never per
// engine method — because the orchestrator calls other engine methods
// internally and a second flock from the same process (different fd) would
// deadlock.
func (e *Engine) LockProject(project string) (func(), error) {
	dir := filepath.Join(e.stateDir(), "locks")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create lock dir: %w", err)
	}
	path := filepath.Join(dir, sanitizeLockName(project)+".lock")

	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open lock file: %w", err)
	}

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		holder := readLockHolder(path)
		_ = f.Close()
		return nil, fmt.Errorf("project %q is locked by another fruitbox process%s; "+
			"wait for it to finish or run with a different -p", project, holder)
	}

	// Record our pid for diagnostics (best effort).
	_ = f.Truncate(0)
	_, _ = f.WriteAt([]byte(strconv.Itoa(os.Getpid())), 0)

	released := false
	return func() {
		if released {
			return
		}
		released = true
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
	}, nil
}

// readLockHolder returns a " (pid N)" hint if the lock file records a pid.
func readLockHolder(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	if pid := strings.TrimSpace(string(b)); pid != "" {
		return " (pid " + pid + ")"
	}
	return ""
}

// sanitizeLockName makes a project name safe for a filename.
func sanitizeLockName(project string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			return r
		default:
			return '_'
		}
	}, project)
}
