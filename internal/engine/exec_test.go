package engine

import (
	"context"
	"io"
	"testing"

	"github.com/urjitbhatia/fruitbox/internal/runner"
)

func TestExecIndexAndDetach(t *testing.T) {
	proj := load(t, "basic")
	fake := &runner.Fake{}
	e := New(fake, io.Discard)

	err := e.Exec(context.Background(), proj, "web", []string{"sh", "-c", "echo hi"}, ExecOptions{
		Detach: true,
		Index:  2,
	})
	if err != nil {
		t.Fatalf("Exec: %v", err)
	}
	calls := fake.CommandArgs()
	if firstMatch(calls, "exec --detach basic-web-2 sh -c echo hi") == -1 {
		t.Errorf("exec should detach and target replica 2, calls: %v", calls)
	}
}

func TestExecInteractiveTTY(t *testing.T) {
	proj := load(t, "basic")
	fake := &runner.Fake{}
	e := New(fake, io.Discard)
	err := e.Exec(context.Background(), proj, "web", []string{"bash"}, ExecOptions{
		Interactive: true,
		TTY:         true,
	})
	if err != nil {
		t.Fatalf("Exec: %v", err)
	}
	if firstMatch(fake.CommandArgs(), "exec --interactive --tty basic-web-1 bash") == -1 {
		t.Errorf("exec should be interactive+tty, calls: %v", fake.CommandArgs())
	}
}
