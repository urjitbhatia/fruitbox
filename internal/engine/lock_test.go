package engine

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/urjitbhatia/fruitbox/internal/runner"
)

func TestLockProjectIsExclusive(t *testing.T) {
	e := New(&runner.Fake{}, io.Discard)
	e.StateDir = t.TempDir()

	release1, err := e.LockProject("proj")
	if err != nil {
		t.Fatalf("first lock failed: %v", err)
	}

	// A second acquisition of the same project must fail while held.
	if _, err := e.LockProject("proj"); err == nil {
		t.Fatal("second lock should have failed while the first is held")
	} else if !strings.Contains(err.Error(), "locked") {
		t.Errorf("unexpected error: %v", err)
	}

	// A different project locks independently.
	releaseOther, err := e.LockProject("other")
	if err != nil {
		t.Fatalf("different project should lock: %v", err)
	}
	releaseOther()

	// After releasing, the project can be re-locked.
	release1()
	release2, err := e.LockProject("proj")
	if err != nil {
		t.Fatalf("re-lock after release failed: %v", err)
	}
	release2()
}

func TestLockReleaseIsIdempotent(t *testing.T) {
	e := New(&runner.Fake{}, io.Discard)
	e.StateDir = t.TempDir()
	release, err := e.LockProject("p")
	if err != nil {
		t.Fatal(err)
	}
	release()
	release() // must not panic or double-unlock
	// Lock is free again.
	r2, err := e.LockProject("p")
	if err != nil {
		t.Fatalf("expected free lock, got %v", err)
	}
	r2()
}

func TestLockAutoReleasedOnCrash(t *testing.T) {
	e := New(&runner.Fake{}, io.Discard)
	e.StateDir = t.TempDir()

	// Simulate a holder that "crashes": open + flock the lock file directly,
	// then close the fd without an explicit unlock (as the kernel does on exit).
	path := filepath.Join(e.StateDir, "locks", "p.lock")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		t.Fatal(err)
	}
	// Held now: LockProject must fail.
	if _, err := e.LockProject("p"); err == nil {
		t.Fatal("lock should be held")
	}
	// "Crash": close the fd. The kernel releases the flock.
	f.Close()
	// Recovery: the lock is acquirable again despite the file lingering.
	release, err := e.LockProject("p")
	if err != nil {
		t.Fatalf("lock should be recoverable after crash, got %v", err)
	}
	release()
}
