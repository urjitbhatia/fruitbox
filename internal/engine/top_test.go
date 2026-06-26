package engine

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/urjitbhatia/fruitbox/internal/runner"
)

func TestTopExecsPs(t *testing.T) {
	proj := load(t, "basic")
	fake := &runner.Fake{}
	fake.On("exec basic-web-1 ps", runner.Result{Stdout: "PID CMD\n1 nginx\n"}, nil)
	fake.On("exec basic-db-1 ps", runner.Result{Stdout: "PID CMD\n1 postgres\n"}, nil)
	var out bytes.Buffer
	e := New(fake, &out)

	if err := e.Top(context.Background(), proj, nil, nil); err != nil {
		t.Fatalf("Top: %v", err)
	}
	calls := fake.CommandArgs()
	if firstMatch(calls, "exec basic-web-1 ps -ef") == -1 {
		t.Errorf("top should exec ps -ef in web, calls: %v", calls)
	}
	if !strings.Contains(out.String(), "nginx") || !strings.Contains(out.String(), "postgres") {
		t.Errorf("top output missing process info:\n%s", out.String())
	}
}

func TestPauseUnpauseSendSignals(t *testing.T) {
	proj := load(t, "basic")
	fake := &runner.Fake{}
	e := New(fake, nil)

	if err := e.Pause(context.Background(), proj, []string{"web"}); err != nil {
		t.Fatalf("Pause: %v", err)
	}
	if err := e.Unpause(context.Background(), proj, []string{"web"}); err != nil {
		t.Fatalf("Unpause: %v", err)
	}
	calls := fake.CommandArgs()
	if firstMatch(calls, "kill --signal SIGSTOP basic-web-1") == -1 {
		t.Errorf("pause should SIGSTOP, calls: %v", calls)
	}
	if firstMatch(calls, "kill --signal SIGCONT basic-web-1") == -1 {
		t.Errorf("unpause should SIGCONT, calls: %v", calls)
	}
}
