package engine

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/urjitbhatia/fruitbox/internal/runner"
	"github.com/urjitbhatia/fruitbox/internal/translate"
)

func TestCreateUsesCreateVerbAndDoesNotStart(t *testing.T) {
	proj := load(t, "basic")
	fake := &runner.Fake{}
	e := New(fake, io.Discard)

	if err := e.Create(context.Background(), proj, CreateOptions{}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	calls := fake.CommandArgs()
	// Containers created with the `create` verb, in dependency order.
	if firstMatch(calls, "create --name basic-db-1") == -1 {
		t.Errorf("db should be created, calls:\n%s", strings.Join(calls, "\n"))
	}
	if firstMatch(calls, "run --name basic-web-1") != -1 {
		t.Errorf("create must not `run` containers")
	}
}

func TestRmForceDeletes(t *testing.T) {
	proj := load(t, "basic")
	fake := &runner.Fake{}
	e := New(fake, io.Discard)
	if err := e.Rm(context.Background(), proj, []string{"web"}, RmOptions{Force: true}); err != nil {
		t.Fatalf("Rm: %v", err)
	}
	if firstMatch(fake.CommandArgs(), "delete --force basic-web-1") == -1 {
		t.Errorf("rm -f should force-delete, calls: %v", fake.CommandArgs())
	}
}

func TestPushDedupesImages(t *testing.T) {
	proj := load(t, "basic")
	fake := &runner.Fake{}
	e := New(fake, io.Discard)
	if err := e.Push(context.Background(), proj, nil, PushOptions{}); err != nil {
		t.Fatalf("Push: %v", err)
	}
	calls := fake.CommandArgs()
	if firstMatch(calls, "image push nginx:1.27") == -1 || firstMatch(calls, "image push postgres:16") == -1 {
		t.Errorf("expected pushes for both images, calls: %v", calls)
	}
}

func TestScaleStartsReplicasAndTrimsSurplus(t *testing.T) {
	proj := load(t, "basic")
	fake := &runner.Fake{}
	e := New(fake, io.Discard)
	if err := e.Scale(context.Background(), proj, map[string]int{"web": 2}); err != nil {
		t.Fatalf("Scale: %v", err)
	}
	calls := fake.CommandArgs()
	if firstMatch(calls, "--name basic-web-1") == -1 || firstMatch(calls, "--name basic-web-2") == -1 {
		t.Errorf("scale should start 2 web replicas, calls: %v", calls)
	}
	// Surplus replica 3 should be trimmed.
	if firstMatch(calls, "delete basic-web-3") == -1 {
		t.Errorf("scale should attempt to remove surplus replica 3, calls: %v", calls)
	}
}

func TestAttachInteractive(t *testing.T) {
	proj := load(t, "basic")
	fake := &runner.Fake{}
	e := New(fake, io.Discard)
	if err := e.Attach(context.Background(), proj, "web", 1); err != nil {
		t.Fatalf("Attach: %v", err)
	}
	if firstMatch(fake.CommandArgs(), "start --attach --interactive basic-web-1") == -1 {
		t.Errorf("attach should start --attach, calls: %v", fake.CommandArgs())
	}
}

func TestCreateNoBuildSkipsBuild(t *testing.T) {
	proj := load(t, "build")
	fake := &runner.Fake{}
	e := New(fake, io.Discard)
	if err := e.Create(context.Background(), proj, CreateOptions{NoBuild: true}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if firstMatch(fake.CommandArgs(), "build --tag") != -1 {
		t.Errorf("--no-build should skip building, calls: %v", fake.CommandArgs())
	}
}

func TestCreateBuildsByDefault(t *testing.T) {
	proj := load(t, "build")
	fake := &runner.Fake{}
	e := New(fake, io.Discard)
	if err := e.Create(context.Background(), proj, CreateOptions{}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if firstMatch(fake.CommandArgs(), "build --tag build_api") == -1 {
		t.Errorf("create should build buildable services by default, calls: %v", fake.CommandArgs())
	}
}

func TestCreateRemoveOrphans(t *testing.T) {
	proj := load(t, "basic")
	fake := &runner.Fake{}
	fake.On("list --all --format json", runner.Result{Stdout: `[
		{"name":"basic-ghost-1","labels":{"com.docker.compose.project":"basic","com.docker.compose.service":"ghost"}}
	]`}, nil)
	e := New(fake, io.Discard)
	if err := e.Create(context.Background(), proj, CreateOptions{RemoveOrphans: true}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if firstMatch(fake.CommandArgs(), "delete basic-ghost-1") == -1 {
		t.Errorf("create --remove-orphans should remove orphan, calls: %v", fake.CommandArgs())
	}
}

func TestRmVolumes(t *testing.T) {
	proj := load(t, "basic")
	fake := &runner.Fake{}
	e := New(fake, io.Discard)
	if err := e.Rm(context.Background(), proj, []string{"web"}, RmOptions{Force: true, Volumes: true}); err != nil {
		t.Fatalf("Rm: %v", err)
	}
	if firstMatch(fake.CommandArgs(), "delete --force --volumes basic-web-1") == -1 {
		t.Errorf("rm -v should pass --volumes, calls: %v", fake.CommandArgs())
	}
}

func TestCreateForceRecreate(t *testing.T) {
	proj := load(t, "basic")
	web, _ := proj.GetService("web")
	fake := &runner.Fake{}
	fake.On("inspect basic-web-1", runner.Result{Stdout: inspectWithHash(translate.ServiceConfigHash(web))}, nil)
	fake.On("inspect basic-db-1", runner.Result{Stdout: `[{"status":"stopped"}]`}, nil)
	e := New(fake, io.Discard)
	if err := e.Create(context.Background(), proj, CreateOptions{ForceRecreate: true}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if firstMatch(fake.CommandArgs(), "delete basic-web-1") == -1 {
		t.Errorf("create --force-recreate should recreate existing, calls: %v", fake.CommandArgs())
	}
}

func TestCreateNoRecreateKeepsExisting(t *testing.T) {
	proj := load(t, "basic")
	fake := &runner.Fake{}
	fake.On("inspect basic-web-1", runner.Result{Stdout: inspectWithHash("stale")}, nil)
	fake.On("inspect basic-db-1", runner.Result{Stdout: inspectWithHash("stale")}, nil)
	e := New(fake, io.Discard)
	if err := e.Create(context.Background(), proj, CreateOptions{NoRecreate: true}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if firstMatch(fake.CommandArgs(), "delete basic-web-1") != -1 {
		t.Errorf("create --no-recreate must NOT recreate, calls: %v", fake.CommandArgs())
	}
}
