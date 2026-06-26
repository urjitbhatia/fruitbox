package engine

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/urjitbhatia/fruitbox/internal/runner"
)

func TestBuildRunsBuildForBuildServicesOnly(t *testing.T) {
	proj := load(t, "build")
	fake := &runner.Fake{}
	e := New(fake, io.Discard)

	if err := e.Build(context.Background(), proj, nil); err != nil {
		t.Fatalf("Build: %v", err)
	}
	calls := fake.CommandArgs()
	// api has a build section; worker has image+build (so it also builds).
	if firstMatch(calls, "build --tag build_api") == -1 {
		t.Errorf("expected api to build, calls:\n%s", strings.Join(calls, "\n"))
	}
	// A pure image service must not trigger a build.
	for _, c := range calls {
		if strings.Contains(c, "build --tag nginx") {
			t.Errorf("image-only service should not build: %s", c)
		}
	}
}

func TestUpBuildsBeforeStarting(t *testing.T) {
	proj := load(t, "build")
	fake := &runner.Fake{}
	e := New(fake, io.Discard)

	if err := e.Up(context.Background(), proj, UpOptions{Detach: true}); err != nil {
		t.Fatalf("Up: %v", err)
	}
	calls := fake.CommandArgs()
	posBuild := firstMatch(calls, "build --tag build_api")
	posRun := firstMatch(calls, "--name build-api-1")
	if posBuild == -1 || posRun == -1 {
		t.Fatalf("missing build or run call:\n%s", strings.Join(calls, "\n"))
	}
	if posBuild > posRun {
		t.Errorf("build must precede run: build@%d run@%d", posBuild, posRun)
	}
}

func TestStopReverseOrder(t *testing.T) {
	proj := load(t, "basic")
	fake := &runner.Fake{}
	e := New(fake, io.Discard)
	if err := e.Stop(context.Background(), proj, nil, nil); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	calls := fake.CommandArgs()
	if firstMatch(calls, "stop basic-web-1") > firstMatch(calls, "stop basic-db-1") {
		t.Errorf("web should stop before db, calls: %v", calls)
	}
}

func TestPullDedupesImages(t *testing.T) {
	proj := load(t, "basic")
	fake := &runner.Fake{}
	e := New(fake, io.Discard)
	if err := e.Pull(context.Background(), proj, nil); err != nil {
		t.Fatalf("Pull: %v", err)
	}
	calls := fake.CommandArgs()
	if firstMatch(calls, "image pull nginx:1.27") == -1 {
		t.Errorf("expected nginx pull, got %v", calls)
	}
	if firstMatch(calls, "image pull postgres:16") == -1 {
		t.Errorf("expected postgres pull, got %v", calls)
	}
}

func TestKillPassesSignal(t *testing.T) {
	proj := load(t, "basic")
	fake := &runner.Fake{}
	e := New(fake, io.Discard)
	if err := e.Kill(context.Background(), proj, []string{"web"}, "SIGTERM", false); err != nil {
		t.Fatalf("Kill: %v", err)
	}
	calls := fake.CommandArgs()
	if firstMatch(calls, "kill --signal SIGTERM basic-web-1") == -1 {
		t.Errorf("expected signalled kill, got %v", calls)
	}
}
