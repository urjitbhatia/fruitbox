package engine

import (
	"context"
	"io"
	"testing"

	"github.com/urjitbhatia/fruitbox/internal/runner"
)

func TestStopTimeout(t *testing.T) {
	proj := load(t, "basic")
	fake := &runner.Fake{}
	e := New(fake, io.Discard)
	seven := 7
	if err := e.Stop(context.Background(), proj, []string{"web"}, &seven); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if firstMatch(fake.CommandArgs(), "stop --time 7 basic-web-1") == -1 {
		t.Errorf("stop should carry --time 7, calls: %v", fake.CommandArgs())
	}
}

func TestRestartTimeout(t *testing.T) {
	proj := load(t, "basic")
	fake := &runner.Fake{}
	e := New(fake, io.Discard)
	three := 3
	if err := e.Restart(context.Background(), proj, []string{"web"}, &three); err != nil {
		t.Fatalf("Restart: %v", err)
	}
	calls := fake.CommandArgs()
	if firstMatch(calls, "stop --time 3 basic-web-1") == -1 {
		t.Errorf("restart should stop with --time 3, calls: %v", calls)
	}
	if firstMatch(calls, "start basic-web-1") == -1 {
		t.Errorf("restart should start again, calls: %v", calls)
	}
}

func TestKillRemoveOrphans(t *testing.T) {
	proj := load(t, "basic")
	fake := &runner.Fake{}
	fake.On("list --all --format json", runner.Result{Stdout: `[
		{"name":"basic-ghost-1","labels":{"com.docker.compose.project":"basic","com.docker.compose.service":"ghost"}}
	]`}, nil)
	e := New(fake, io.Discard)
	if err := e.Kill(context.Background(), proj, nil, "SIGKILL", true); err != nil {
		t.Fatalf("Kill: %v", err)
	}
	calls := fake.CommandArgs()
	if firstMatch(calls, "kill --signal SIGKILL basic-web-1") == -1 {
		t.Errorf("kill should signal, calls: %v", calls)
	}
	if firstMatch(calls, "delete basic-ghost-1") == -1 {
		t.Errorf("kill --remove-orphans should remove orphan, calls: %v", calls)
	}
}
