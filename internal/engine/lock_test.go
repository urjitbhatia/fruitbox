package engine

import (
	"io"
	"strings"
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
