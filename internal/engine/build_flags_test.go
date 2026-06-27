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

func TestBuildWithDependencies(t *testing.T) {
	proj := load(t, "builddeps") // app (build) depends_on lib (build)
	fake := &runner.Fake{}
	e := New(fake, io.Discard)
	if err := e.Build(context.Background(), proj, []string{"app"}, BuildOptions{WithDependencies: true}); err != nil {
		t.Fatalf("Build: %v", err)
	}
	calls := fake.CommandArgs()
	if firstMatch(calls, "build --tag builddeps_app") == -1 {
		t.Errorf("should build app, calls: %v", calls)
	}
	if firstMatch(calls, "build --tag builddeps_lib") == -1 {
		t.Errorf("--with-dependencies should also build lib, calls: %v", calls)
	}
}

func TestBuildWithoutDependenciesSkipsDeps(t *testing.T) {
	proj := load(t, "builddeps")
	fake := &runner.Fake{}
	e := New(fake, io.Discard)
	if err := e.Build(context.Background(), proj, []string{"app"}, BuildOptions{}); err != nil {
		t.Fatalf("Build: %v", err)
	}
	if firstMatch(fake.CommandArgs(), "build --tag builddeps_lib") != -1 {
		t.Errorf("without --with-dependencies, lib should NOT build: %v", fake.CommandArgs())
	}
}

func TestBuildPush(t *testing.T) {
	proj := load(t, "build") // api builds (tag build_api)
	fake := &runner.Fake{}
	e := New(fake, io.Discard)
	if err := e.Build(context.Background(), proj, []string{"api"}, BuildOptions{Push: true}); err != nil {
		t.Fatalf("Build: %v", err)
	}
	calls := fake.CommandArgs()
	posBuild := firstMatch(calls, "build --tag build_api")
	posPush := firstMatch(calls, "image push build_api")
	if posBuild == -1 || posPush == -1 || posBuild > posPush {
		t.Errorf("--push should push after build, calls: %v", calls)
	}
}
