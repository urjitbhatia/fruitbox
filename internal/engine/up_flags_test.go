package engine

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/urjitbhatia/fruitbox/internal/runner"
)

func TestUpNoStartCreatesWithoutStarting(t *testing.T) {
	proj := load(t, "basic")
	fake := &runner.Fake{}
	e := New(fake, io.Discard)
	if err := e.Up(context.Background(), proj, UpOptions{Detach: true, NoStart: true}); err != nil {
		t.Fatalf("Up: %v", err)
	}
	calls := fake.CommandArgs()
	if firstMatch(calls, "create --name basic-db-1") == -1 {
		t.Errorf("--no-start should create containers, calls:\n%s", strings.Join(calls, "\n"))
	}
	if firstMatch(calls, "run --name basic-db-1") != -1 {
		t.Errorf("--no-start must not run containers")
	}
}

func TestUpPullAlwaysPullsBeforeStart(t *testing.T) {
	proj := load(t, "basic")
	fake := &runner.Fake{}
	e := New(fake, io.Discard)
	if err := e.Up(context.Background(), proj, UpOptions{Detach: true, Pull: "always"}); err != nil {
		t.Fatalf("Up: %v", err)
	}
	calls := fake.CommandArgs()
	posPull := firstMatch(calls, "image pull nginx:1.27")
	posRun := firstMatch(calls, "run --name basic-web-1")
	if posPull == -1 {
		t.Fatalf("--pull always should pull images, calls:\n%s", strings.Join(calls, "\n"))
	}
	if posPull > posRun {
		t.Errorf("pull should happen before run: pull@%d run@%d", posPull, posRun)
	}
}

func TestUpWaitWaitsForHealthchecks(t *testing.T) {
	proj := load(t, "health")
	fake := &runner.Fake{}
	// db healthcheck passes immediately; migrate inspect = completed.
	fake.On("exec health-db-1 pg_isready", runner.Result{ExitCode: 0}, nil)
	fake.On("inspect health-migrate-1", runner.Result{Stdout: `[{"status":"stopped","exit_code":0}]`}, nil)
	e := newTestEngine(fake)

	if err := e.Up(context.Background(), proj, UpOptions{Detach: true, Wait: true}); err != nil {
		t.Fatalf("Up: %v", err)
	}
	// With --wait, the db healthcheck is probed after services start too.
	if fake.CountMatching("exec health-db-1 pg_isready") == 0 {
		t.Errorf("--wait should probe healthchecks, calls: %v", fake.CommandArgs())
	}
}

func TestUpServiceSelectionIncludesDeps(t *testing.T) {
	proj := load(t, "basic") // web depends_on db
	fake := &runner.Fake{}
	e := New(fake, io.Discard)
	// up only "web" -> should also start its dependency db.
	if err := e.Up(context.Background(), proj, UpOptions{Detach: true, Services: []string{"web"}}); err != nil {
		t.Fatalf("Up: %v", err)
	}
	calls := fake.CommandArgs()
	if firstMatch(calls, "run --name basic-db-1") == -1 {
		t.Errorf("up web should also start dependency db, calls: %v", calls)
	}
	if firstMatch(calls, "run --name basic-web-1") == -1 {
		t.Errorf("up web should start web, calls: %v", calls)
	}
}

func TestUpNoDepsStartsOnlyNamed(t *testing.T) {
	proj := load(t, "basic")
	fake := &runner.Fake{}
	e := New(fake, io.Discard)
	if err := e.Up(context.Background(), proj, UpOptions{Detach: true, Services: []string{"web"}, NoDeps: true}); err != nil {
		t.Fatalf("Up: %v", err)
	}
	calls := fake.CommandArgs()
	if firstMatch(calls, "run --name basic-web-1") == -1 {
		t.Errorf("up --no-deps web should start web, calls: %v", calls)
	}
	if firstMatch(calls, "run --name basic-db-1") != -1 {
		t.Errorf("up --no-deps must NOT start dependency db, calls: %v", calls)
	}
}
