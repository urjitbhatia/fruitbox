package engine

import (
	"context"
	"io"
	"testing"

	"github.com/urjitbhatia/fruitbox/internal/runner"
)

func TestOrphans(t *testing.T) {
	proj := load(t, "basic")
	fake := &runner.Fake{}
	fake.On("list --all --format json", runner.Result{Stdout: `[
		{"name":"basic-web-1","status":"running","labels":{"com.docker.compose.project":"basic","com.docker.compose.service":"web"}},
		{"name":"basic-ghost-1","status":"running","labels":{"com.docker.compose.project":"basic","com.docker.compose.service":"ghost"}}
	]`}, nil)
	e := New(fake, io.Discard)

	got := e.Orphans(context.Background(), proj)
	if len(got) != 1 || got[0].Service != "ghost" || got[0].Name != "basic-ghost-1" {
		t.Errorf("Orphans should return only the ghost service, got %+v", got)
	}
}
