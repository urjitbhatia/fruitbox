package engine

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/urjitbhatia/fruitbox/internal/runner"
)

func TestPullIncludeDeps(t *testing.T) {
	proj := load(t, "basic") // web depends_on db
	fake := &runner.Fake{}
	e := New(fake, io.Discard)
	// Pull only web, but --include-deps should also pull db.
	if err := e.Pull(context.Background(), proj, []string{"web"}, PullOptions{IncludeDeps: true}); err != nil {
		t.Fatalf("Pull: %v", err)
	}
	calls := fake.CommandArgs()
	if firstMatch(calls, "image pull nginx:1.27") == -1 || firstMatch(calls, "image pull postgres:16") == -1 {
		t.Errorf("--include-deps should pull web and db, calls: %v", calls)
	}
}

func TestPullIgnoreBuildableSkipsBuildServices(t *testing.T) {
	proj := load(t, "build") // worker has image+build; api builds only
	fake := &runner.Fake{}
	e := New(fake, io.Discard)
	if err := e.Pull(context.Background(), proj, nil, PullOptions{IgnoreBuildable: true}); err != nil {
		t.Fatalf("Pull: %v", err)
	}
	// worker has a build section, so --ignore-buildable skips it.
	if firstMatch(fake.CommandArgs(), "image pull myorg/worker:latest") != -1 {
		t.Errorf("--ignore-buildable should skip buildable services, calls: %v", fake.CommandArgs())
	}
}

func TestPullIgnoreFailuresContinues(t *testing.T) {
	proj := load(t, "basic")
	fake := &runner.Fake{}
	fake.On("image pull nginx:1.27", runner.Result{ExitCode: 1}, errors.New("boom"))
	e := New(fake, io.Discard)
	// Without ignore, the failure surfaces.
	if err := e.Pull(context.Background(), proj, nil, PullOptions{}); err == nil {
		t.Error("expected pull failure to surface")
	}
	// With ignore, it continues and pulls the other image.
	fake2 := &runner.Fake{}
	fake2.On("image pull nginx:1.27", runner.Result{ExitCode: 1}, errors.New("boom"))
	e2 := New(fake2, io.Discard)
	if err := e2.Pull(context.Background(), proj, nil, PullOptions{IgnoreFailures: true}); err != nil {
		t.Fatalf("ignore-pull-failures should not error: %v", err)
	}
	if firstMatch(fake2.CommandArgs(), "image pull postgres:16") == -1 {
		t.Errorf("should continue to next image after failure, calls: %v", fake2.CommandArgs())
	}
}

func TestPushIncludeDeps(t *testing.T) {
	proj := load(t, "basic")
	fake := &runner.Fake{}
	e := New(fake, io.Discard)
	if err := e.Push(context.Background(), proj, []string{"web"}, PushOptions{IncludeDeps: true}); err != nil {
		t.Fatalf("Push: %v", err)
	}
	calls := fake.CommandArgs()
	if firstMatch(calls, "image push nginx:1.27") == -1 || firstMatch(calls, "image push postgres:16") == -1 {
		t.Errorf("--include-deps should push web and db, calls: %v", calls)
	}
}

func TestPullPolicyNeverSkips(t *testing.T) {
	proj := load(t, "basic")
	fake := &runner.Fake{}
	e := New(fake, io.Discard)
	if err := e.Pull(context.Background(), proj, nil, PullOptions{Policy: "never"}); err != nil {
		t.Fatalf("Pull: %v", err)
	}
	if firstMatch(fake.CommandArgs(), "image pull") != -1 {
		t.Errorf("--policy never should skip pulling, calls: %v", fake.CommandArgs())
	}
}
