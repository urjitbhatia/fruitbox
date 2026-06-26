package engine

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/urjitbhatia/fruitbox/internal/runner"
)

func TestDiffEvents(t *testing.T) {
	prev := containerState{"a": "running", "b": "running"}
	cur := containerState{"a": "stopped", "c": "running"}
	got := diffEvents(prev, cur)

	want := []Event{
		{"a", "die"},     // a: running -> stopped
		{"b", "destroy"}, // b: gone
		{"c", "create"},  // c: new...
		{"c", "start"},   // ...and running
	}
	if len(got) != len(want) {
		t.Fatalf("got %d events, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("event[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestEventsStreamsTransitions(t *testing.T) {
	proj := load(t, "basic")
	fake := &runner.Fake{}
	// Poll 1: db running only. Poll 2: web also running.
	fake.OnSequence("list --all --format json",
		runner.Result{Stdout: `[{"name":"basic-db-1","status":"running","labels":{"com.docker.compose.project":"basic","com.docker.compose.service":"db"}}]`},
		runner.Result{Stdout: `[
			{"name":"basic-db-1","status":"running","labels":{"com.docker.compose.project":"basic","com.docker.compose.service":"db"}},
			{"name":"basic-web-1","status":"running","labels":{"com.docker.compose.project":"basic","com.docker.compose.service":"web"}}
		]`},
	)
	var out bytes.Buffer
	e := newTestEngine(fake)
	e.Out = &out

	if err := e.Events(context.Background(), proj, 2); err != nil {
		t.Fatalf("Events: %v", err)
	}
	s := out.String()
	if !strings.Contains(s, "container create basic-web-1") || !strings.Contains(s, "container start basic-web-1") {
		t.Errorf("expected web create/start events, got:\n%s", s)
	}
}
