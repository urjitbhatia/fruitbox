package engine

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/urjitbhatia/fruitbox/internal/runner"
)

func TestBuildCLIOverrides(t *testing.T) {
	proj := load(t, "build")
	fake := &runner.Fake{}
	e := New(fake, io.Discard)
	err := e.Build(context.Background(), proj, []string{"api"}, BuildOptions{
		BuildArgs: []string{"TOKEN=abc"},
		NoCache:   true,
		Pull:      true,
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	var buildCall string
	for _, c := range fake.CommandArgs() {
		if strings.HasPrefix(c, "build ") {
			buildCall = c
		}
	}
	for _, want := range []string{"--no-cache", "--pull", "--build-arg TOKEN=abc"} {
		if !strings.Contains(buildCall, want) {
			t.Errorf("build call missing %q: %s", want, buildCall)
		}
	}
}

func TestBuildQuietMemory(t *testing.T) {
	proj := load(t, "build")
	fake := &runner.Fake{}
	e := New(fake, io.Discard)
	if err := e.Build(context.Background(), proj, []string{"api"}, BuildOptions{Quiet: true, Memory: "512m"}); err != nil {
		t.Fatalf("Build: %v", err)
	}
	var buildCall string
	for _, c := range fake.CommandArgs() {
		if strings.HasPrefix(c, "build ") {
			buildCall = c
		}
	}
	if !strings.Contains(buildCall, "--quiet") || !strings.Contains(buildCall, "--memory 512m") {
		t.Errorf("build should carry --quiet --memory: %s", buildCall)
	}
}
