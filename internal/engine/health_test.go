package engine

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/urjitbhatia/fruitbox/internal/runner"
)

// newTestEngine returns an engine whose clock is virtual: sleeps advance a
// fake clock instead of blocking, so health polling loops run instantly.
func newTestEngine(fake *runner.Fake) *Engine {
	now := time.Unix(0, 0)
	e := New(fake, io.Discard)
	e.Now = func() time.Time { return now }
	e.Sleep = func(_ context.Context, d time.Duration) error {
		now = now.Add(d)
		return nil
	}
	return e
}

func TestWaitHealthyRetriesThenSucceeds(t *testing.T) {
	fake := &runner.Fake{}
	// The healthcheck exec fails twice, then passes.
	fake.OnSequence("exec db-1 pg_isready",
		runner.Result{ExitCode: 1},
		runner.Result{ExitCode: 1},
		runner.Result{ExitCode: 0},
	)
	e := newTestEngine(fake)

	proj := load(t, "health")
	dbSvc, _ := proj.GetService("db")
	// Override the project name's effect: container name is health-db-1.
	if err := e.waitHealthy(context.Background(), "db-1", dbSvc.HealthCheck); err != nil {
		t.Fatalf("waitHealthy: %v", err)
	}
	if got := fake.CountMatching("exec db-1 pg_isready"); got != 3 {
		t.Errorf("expected 3 healthcheck probes, got %d", got)
	}
}

func TestWaitHealthyFailsAfterRetries(t *testing.T) {
	fake := &runner.Fake{}
	fake.On("exec db-1 pg_isready", runner.Result{ExitCode: 1}, nil)
	e := newTestEngine(fake)

	proj := load(t, "health")
	dbSvc, _ := proj.GetService("db")
	err := e.waitHealthy(context.Background(), "db-1", dbSvc.HealthCheck)
	if err == nil {
		t.Fatal("expected health failure after retries, got nil")
	}
	// retries: 5 in the fixture.
	if got := fake.CountMatching("exec db-1 pg_isready"); got != 5 {
		t.Errorf("expected 5 probes before giving up, got %d", got)
	}
}

func TestUpWaitsForHealthyDependencyBeforeStartingDependent(t *testing.T) {
	fake := &runner.Fake{}
	// db healthcheck: unhealthy once, then healthy.
	fake.OnSequence("exec health-db-1 pg_isready",
		runner.Result{ExitCode: 1},
		runner.Result{ExitCode: 0},
	)
	// migrate completes successfully immediately when inspected.
	fake.On("inspect health-migrate-1", runner.Result{
		Stdout: `[{"status":"stopped","exit_code":0}]`,
	}, nil)
	e := newTestEngine(fake)

	proj := load(t, "health")
	if err := e.Up(context.Background(), proj, UpOptions{Detach: true}); err != nil {
		t.Fatalf("Up: %v", err)
	}

	calls := fake.CommandArgs()
	posDBRun := firstMatch(calls, "--name health-db-1")
	posHealth := firstMatch(calls, "exec health-db-1 pg_isready")
	posMigrateRun := firstMatch(calls, "--name health-migrate-1")
	posInspect := firstMatch(calls, "inspect health-migrate-1")
	posWebRun := firstMatch(calls, "--name health-web-1")

	for label, pos := range map[string]int{
		"db run": posDBRun, "db health probe": posHealth,
		"migrate run": posMigrateRun, "migrate inspect": posInspect, "web run": posWebRun,
	} {
		if pos == -1 {
			t.Fatalf("missing call %q:\n%s", label, strings.Join(calls, "\n"))
		}
	}
	// db must be started and pass health before migrate starts.
	if !(posDBRun < posHealth && posHealth < posMigrateRun) {
		t.Errorf("ordering wrong: dbRun@%d health@%d migrateRun@%d", posDBRun, posHealth, posMigrateRun)
	}
	// migrate must complete (inspect) before web starts.
	if !(posMigrateRun < posInspect && posInspect < posWebRun) {
		t.Errorf("ordering wrong: migrateRun@%d inspect@%d webRun@%d", posMigrateRun, posInspect, posWebRun)
	}
}

func TestWaitCompletedNonZeroExitErrors(t *testing.T) {
	fake := &runner.Fake{}
	fake.On("inspect job-1", runner.Result{Stdout: `[{"status":"exited","exit_code":2}]`}, nil)
	e := newTestEngine(fake)
	err := e.waitCompleted(context.Background(), "job-1")
	if err == nil || !strings.Contains(err.Error(), "code 2") {
		t.Fatalf("expected non-zero exit error, got %v", err)
	}
}
