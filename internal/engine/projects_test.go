package engine

import (
	"context"
	"io"
	"testing"

	"github.com/urjitbhatia/fruitbox/internal/runner"
)

func TestListProjectsGroupsByLabel(t *testing.T) {
	fake := &runner.Fake{}
	fake.On("list --all --format json", runner.Result{Stdout: `[
		{"name":"a-web-1","labels":{"com.docker.compose.project":"a","com.docker.compose.service":"web"}},
		{"name":"a-db-1","labels":{"com.docker.compose.project":"a","com.docker.compose.service":"db"}},
		{"name":"b-x-1","labels":{"com.docker.compose.project":"b","com.docker.compose.service":"x"}},
		{"name":"loose-1","labels":{}}
	]`}, nil)
	e := New(fake, io.Discard)

	projects, err := e.ListProjects(context.Background())
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if len(projects) != 2 {
		t.Fatalf("expected 2 projects, got %d: %+v", len(projects), projects)
	}
	if projects[0].Name != "a" || projects[0].ContainerCount != 2 {
		t.Errorf("project a = %+v, want {a 2}", projects[0])
	}
	if projects[1].Name != "b" || projects[1].ContainerCount != 1 {
		t.Errorf("project b = %+v, want {b 1}", projects[1])
	}
}

func TestWaitReturnsExitCode(t *testing.T) {
	proj := load(t, "basic")
	fake := &runner.Fake{}
	// db then web both inspected as exited; web exits 0.
	fake.On("inspect basic-db-1", runner.Result{Stdout: `[{"status":"exited","exit_code":0}]`}, nil)
	fake.On("inspect basic-web-1", runner.Result{Stdout: `[{"status":"exited","exit_code":0}]`}, nil)
	e := newTestEngine(fake)

	code, err := e.Wait(context.Background(), proj, nil)
	if err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if code != 0 {
		t.Errorf("Wait code = %d, want 0", code)
	}
}
