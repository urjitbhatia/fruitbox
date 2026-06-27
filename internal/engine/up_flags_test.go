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

func TestUpSkipsExistingNetwork(t *testing.T) {
	proj := load(t, "basic") // declares network basic_net
	fake := &runner.Fake{}
	// The network already exists (inspect returns a payload).
	fake.On("network inspect basic_net", runner.Result{Stdout: `{"name":"basic_net"}`}, nil)
	e := New(fake, io.Discard)
	if err := e.Up(context.Background(), proj, UpOptions{Detach: true}); err != nil {
		t.Fatalf("Up: %v", err)
	}
	// Idempotent: must NOT attempt to create the existing network.
	if firstMatch(fake.CommandArgs(), "network create") != -1 {
		t.Errorf("up must skip creating an existing network, calls: %v", fake.CommandArgs())
	}
}

func TestUpCreatesMissingNetwork(t *testing.T) {
	proj := load(t, "basic")
	fake := &runner.Fake{} // no inspect mock -> network treated as absent
	e := New(fake, io.Discard)
	if err := e.Up(context.Background(), proj, UpOptions{Detach: true}); err != nil {
		t.Fatalf("Up: %v", err)
	}
	if firstMatch(fake.CommandArgs(), "network create") == -1 {
		t.Errorf("up should create a missing network, calls: %v", fake.CommandArgs())
	}
}

func TestUpForegroundStreamsLogs(t *testing.T) {
	proj := load(t, "basic")
	fake := &runner.Fake{}
	// Containers are already stopped so Supervise returns immediately.
	fake.On("inspect basic-web-1", runner.Result{Stdout: `[{"status":"stopped"}]`}, nil)
	fake.On("inspect basic-db-1", runner.Result{Stdout: `[{"status":"stopped"}]`}, nil)
	// Log output per container.
	fake.On("logs --follow basic-web-1", runner.Result{Stdout: "web hello\n"}, nil)
	fake.On("logs --follow basic-db-1", runner.Result{Stdout: "db hello\n"}, nil)
	var out strings.Builder
	e := newTestEngine(fake)
	e.Out = &out

	// Foreground (Detach:false) -> streams logs then supervises to completion.
	if err := e.Up(context.Background(), proj, UpOptions{NoColor: true}); err != nil {
		t.Fatalf("Up: %v", err)
	}
	s := out.String()
	if !strings.Contains(s, "web | web hello") || !strings.Contains(s, "db  | db hello") {
		t.Errorf("foreground up should stream prefixed logs, got:\n%s", s)
	}
}

func TestUpForegroundNoAttachExcludes(t *testing.T) {
	proj := load(t, "basic")
	fake := &runner.Fake{}
	fake.On("inspect basic-web-1", runner.Result{Stdout: `[{"status":"stopped"}]`}, nil)
	fake.On("inspect basic-db-1", runner.Result{Stdout: `[{"status":"stopped"}]`}, nil)
	fake.On("logs --follow basic-web-1", runner.Result{Stdout: "web hello\n"}, nil)
	fake.On("logs --follow basic-db-1", runner.Result{Stdout: "db hello\n"}, nil)
	var out strings.Builder
	e := newTestEngine(fake)
	e.Out = &out

	if err := e.Up(context.Background(), proj, UpOptions{NoColor: true, NoAttach: []string{"db"}}); err != nil {
		t.Fatalf("Up: %v", err)
	}
	s := out.String()
	if !strings.Contains(s, "web hello") {
		t.Errorf("web logs should stream:\n%s", s)
	}
	if strings.Contains(s, "db hello") {
		t.Errorf("--no-attach db should suppress db logs:\n%s", s)
	}
}

func TestUpForegroundGracefulStopOnCancel(t *testing.T) {
	proj := load(t, "basic")
	fake := &runner.Fake{}
	// Simulate Ctrl-C: the context is already cancelled when supervising.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	e := New(fake, io.Discard)

	// Foreground up: should start, see the cancellation, and gracefully stop
	// the started containers, exiting cleanly (nil).
	if err := e.Up(ctx, proj, UpOptions{}); err != nil {
		t.Fatalf("Up should exit cleanly on Ctrl-C, got %v", err)
	}
	calls := fake.CommandArgs()
	if firstMatch(calls, "stop basic-web-1") == -1 || firstMatch(calls, "stop basic-db-1") == -1 {
		t.Errorf("Ctrl-C should gracefully stop started containers, calls:\n%v", calls)
	}
}
