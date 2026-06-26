package engine

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/urjitbhatia/fruitbox/internal/runner"
)

func TestDownTimeoutOverridesStopTime(t *testing.T) {
	proj := load(t, "basic")
	fake := &runner.Fake{}
	e := New(fake, io.Discard)

	five := 5
	if err := e.Down(context.Background(), proj, DownOptions{Timeout: &five}); err != nil {
		t.Fatalf("Down: %v", err)
	}
	calls := fake.CommandArgs()
	// Every stop should carry the override timeout.
	for _, c := range calls {
		if strings.HasPrefix(c, "stop ") && !strings.Contains(c, "--time 5") {
			t.Errorf("stop without override timeout: %q", c)
		}
	}
	if firstMatch(calls, "stop --time 5 basic-web-1") == -1 {
		t.Errorf("expected timeout-overridden stop, calls: %v", calls)
	}
}

func TestDownRemoveOrphans(t *testing.T) {
	proj := load(t, "basic")
	fake := &runner.Fake{}
	fake.On("list --all --format json", runner.Result{Stdout: `[
		{"name":"basic-ghost-1","labels":{"com.docker.compose.project":"basic","com.docker.compose.service":"ghost"}}
	]`}, nil)
	e := New(fake, io.Discard)

	if err := e.Down(context.Background(), proj, DownOptions{RemoveOrphans: true}); err != nil {
		t.Fatalf("Down: %v", err)
	}
	if firstMatch(fake.CommandArgs(), "delete basic-ghost-1") == -1 {
		t.Errorf("orphan should be removed on down, calls: %v", fake.CommandArgs())
	}
}

func TestDownRemoveImagesAll(t *testing.T) {
	proj := load(t, "basic")
	fake := &runner.Fake{}
	e := New(fake, io.Discard)

	if err := e.Down(context.Background(), proj, DownOptions{RemoveImages: "all"}); err != nil {
		t.Fatalf("Down: %v", err)
	}
	calls := fake.CommandArgs()
	if firstMatch(calls, "image delete nginx:1.27") == -1 || firstMatch(calls, "image delete postgres:16") == -1 {
		t.Errorf("--rmi all should delete all service images, calls: %v", calls)
	}
}

func TestDownRemoveImagesLocalOnly(t *testing.T) {
	// In the build fixture, `api` builds with no explicit image (local), while
	// `worker` has an explicit image (not local).
	proj := load(t, "build")
	fake := &runner.Fake{}
	e := New(fake, io.Discard)

	if err := e.Down(context.Background(), proj, DownOptions{RemoveImages: "local"}); err != nil {
		t.Fatalf("Down: %v", err)
	}
	calls := fake.CommandArgs()
	if firstMatch(calls, "image delete build_api") == -1 {
		t.Errorf("--rmi local should delete the locally-built image, calls: %v", calls)
	}
	if firstMatch(calls, "image delete myorg/worker:latest") != -1 {
		t.Errorf("--rmi local must NOT delete explicitly-named images, calls: %v", calls)
	}
}
