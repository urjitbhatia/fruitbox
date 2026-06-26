package engine

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/urjitbhatia/fruitbox/internal/runner"
)

func TestRunOneOffCommandOverrideAndName(t *testing.T) {
	proj := load(t, "basic")
	fake := &runner.Fake{}
	e := New(fake, io.Discard)

	err := e.RunOneOff(context.Background(), proj, "web", RunOneOffOptions{
		Command: []string{"echo", "hi"},
		NoDeps:  true,
		Remove:  true,
	})
	if err != nil {
		t.Fatalf("RunOneOff: %v", err)
	}
	calls := fake.CommandArgs()
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d: %v", len(calls), calls)
	}
	c := calls[0]
	if !strings.Contains(c, "--name basic-web-run") {
		t.Errorf("one-off should use run name, got: %s", c)
	}
	if !strings.Contains(c, "--rm") {
		t.Errorf("one-off should pass --rm, got: %s", c)
	}
	if !strings.Contains(c, "com.docker.compose.oneoff=True") {
		t.Errorf("one-off should be labelled oneoff=True, got: %s", c)
	}
	if !strings.HasSuffix(c, "nginx:1.27 echo hi") {
		t.Errorf("command override should follow image, got: %s", c)
	}
}

func TestRunOneOffStartsDependencies(t *testing.T) {
	proj := load(t, "basic") // web depends_on db
	fake := &runner.Fake{}
	e := New(fake, io.Discard)

	err := e.RunOneOff(context.Background(), proj, "web", RunOneOffOptions{Remove: true})
	if err != nil {
		t.Fatalf("RunOneOff: %v", err)
	}
	calls := fake.CommandArgs()
	posDB := firstMatch(calls, "--name basic-db-1")
	posWeb := firstMatch(calls, "--name basic-web-run")
	if posDB == -1 {
		t.Fatalf("dependency db should start, calls:\n%s", strings.Join(calls, "\n"))
	}
	if posDB > posWeb {
		t.Errorf("db dependency should start before one-off web: db@%d web@%d", posDB, posWeb)
	}
}

func TestUpScaleOverride(t *testing.T) {
	proj := load(t, "basic")
	fake := &runner.Fake{}
	e := New(fake, io.Discard)

	err := e.Up(context.Background(), proj, UpOptions{Detach: true, Scale: map[string]int{"web": 3}})
	if err != nil {
		t.Fatalf("Up: %v", err)
	}
	if got := fake.CountMatching("--name basic-web-"); got != 3 {
		t.Errorf("expected 3 web replicas, got %d", got)
	}
	if got := fake.CountMatching("--name basic-db-"); got != 1 {
		t.Errorf("expected 1 db replica, got %d", got)
	}
}

func TestRemoveOrphans(t *testing.T) {
	proj := load(t, "basic")
	fake := &runner.Fake{}
	// A stale "cache" service container belongs to the project but is gone
	// from the compose file.
	fake.On("list --all --format json", runner.Result{Stdout: `[
		{"name":"basic-cache-1","labels":{"com.docker.compose.project":"basic","com.docker.compose.service":"cache"}},
		{"name":"basic-web-1","labels":{"com.docker.compose.project":"basic","com.docker.compose.service":"web"}},
		{"name":"other-x-1","labels":{"com.docker.compose.project":"other","com.docker.compose.service":"x"}}
	]`}, nil)
	e := New(fake, io.Discard)

	err := e.Up(context.Background(), proj, UpOptions{Detach: true, RemoveOrphans: true})
	if err != nil {
		t.Fatalf("Up: %v", err)
	}
	calls := fake.CommandArgs()
	if firstMatch(calls, "delete basic-cache-1") == -1 {
		t.Errorf("orphan cache container should be deleted:\n%s", strings.Join(calls, "\n"))
	}
	if firstMatch(calls, "delete basic-web-1") != -1 {
		t.Errorf("active web container must NOT be deleted")
	}
	if firstMatch(calls, "delete other-x-1") != -1 {
		t.Errorf("other project's container must NOT be deleted")
	}
}
