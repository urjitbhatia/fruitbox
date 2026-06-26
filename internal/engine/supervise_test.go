package engine

import (
	"context"
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/urjitbhatia/fruitbox/internal/runner"
)

func TestParseRestartPolicy(t *testing.T) {
	cases := map[string]restartPolicy{
		"":               {mode: ""},
		"no":             {mode: "no"},
		"always":         {mode: "always"},
		"unless-stopped": {mode: "unless-stopped"},
		"on-failure":     {mode: "on-failure"},
		"on-failure:5":   {mode: "on-failure", maxRetries: 5},
	}
	for spec, want := range cases {
		got := parseRestartPolicy(types.ServiceConfig{Restart: spec})
		if got != want {
			t.Errorf("parseRestartPolicy(%q) = %+v, want %+v", spec, got, want)
		}
	}
}

func TestRestartPolicyWantsRestart(t *testing.T) {
	always := restartPolicy{mode: "always"}
	if !always.wantsRestart(0, 100) {
		t.Error("always should restart even on clean exit")
	}
	onfail := restartPolicy{mode: "on-failure", maxRetries: 2}
	if onfail.wantsRestart(0, 0) {
		t.Error("on-failure should not restart on exit 0")
	}
	if !onfail.wantsRestart(1, 1) {
		t.Error("on-failure should restart while under max retries")
	}
	if onfail.wantsRestart(1, 2) {
		t.Error("on-failure should stop once max retries reached")
	}
	no := restartPolicy{mode: "no"}
	if no.wantsRestart(1, 0) {
		t.Error("no policy should never restart")
	}
}

func TestSuperviseRestartsOnFailureUntilExhausted(t *testing.T) {
	proj := load(t, "restart")
	fake := &runner.Fake{}
	// The worker is always observed as exited with code 1.
	fake.On("inspect restart-worker-1", runner.Result{Stdout: `[{"status":"exited","exit_code":1}]`}, nil)
	e := newTestEngine(fake)

	if err := e.Supervise(context.Background(), proj, nil); err != nil {
		t.Fatalf("Supervise: %v", err)
	}
	// on-failure:1 means exactly one restart, then give up.
	if got := fake.CountMatching("start restart-worker-1"); got != 1 {
		t.Errorf("expected exactly 1 restart, got %d", got)
	}
}

func TestSuperviseNoPolicyReturnsWhenStopped(t *testing.T) {
	proj := load(t, "basic") // no restart policies
	fake := &runner.Fake{}
	fake.On("inspect", runner.Result{Stdout: `[{"status":"exited","exit_code":0}]`}, nil)
	e := newTestEngine(fake)

	// Should return promptly without restarting anything.
	if err := e.Supervise(context.Background(), proj, nil); err != nil {
		t.Fatalf("Supervise: %v", err)
	}
	if got := fake.CountMatching("start "); got != 0 {
		t.Errorf("expected no restarts for policy-less services, got %d", got)
	}
}
