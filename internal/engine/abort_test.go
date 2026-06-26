package engine

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/urjitbhatia/fruitbox/internal/runner"
)

func TestSuperviseAbortOnExitStopsOthers(t *testing.T) {
	proj := load(t, "basic") // db + web, no restart policy
	fake := &runner.Fake{}
	// web has exited; db is still running.
	fake.On("inspect basic-web-1", runner.Result{Stdout: `[{"status":"exited","exit_code":0}]`}, nil)
	fake.On("inspect basic-db-1", runner.Result{Stdout: `[{"status":"running"}]`}, nil)
	e := newTestEngine(fake)

	err := e.Supervise(context.Background(), proj, nil, SuperviseOptions{AbortOnExit: true})
	var exit ExitError
	if !errors.As(err, &exit) {
		t.Fatalf("expected ExitError, got %v", err)
	}
	// The still-running db must be stopped.
	if firstMatch(fake.CommandArgs(), "stop basic-db-1") == -1 {
		t.Errorf("abort-on-exit should stop other containers, calls: %v", fake.CommandArgs())
	}
}

func TestSuperviseAbortOnFailureIgnoresCleanExit(t *testing.T) {
	proj := load(t, "basic")
	fake := &runner.Fake{}
	// Both exit cleanly (code 0): --abort-on-container-failure must NOT abort.
	fake.On("inspect basic-web-1", runner.Result{Stdout: `[{"status":"exited","exit_code":0}]`}, nil)
	fake.On("inspect basic-db-1", runner.Result{Stdout: `[{"status":"exited","exit_code":0}]`}, nil)
	e := newTestEngine(fake)

	if err := e.Supervise(context.Background(), proj, nil, SuperviseOptions{AbortOnFailure: true}); err != nil {
		t.Fatalf("clean exits should not abort: %v", err)
	}
	if firstMatch(fake.CommandArgs(), "stop ") != -1 {
		t.Errorf("no stop expected on clean exit, calls: %v", fake.CommandArgs())
	}
}

func TestSuperviseAbortOnFailureAborts(t *testing.T) {
	proj := load(t, "basic")
	fake := &runner.Fake{}
	fake.On("inspect basic-web-1", runner.Result{Stdout: `[{"status":"exited","exit_code":3}]`}, nil)
	fake.On("inspect basic-db-1", runner.Result{Stdout: `[{"status":"running"}]`}, nil)
	e := newTestEngine(fake)

	err := e.Supervise(context.Background(), proj, nil, SuperviseOptions{AbortOnFailure: true})
	var exit ExitError
	if !errors.As(err, &exit) || exit.Code != 3 {
		t.Fatalf("expected ExitError{3}, got %v", err)
	}
}

func TestSuperviseExitCodeFrom(t *testing.T) {
	proj := load(t, "basic")
	fake := &runner.Fake{}
	// db exits with 7; web still running. --exit-code-from db should return 7
	// and stop web.
	fake.On("inspect basic-db-1", runner.Result{Stdout: `[{"status":"exited","exit_code":7}]`}, nil)
	fake.On("inspect basic-web-1", runner.Result{Stdout: `[{"status":"running"}]`}, nil)
	e := newTestEngine(fake)

	err := e.Supervise(context.Background(), proj, nil, SuperviseOptions{ExitCodeFrom: "db"})
	var exit ExitError
	if !errors.As(err, &exit) || exit.Code != 7 {
		t.Fatalf("expected ExitError{7}, got %v", err)
	}
	if firstMatch(fake.CommandArgs(), "stop basic-web-1") == -1 {
		t.Errorf("exit-code-from should stop other services, calls: %v", fake.CommandArgs())
	}
}

func TestExitErrorPropagates(t *testing.T) {
	// Sanity: ExitError unwraps via errors.As.
	var target ExitError
	if !errors.As(error(ExitError{Code: 42}), &target) || target.Code != 42 {
		t.Fatalf("ExitError should unwrap to code 42, got %+v", target)
	}
	_ = io.Discard
}
