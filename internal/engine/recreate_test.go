package engine

import (
	"context"
	"io"
	"testing"

	"github.com/urjitbhatia/fruitbox/internal/runner"
	"github.com/urjitbhatia/fruitbox/internal/translate"
)

// runningWith returns an inspect payload carrying the given config-hash label.
func inspectWithHash(hash string) string {
	return `[{"status":"running","labels":{"com.docker.compose.config-hash":"` + hash + `"}}]`
}

func TestUpReusesUpToDateContainer(t *testing.T) {
	proj := load(t, "basic")
	web, _ := proj.GetService("web")
	hash := translate.ServiceConfigHash(web)

	fake := &runner.Fake{}
	// Both containers already exist with the current config hash.
	fake.On("inspect basic-web-1", runner.Result{Stdout: inspectWithHash(hash)}, nil)
	db, _ := proj.GetService("db")
	fake.On("inspect basic-db-1", runner.Result{Stdout: inspectWithHash(translate.ServiceConfigHash(db))}, nil)
	e := New(fake, io.Discard)

	if err := e.Up(context.Background(), proj, UpOptions{Detach: true}); err != nil {
		t.Fatalf("Up: %v", err)
	}
	calls := fake.CommandArgs()
	// Up-to-date: just `start`, no fresh `run`.
	if firstMatch(calls, "start basic-web-1") == -1 {
		t.Errorf("up-to-date container should be started, calls: %v", calls)
	}
	if firstMatch(calls, "run --name basic-web-1") != -1 {
		t.Errorf("up-to-date container must NOT be recreated, calls: %v", calls)
	}
}

func TestUpRecreatesOnConfigChange(t *testing.T) {
	proj := load(t, "basic")
	fake := &runner.Fake{}
	// Existing container carries a stale hash -> recreate.
	fake.On("inspect basic-web-1", runner.Result{Stdout: inspectWithHash("stalehash")}, nil)
	fake.On("inspect basic-db-1", runner.Result{Stdout: inspectWithHash("stalehash")}, nil)
	e := New(fake, io.Discard)

	if err := e.Up(context.Background(), proj, UpOptions{Detach: true}); err != nil {
		t.Fatalf("Up: %v", err)
	}
	calls := fake.CommandArgs()
	posStop := firstMatch(calls, "stop basic-web-1")
	posDelete := firstMatch(calls, "delete basic-web-1")
	posRun := firstMatch(calls, "run --name basic-web-1")
	if posStop == -1 || posDelete == -1 || posRun == -1 {
		t.Fatalf("recreate should stop+delete+run, calls: %v", calls)
	}
	if !(posStop < posDelete && posDelete < posRun) {
		t.Errorf("recreate order wrong: stop@%d delete@%d run@%d", posStop, posDelete, posRun)
	}
}

func TestUpForceRecreate(t *testing.T) {
	proj := load(t, "basic")
	web, _ := proj.GetService("web")
	fake := &runner.Fake{}
	// Container exists with the CURRENT hash, but --force-recreate ignores it.
	fake.On("inspect basic-web-1", runner.Result{Stdout: inspectWithHash(translate.ServiceConfigHash(web))}, nil)
	e := New(fake, io.Discard)

	if err := e.Up(context.Background(), proj, UpOptions{Detach: true, ForceRecreate: true}); err != nil {
		t.Fatalf("Up: %v", err)
	}
	if firstMatch(fake.CommandArgs(), "delete basic-web-1") == -1 {
		t.Errorf("--force-recreate should recreate even unchanged containers, calls: %v", fake.CommandArgs())
	}
}

func TestUpNoRecreate(t *testing.T) {
	proj := load(t, "basic")
	fake := &runner.Fake{}
	// Container exists with a STALE hash, but --no-recreate keeps it.
	fake.On("inspect basic-web-1", runner.Result{Stdout: inspectWithHash("stalehash")}, nil)
	fake.On("inspect basic-db-1", runner.Result{Stdout: inspectWithHash("stalehash")}, nil)
	e := New(fake, io.Discard)

	if err := e.Up(context.Background(), proj, UpOptions{Detach: true, NoRecreate: true}); err != nil {
		t.Fatalf("Up: %v", err)
	}
	calls := fake.CommandArgs()
	if firstMatch(calls, "delete basic-web-1") != -1 {
		t.Errorf("--no-recreate must NOT recreate, calls: %v", calls)
	}
	if firstMatch(calls, "start basic-web-1") == -1 {
		t.Errorf("--no-recreate should still start the existing container, calls: %v", calls)
	}
}

func TestRunArgsCarryConfigHashOnCreate(t *testing.T) {
	proj := load(t, "basic")
	fake := &runner.Fake{}
	e := New(fake, io.Discard)
	// Fresh up (no existing containers): run args should carry the hash label.
	if err := e.Up(context.Background(), proj, UpOptions{Detach: true}); err != nil {
		t.Fatalf("Up: %v", err)
	}
	if firstMatch(fake.CommandArgs(), "com.docker.compose.config-hash=") == -1 {
		t.Errorf("fresh containers should be stamped with a config-hash label, calls: %v", fake.CommandArgs())
	}
}

func TestUpRecreateTimeout(t *testing.T) {
	proj := load(t, "basic")
	fake := &runner.Fake{}
	fake.On("inspect basic-web-1", runner.Result{Stdout: inspectWithHash("stale")}, nil)
	fake.On("inspect basic-db-1", runner.Result{Stdout: inspectWithHash("stale")}, nil)
	e := New(fake, io.Discard)
	nine := 9
	if err := e.Up(context.Background(), proj, UpOptions{Detach: true, Timeout: &nine}); err != nil {
		t.Fatalf("Up: %v", err)
	}
	if firstMatch(fake.CommandArgs(), "stop --time 9 basic-web-1") == -1 {
		t.Errorf("up --timeout should set recreate stop --time, calls: %v", fake.CommandArgs())
	}
}
