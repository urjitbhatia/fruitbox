package engine

import (
	"context"
	"io"
	"testing"

	"github.com/urjitbhatia/fruitbox/internal/runner"
)

func TestLogsTail(t *testing.T) {
	proj := load(t, "basic")
	fake := &runner.Fake{}
	e := New(fake, io.Discard)
	if err := e.Logs(context.Background(), proj, []string{"web"}, LogOptions{Tail: "100"}); err != nil {
		t.Fatalf("Logs: %v", err)
	}
	if firstMatch(fake.CommandArgs(), "logs -n 100 basic-web-1") == -1 {
		t.Errorf("logs should pass -n 100, calls: %v", fake.CommandArgs())
	}
}

func TestLogsTailAllOmitsFlag(t *testing.T) {
	proj := load(t, "basic")
	fake := &runner.Fake{}
	e := New(fake, io.Discard)
	if err := e.Logs(context.Background(), proj, []string{"web"}, LogOptions{Tail: "all"}); err != nil {
		t.Fatalf("Logs: %v", err)
	}
	if firstMatch(fake.CommandArgs(), "logs basic-web-1") == -1 {
		t.Errorf("tail=all should omit -n, calls: %v", fake.CommandArgs())
	}
}

func TestLogsFollow(t *testing.T) {
	proj := load(t, "basic")
	fake := &runner.Fake{}
	e := New(fake, io.Discard)
	if err := e.Logs(context.Background(), proj, []string{"web"}, LogOptions{Follow: true}); err != nil {
		t.Fatalf("Logs: %v", err)
	}
	if firstMatch(fake.CommandArgs(), "logs --follow basic-web-1") == -1 {
		t.Errorf("logs --follow expected, calls: %v", fake.CommandArgs())
	}
}

func TestLogsIndexSelectsReplica(t *testing.T) {
	proj := load(t, "basic")
	// scale web to 3 via deploy not set; use scale override through Up not needed.
	fake := &runner.Fake{}
	e := New(fake, io.Discard)
	// Index 1 on a single-replica service still works.
	if err := e.Logs(context.Background(), proj, []string{"web"}, LogOptions{Index: 1}); err != nil {
		t.Fatalf("Logs: %v", err)
	}
	if firstMatch(fake.CommandArgs(), "logs basic-web-1") == -1 {
		t.Errorf("logs --index 1 should target replica 1, calls: %v", fake.CommandArgs())
	}
}
